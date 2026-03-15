package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"go-pathprobe/pkg/cli"
	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/logging"
	"go-pathprobe/pkg/netprobe"
	"go-pathprobe/pkg/syscheck"
)

func main() {
	logger, levelVar := logging.NewLogger(slog.LevelInfo)

	// Detect raw ICMP socket availability and warn the user when unavailable.
	avail := syscheck.RawICMPChecker{}.Check()
	if !avail.Available {
		logger.Warn(avail.Notice(), "error", avail.Err)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	dispatcher := diag.NewDispatcher(nil)

	// Shared prober for connectivity.
	tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
	connectRunner := diag.NewConnectivityRunner(tcpProber, logger)

	// Select traceroute prober: prefer ICMP when raw sockets are available,
	// fall back to TCP (privilege-free) otherwise.
	webTracerouteRunner := diag.NewWebTracerouteRunner(diag.NewTracerouteRunner(diag.SelectTracerouteProber(avail.Available), logger))

	// Register web runner: STUN → HTTPS echo fallback for public IP, plus DNS compare and HTTP probe.
	webFetcher := &netprobe.MultiSourcePublicIPFetcher{
		Sources: []netprobe.PublicIPFetcher{
			&netprobe.STUNPublicIPFetcher{Server: "stun.l.google.com:19302", Source: "stun-google"},
			&netprobe.HTTPPublicIPFetcher{Client: httpClient, URL: "https://api.ipify.org", Source: "https-echo"},
		},
	}
	webComparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{
		&netprobe.SystemResolver{Name: "system"},
		&netprobe.HTTPDNSResolver{Client: httpClient, Endpoint: "https://cloudflare-dns.com/dns-query", Name: "doh-1.1.1.1"},
		&netprobe.HTTPDNSResolver{Client: httpClient, Endpoint: "https://dns.google/resolve", Name: "doh-8.8.8.8"},
	}}
	httpProber := &netprobe.ClientHTTPProber{Client: httpClient}
	dispatcher.Register(diag.TargetWeb, diag.NewMultiRunner(diag.NewWebRunner(webFetcher, webComparator, logger), diag.NewHTTPRunner(httpProber, logger), diag.NewWebPortRunner(connectRunner), webTracerouteRunner))

	// SMTP runner with MX resolution and connectivity.
	smtpProber := &netprobe.DialSMTPProber{}
	dispatcher.Register(diag.TargetSMTP, diag.NewMultiRunner(connectRunner, diag.NewSMTPRunner(smtpProber, &netprobe.SystemResolver{Name: "system"}, logger)))

	// FTP/FTPS runner.
	ftpProber := &netprobe.DialFTPProber{}
	dispatcher.Register(diag.TargetFTP, diag.NewMultiRunner(connectRunner, diag.NewFTPRunner(ftpProber, logger)))

	// SFTP runner.
	sftpProber := &netprobe.DialSFTPProber{}
	dispatcher.Register(diag.TargetSFTP, diag.NewMultiRunner(connectRunner, diag.NewSFTPRunner(sftpProber, logger)))

	// Connectivity for remaining targets.
	for _, target := range diag.AllTargets {
		if target == diag.TargetWeb || target == diag.TargetSMTP || target == diag.TargetFTP || target == diag.TargetSFTP {
			continue
		}
		dispatcher.Register(target, connectRunner)
	}

	rootCmd := cli.NewRootCommand(dispatcher, logger, levelVar)
	if err := rootCmd.Execute(); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}
