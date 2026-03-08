package diag

// ProgressEvent is a single progress update emitted by a Runner during
// execution. Stage is a short machine-readable identifier (e.g. "network",
// "smtp"); Message is the human-readable description of the current activity.
type ProgressEvent struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// ProgressHook is an optional callback that Runners invoke to report progress.
// Implementations must be safe for concurrent use.
type ProgressHook func(ProgressEvent)
