package app

import (
	"fmt"
	"time"

	"go-pathprobe/pkg/cli"
	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/netprobe"
	"go-pathprobe/pkg/server"
)

// AllPlugins is the canonical list of all registered diagnostic protocols.
// To add a new protocol: create a new ProtocolPlugin variable and append it here.
// No other code needs to change (Open/Closed Principle).
var AllPlugins = []ProtocolPlugin{
	WebPlugin,
	SMTPPlugin,
	FTPPlugin,
	SFTPPlugin,
	IMAPPlugin,
	POPPlugin,
}

// WebPlugin registers the web diagnostic target (public-IP discovery, DNS
// comparison across resolvers, HTTP probe, port reachability, and traceroute).
var WebPlugin = ProtocolPlugin{
	Target: diag.TargetWeb,
	NewRunner: func(deps Deps) diag.Runner {
		tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
		connectRunner := diag.NewConnectivityRunner(tcpProber, deps.Logger)

		webFetcher := &netprobe.MultiSourcePublicIPFetcher{
			Sources: []netprobe.PublicIPFetcher{
				&netprobe.STUNPublicIPFetcher{
					Server: "stun.l.google.com:19302",
					Source: "stun-google",
				},
				&netprobe.HTTPPublicIPFetcher{
					Client: deps.HTTP,
					URL:    "https://api.ipify.org",
					Source: "https-echo",
				},
			},
		}
		webComparator := netprobe.DNSComparator{Resolvers: []netprobe.DNSResolver{
			&netprobe.SystemResolver{Name: "system"},
			&netprobe.HTTPDNSResolver{
				Client:   deps.HTTP,
				Endpoint: "https://cloudflare-dns.com/dns-query",
				Name:     "doh-1.1.1.1",
			},
			&netprobe.HTTPDNSResolver{
				Client:   deps.HTTP,
				Endpoint: "https://dns.google/resolve",
				Name:     "doh-8.8.8.8",
			},
		}}
		httpProber := &netprobe.ClientHTTPProber{Client: deps.HTTP}
		tracerouteRunner := diag.NewTracerouteRunner(
			diag.SelectTracerouteProber(deps.ICMPAvail), deps.Logger,
		)
		webTracerouteRunner := diag.NewWebTracerouteRunner(tracerouteRunner)

		return diag.NewMultiRunner(
			diag.NewWebRunner(webFetcher, webComparator, deps.Logger),
			diag.NewHTTPRunner(httpProber, deps.Logger),
			diag.NewWebPortRunner(connectRunner),
			webTracerouteRunner,
		)
	},
	RegisterCLI: cli.RegisterWebFlags,
	BuildOptions: func(req server.ReqOptions) (diag.Options, error) {
		opts := server.BuildGlobalOptions(req)
		if n := req.Net; n != nil {
			opts.Net.Host = n.Host
			opts.Net.Ports = n.Ports
		}
		if w := req.Web; w != nil {
			webMode := diag.WebMode(w.Mode)
			if !diag.IsValidWebMode(webMode) {
				return diag.Options{}, fmt.Errorf("web.mode: unknown mode %q", w.Mode)
			}
			opts.Web.Mode = webMode
			opts.Web.Domains = w.Domains
			opts.Web.URL = w.URL
			opts.Web.MaxHops = w.MaxHops
			if len(w.Types) > 0 {
				types, err := netprobe.ParseRecordTypes(w.Types)
				if err != nil {
					return diag.Options{}, fmt.Errorf("web.types: %w", err)
				}
				opts.Web.Types = types
			}
		}
		return opts, nil
	},
}

// SMTPPlugin registers the SMTP diagnostic target (connectivity + protocol handshake).
var SMTPPlugin = ProtocolPlugin{
	Target: diag.TargetSMTP,
	NewRunner: func(deps Deps) diag.Runner {
		tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
		connectRunner := diag.NewConnectivityRunner(tcpProber, deps.Logger)
		smtpProber := &netprobe.DialSMTPProber{}
		return diag.NewMultiRunner(
			connectRunner,
			diag.NewSMTPRunner(smtpProber, &netprobe.SystemResolver{Name: "system"}, deps.Logger),
		)
	},
	RegisterCLI: cli.RegisterSMTPFlags,
	BuildOptions: func(req server.ReqOptions) (diag.Options, error) {
		opts := server.BuildGlobalOptions(req)
		if n := req.Net; n != nil {
			opts.Net.Host = n.Host
			opts.Net.Ports = n.Ports
		}
		if s := req.SMTP; s != nil {
			smtpMode := diag.SMTPMode(s.Mode)
			if !diag.IsValidSMTPMode(smtpMode) {
				return diag.Options{}, fmt.Errorf("smtp.mode: unknown mode %q", s.Mode)
			}
			opts.SMTP = diag.SMTPOptions{
				Mode:        smtpMode,
				Domain:      s.Domain,
				Username:    s.Username,
				Password:    s.Password,
				From:        s.From,
				To:          s.To,
				UseTLS:      s.UseTLS,
				StartTLS:    s.StartTLS,
				AuthMethods: s.AuthMethods,
				MXProbeAll:  s.MXProbeAll,
			}
		}
		return opts, nil
	},
}

