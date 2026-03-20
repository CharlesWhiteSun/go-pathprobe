package cli

import (
	"github.com/spf13/cobra"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/netprobe"
)

// FlagRegistrar registers protocol-specific CLI flags on cmd and binds them
// into the relevant sub-fields of opts. It returns an optional OptionsPreparer
// that is invoked just before dispatch to perform any deferred transformations
// (e.g. parsing []string into typed values). A nil return means no preparation
// is needed.
//
// To add support for a new diagnostic target, create a FlagRegistrar and
// supply it via a ProtocolPlugin.RegisterCLI — no other code needs to change.
type FlagRegistrar func(cmd *cobra.Command, opts *diag.Options) OptionsPreparer

// OptionsPreparer is called after flag parsing and before dispatch to
// transform CLI-level raw values into fully-typed diag.Options fields.
type OptionsPreparer func(opts *diag.Options) error

// DefaultRegistrars returns the built-in FlagRegistrar map for all protocols
// that carry protocol-specific CLI flags.  Targets absent from the map
// (TargetIMAP, TargetPOP) use only the shared network flags.
//
// Callers that use ProtocolPlugins should build this map via
// app.BuildRegistrars(plugins) instead of calling DefaultRegistrars directly.
func DefaultRegistrars() map[diag.Target]FlagRegistrar {
	return map[diag.Target]FlagRegistrar{
		diag.TargetWeb:  RegisterWebFlags,
		diag.TargetSMTP: RegisterSMTPFlags,
		diag.TargetFTP:  RegisterFTPFlags,
		diag.TargetSFTP: RegisterSFTPFlags,
		// TargetIMAP and TargetPOP intentionally omitted: only shared network flags apply.
	}
}

// RegisterWebFlags registers DNS/HTTP web-diagnostic flags and returns a
// preparer that converts raw record-type strings to typed RecordType values.
func RegisterWebFlags(cmd *cobra.Command, opts *diag.Options) OptionsPreparer {
	opts.Web.Domains = []string{"example.com"}
	rawTypes := []string{"A", "AAAA", "MX"}
	cmd.Flags().StringSliceVar(&opts.Web.Domains, "dns-domain", opts.Web.Domains, "domains to compare across resolvers")
	cmd.Flags().StringSliceVar(&rawTypes, "dns-type", rawTypes, "record types to query (A, AAAA, MX)")
	cmd.Flags().StringVar(&opts.Web.URL, "http-url", opts.Web.URL, "HTTP/HTTPS URL for protocol probe")
	return func(o *diag.Options) error {
		types, err := netprobe.ParseRecordTypes(rawTypes)
		if err != nil {
			return err
		}
		o.Web.Types = types
		return nil
	}
}

// RegisterSMTPFlags registers all SMTP-specific flags. Flags bind directly into
// opts.SMTP, so no deferred preparation is required (returns nil).
func RegisterSMTPFlags(cmd *cobra.Command, opts *diag.Options) OptionsPreparer {
	cmd.Flags().StringVar(&opts.SMTP.Domain, "smtp-domain", opts.SMTP.Domain, "domain for MX lookup or EHLO")
	cmd.Flags().StringVar(&opts.SMTP.Username, "smtp-user", opts.SMTP.Username, "SMTP username for auth")
	cmd.Flags().StringVar(&opts.SMTP.Password, "smtp-pass", opts.SMTP.Password, "SMTP password or app password")
	cmd.Flags().StringVar(&opts.SMTP.From, "smtp-from", opts.SMTP.From, "MAIL FROM address")
	cmd.Flags().StringSliceVar(&opts.SMTP.To, "smtp-to", opts.SMTP.To, "RCPT TO addresses")
	cmd.Flags().BoolVar(&opts.SMTP.UseTLS, "smtp-ssl", false, "use implicit SSL/TLS (SMTPS)")
	cmd.Flags().BoolVar(&opts.SMTP.StartTLS, "smtp-starttls", true, "attempt STARTTLS when supported")
	cmd.Flags().StringSliceVar(&opts.SMTP.AuthMethods, "smtp-auth-methods", nil, "auth mechanisms to try in order (PLAIN, LOGIN, XOAUTH2)")
	cmd.Flags().BoolVar(&opts.SMTP.MXProbeAll, "smtp-mx-all", false, "probe all MX records for the domain")
	return nil
}

// RegisterFTPFlags registers FTP/FTPS-specific flags (returns nil preparer).
func RegisterFTPFlags(cmd *cobra.Command, opts *diag.Options) OptionsPreparer {
	cmd.Flags().StringVar(&opts.FTP.Username, "ftp-user", opts.FTP.Username, "FTP username")
	cmd.Flags().StringVar(&opts.FTP.Password, "ftp-pass", opts.FTP.Password, "FTP password")
	cmd.Flags().BoolVar(&opts.FTP.UseTLS, "ftp-ssl", false, "use implicit FTPS (port 990)")
	cmd.Flags().BoolVar(&opts.FTP.AuthTLS, "ftp-auth-tls", false, "use explicit FTPS via AUTH TLS")
	cmd.Flags().BoolVar(&opts.FTP.RunLIST, "ftp-list", false, "attempt PASV + LIST after login")
	return nil
}

// RegisterSFTPFlags registers SFTP/SSH-specific flags (returns nil preparer).
func RegisterSFTPFlags(cmd *cobra.Command, opts *diag.Options) OptionsPreparer {
	cmd.Flags().StringVar(&opts.SFTP.Username, "sftp-user", opts.SFTP.Username, "SSH/SFTP username")
	cmd.Flags().StringVar(&opts.SFTP.Password, "sftp-pass", opts.SFTP.Password, "SSH/SFTP password")
	cmd.Flags().BoolVar(&opts.SFTP.RunLS, "sftp-ls", false, "attempt to list remote default directory via SFTP subsystem")
	return nil
}
