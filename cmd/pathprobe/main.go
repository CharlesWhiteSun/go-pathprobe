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
)

func main() {
	logger, levelVar := logging.NewLogger(slog.LevelInfo)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	dispatcher := diag.NewDispatcher(nil)

	// Shared prober for connectivity.
	tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
	connectRunner := diag.NewConnectivityRunner(tcpProber, logger)

	// Register web runner with DoH resolvers and HTTPS echo for public IP, combined with connectivity.
	webFetcher := &netprobe.HTTPPublicIPFetcher{Client: httpClient, URL: "https://api.ipify.org", Source: "https-echo"}
	webComparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{
		&netprobe.SystemResolver{Name: "system"},
		&netprobe.HTTPDNSResolver{Client: httpClient, Endpoint: "https://cloudflare-dns.com/dns-query", Name: "doh-1.1.1.1"},
		&netprobe.HTTPDNSResolver{Client: httpClient, Endpoint: "https://dns.google/resolve", Name: "doh-8.8.8.8"},
	}}
	dispatcher.Register(diag.TargetWeb, diag.NewMultiRunner(diag.NewWebRunner(webFetcher, webComparator, logger), connectRunner))

	// Connectivity for remaining targets.
	for _, target := range diag.AllTargets {
		if target == diag.TargetWeb {
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
