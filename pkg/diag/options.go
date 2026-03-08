package diag

import (
	"errors"
	"log/slog"
	"time"
)

const (
	// DefaultMTRCount specifies default probe count per hop for traceroute-style diagnostics.
	DefaultMTRCount = 5
)

// GlobalOptions captures flags shared across diagnostic targets.
type GlobalOptions struct {
	JSON     bool
	Report   string
	MTRCount int
	Timeout  time.Duration
	Insecure bool
	LogLevel slog.Level
}

// Validate ensures the option values are within acceptable ranges.
func (o GlobalOptions) Validate() error {
	if o.MTRCount <= 0 {
		return errors.New("mtr-count must be greater than zero")
	}
	if o.Timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	return nil
}

// Options bundles global options with target-specific placeholders for future extension.
type Options struct {
	Global GlobalOptions
	Web    WebOptions
	Net    NetworkOptions
	SMTP   SMTPOptions
}

// NetworkOptions configures connectivity and traceroute-style probes.
type NetworkOptions struct {
	Host  string
	Ports []int
}

// SMTPOptions carries mail probe configuration.
type SMTPOptions struct {
	Domain      string
	Username    string
	Password    string
	From        string
	To          []string
	UseTLS      bool     // implicit TLS (SMTPS)
	StartTLS    bool     // attempt STARTTLS after EHLO
	AuthMethods []string // ordered auth mechanisms to try (PLAIN, LOGIN, XOAUTH2); empty = server preference [PLAIN, LOGIN]
	MXProbeAll  bool     // when true, probe all MX records for the domain instead of only the first
}
