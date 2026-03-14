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
	JSON      bool
	Report    string
	MTRCount  int
	Timeout   time.Duration
	Insecure  bool
	LogLevel  slog.Level
	GeoDBCity string // path to GeoLite2-City.mmdb
	GeoDBASN  string // path to GeoLite2-ASN.mmdb
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
	FTP    FTPOptions
	SFTP   SFTPOptions
}

// NetworkOptions configures connectivity and traceroute-style probes.
type NetworkOptions struct {
	Host  string
	Ports []int
}

// SMTPOptions carries mail probe configuration.
type SMTPOptions struct {
	Mode        SMTPMode // "" | "handshake" | "auth" | "send"
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

// FTPOptions carries FTP/FTPS probe configuration.
type FTPOptions struct {
	Mode     FTPMode // "" | "login" | "list"
	Username string
	Password string
	UseTLS   bool // implicit FTPS
	AuthTLS  bool // explicit FTPS via AUTH TLS
	RunLIST  bool // attempt PASV + LIST after login
}

// SFTPOptions carries SFTP/SSH probe configuration.
type SFTPOptions struct {
	Mode       SFTPMode // "" | "auth" | "ls"
	Username   string
	Password   string
	PrivateKey []byte // PEM-encoded private key for public-key auth
	RunLS      bool   // attempt to list the remote default directory
}
