package diag

import (
	"context"
	"log/slog"
	"strings"

	"go-pathprobe/pkg/netprobe"
)

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
func (r *SMTPRunner) Run(ctx context.Context, req Request) error {
	if r.Prober == nil {
		return ErrRunnerNotFound
	}
	opts := req.Options.SMTP

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

	for _, host := range hosts {
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
		r.Logger.Info("smtp probe", "host", host, "port", port, "banner", res.Banner, "starttls", res.UsedStartTLS, "rcpt", res.RcptAccepted)
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