// FTPPlugin registers the FTP/FTPS diagnostic target.
var FTPPlugin = ProtocolPlugin{
	Target: diag.TargetFTP,
	NewRunner: func(deps Deps) diag.Runner {
		tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
		connectRunner := diag.NewConnectivityRunner(tcpProber, deps.Logger)
		ftpProber := &netprobe.DialFTPProber{}
		return diag.NewMultiRunner(
			connectRunner,
			diag.NewFTPRunner(ftpProber, deps.Logger),
		)
	},
	RegisterCLI: cli.RegisterFTPFlags,
	BuildOptions: func(req server.ReqOptions) (diag.Options, error) {
		opts := server.BuildGlobalOptions(req)
		if n := req.Net; n != nil {
			opts.Net.Host = n.Host
			opts.Net.Ports = n.Ports
		}
		if f := req.FTP; f != nil {
			ftpMode := diag.FTPMode(f.Mode)
			if !diag.IsValidFTPMode(ftpMode) {
				return diag.Options{}, fmt.Errorf("ftp.mode: unknown mode %q", f.Mode)
			}
			opts.FTP = diag.FTPOptions{
				Mode:     ftpMode,
				Username: f.Username,
				Password: f.Password,
				UseTLS:   f.UseTLS,
				AuthTLS:  f.AuthTLS,
				RunLIST:  f.RunLIST,
			}
		}
		return opts, nil
	},
}

// SFTPPlugin registers the SFTP/SSH diagnostic target.
var SFTPPlugin = ProtocolPlugin{
	Target: diag.TargetSFTP,
	NewRunner: func(deps Deps) diag.Runner {
		tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
		connectRunner := diag.NewConnectivityRunner(tcpProber, deps.Logger)
		sftpProber := &netprobe.DialSFTPProber{}
		return diag.NewMultiRunner(
			connectRunner,
			diag.NewSFTPRunner(sftpProber, deps.Logger),
		)
	},
	RegisterCLI: cli.RegisterSFTPFlags,
	BuildOptions: func(req server.ReqOptions) (diag.Options, error) {
		opts := server.BuildGlobalOptions(req)
		if n := req.Net; n != nil {
			opts.Net.Host = n.Host
			opts.Net.Ports = n.Ports
		}
		if sf := req.SFTP; sf != nil {
			sftpMode := diag.SFTPMode(sf.Mode)
			if !diag.IsValidSFTPMode(sftpMode) {
				return diag.Options{}, fmt.Errorf("sftp.mode: unknown mode %q", sf.Mode)
			}
			opts.SFTP = diag.SFTPOptions{
				Mode:     sftpMode,
				Username: sf.Username,
				Password: sf.Password,
				RunLS:    sf.RunLS,
				// PrivateKey is intentionally not exposed via the HTTP API.
			}
		}
		return opts, nil
	},
}

// IMAPPlugin registers the IMAP diagnostic target (connectivity only).
// No protocol-specific CLI flags or option mapping are needed; the shared
// --target-host / --port flags and global options are sufficient.
var IMAPPlugin = ProtocolPlugin{
	Target: diag.TargetIMAP,
	NewRunner: func(deps Deps) diag.Runner {
		tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
		return diag.NewConnectivityRunner(tcpProber, deps.Logger)
	},
}

// POPPlugin registers the POP3 diagnostic target (connectivity only).
var POPPlugin = ProtocolPlugin{
	Target: diag.TargetPOP,
	NewRunner: func(deps Deps) diag.Runner {
		tcpProber := &netprobe.TCPPortProber{Timeout: 2 * time.Second}
		return diag.NewConnectivityRunner(tcpProber, deps.Logger)
	},
}
