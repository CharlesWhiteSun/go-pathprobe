package diag

import (
	"context"
	"log/slog"
	"strings"

	"go-pathprobe/pkg/netprobe"
)

// SMTPMode controls which part of the SMTP flow the runner executes.
// An empty value retains legacy behaviour (full flow based on provided options).
type SMTPMode string

const (
	SMTPModeAll       SMTPMode = ""          // legacy: full flow determined by options
	SMTPModeHandshake SMTPMode = "handshake" // TCP connect + EHLO banner only
	SMTPModeAuth      SMTPMode = "auth"      // EHLO + authentication
	SMTPModeSend      SMTPMode = "send"      // full send flow (EHLO + AUTH + MAIL FROM + RCPT TO)
)

// IsValidSMTPMode reports whether m is a recognised SMTP sub-mode value.
func IsValidSMTPMode(m SMTPMode) bool {
	switch m {
	case SMTPModeAll, SMTPModeHandshake, SMTPModeAuth, SMTPModeSend:
		return true
	}
	return false
}

// SMTPRunner performs SMTP handshake and mail/rcpt probes.
type SMTPRunner struct {
	Prober     netprobe.SMTPProber
	MXResolver netprobe.DNSResolver
	Logger     *slog.Logger
}

// NewSMTPRunner constructs an SMTP runner.
func NewSMTPRunner(prober netprobe.SMTPProber, resolver netprobe.DNSResolver, logger *slog.Logger) *SMTPRunner {
	return &SMTPRunner{Prober: prober, MXResolver: resolver, Logger: logger}
}

// Run resolves MX (if needed) and probes each SMTP host.
// The request mode controls which parts of the SMTP flow are exercised:
//   - SMTPModeHandshake: strips credentials, from, to — only EHLO banner is tested.
//   - SMTPModeAuth: strips from/to — EHLO + authentication.
//   - SMTPModeSend (or legacy ""): full flow as specified by options.
func (r *SMTPRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}
	opts := req.Options.SMTP

	// Apply mode overrides: remove fields irrelevant for the chosen sub-mode.
	switch opts.Mode {
	case SMTPModeHandshake:
		opts.Username = ""
		opts.Password = ""
		opts.From = ""
		opts.To = nil
		opts.MXProbeAll = false
	case SMTPModeAuth:
		opts.From = ""
		opts.To = nil
		opts.MXProbeAll = false
		// SMTPModeAll and SMTPModeSend pass through as-is.
	}

	var hosts []string
	if req.Options.Net.Host != "" {
		hosts = []string{req.Options.Net.Host}
	} else if opts.MXProbeAll {
		hosts = resolveMXList(ctx, r.MXResolver, opts.Domain)
	} else {
		if h := resolveMXFallback(ctx, r.MXResolver, opts.Domain); h != "" {
			hosts = []string{h}
		}
	}
	if len(hosts) == 0 {
		hosts = []string{"localhost"}
	}

	port := choosePort(req.Options.Net.Ports, TargetSMTP)

	req.Emitf("smtp", "Probing %d SMTP host(s) on port %d …", len(hosts), port)

	for _, host := range hosts {
		req.Emitf("smtp", "Connecting to %s:%d …", host, port)
		res, err := r.Prober.Probe(ctx, netprobe.SMTPProbeRequest{
			Host:        host,
			Port:        port,
			Domain:      opts.Domain,
			Username:    opts.Username,
			Password:    opts.Password,
			From:        opts.From,
			To:          opts.To,
			UseTLS:      opts.UseTLS,
			StartTLS:    opts.StartTLS,
			Timeout:     req.Options.Global.Timeout,
			HelloName:   opts.Domain,
			AuthMethods: opts.AuthMethods,
		})
		if err != nil {
			return err
		}
		req.Emitf("smtp_result", "%s:%d: banner=%q starttls=%v rcpt=%v",
			host, port, res.Banner, res.UsedStartTLS, res.RcptAccepted)
		r.Logger.Info("smtp probe", "host", host, "port", port, "banner", res.Banner, "starttls", res.UsedStartTLS, "rcpt", res.RcptAccepted)
		summary := res.Banner
		if res.UsedStartTLS {
			summary += " [STARTTLS]"
		}
		req.Report.AddProto(ProtoResult{
			Protocol: "smtp",
			Host:     host,
			Port:     port,
			OK:       true,
			Summary:  summary,
		})
	}
	return nil
}

func resolveMXFallback(ctx context.Context, resolver netprobe.DNSResolver, domain string) string {
	if resolver == nil || strings.TrimSpace(domain) == "" {
		return ""
	}
	ans, err := resolver.Lookup(ctx, domain, netprobe.RecordTypeMX)
	if err != nil || len(ans.Values) == 0 {
		return ""
	}
	// Values in stringifyMX format host:pref; take first host part
	parts := strings.Split(ans.Values[0], ":")
	return parts[0]
}

// resolveMXList returns all MX host names for a domain, preserving DNS-returned order.
func resolveMXList(ctx context.Context, resolver netprobe.DNSResolver, domain string) []string {
	if resolver == nil || strings.TrimSpace(domain) == "" {
		return nil
	}
	ans, err := resolver.Lookup(ctx, domain, netprobe.RecordTypeMX)
	if err != nil || len(ans.Values) == 0 {
		return nil
	}
	hosts := make([]string, 0, len(ans.Values))
	for _, v := range ans.Values {
		parts := strings.Split(v, ":")
		if len(parts) > 0 && parts[0] != "" {
			hosts = append(hosts, parts[0])
		}
	}
	return hosts
}

func choosePort(ports []int, target Target) int {
	if len(ports) > 0 {
		return ports[0]
	}
	def := DefaultPorts(target)
	if len(def) > 0 {
		return def[0]
	}
	return 25
}
