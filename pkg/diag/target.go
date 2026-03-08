package diag

// Target represents a diagnostic domain.
type Target string

const (
	TargetWeb  Target = "web"
	TargetSMTP Target = "smtp"
	TargetIMAP Target = "imap"
	TargetPOP  Target = "pop"
	TargetFTP  Target = "ftp"
	TargetSFTP Target = "sftp"
)

// AllTargets lists all supported diagnostic targets.
var AllTargets = []Target{TargetWeb, TargetSMTP, TargetIMAP, TargetPOP, TargetFTP, TargetSFTP}

func (t Target) String() string {
	return string(t)
}
