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

// DefaultPorts returns recommended ports per target; callers may override via CLI flags.
func DefaultPorts(target Target) []int {
	switch target {
	case TargetWeb:
		return []int{443}
	case TargetSMTP:
		return []int{25, 465, 587}
	case TargetIMAP:
		return []int{143, 993}
	case TargetPOP:
		return []int{110, 995}
	case TargetFTP:
		return []int{21, 990}
	case TargetSFTP:
		return []int{22}
	default:
		return []int{}
	}
}

func (t Target) String() string {
	return string(t)
}
