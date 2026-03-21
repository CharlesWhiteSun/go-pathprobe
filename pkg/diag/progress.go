package diag

// HopProgressData carries structured per-hop data emitted during a traceroute.
// It is attached to ProgressEvent when Stage == "traceroute-hop".
// IP is an empty string for hops that did not respond within the timeout.
type HopProgressData struct {
	TTL      int     `json:"ttl"`
	MaxHops  int     `json:"max_hops"` // configured maximum hops for this run
	IP       string  `json:"ip"`       // empty for timed-out hops
	Hostname string  `json:"hostname"`
	AvgRTT   string  `json:"avg_rtt"`
	LossPct  float64 `json:"loss_pct"`
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
}

// ProgressEvent is a single progress update emitted by a Runner during
// execution. Stage is a short machine-readable identifier (e.g. "network",
// "smtp"); Message is the human-readable description of the current activity.
// Hop is non-nil only for "traceroute-hop" events.
type ProgressEvent struct {
	Stage   string           `json:"stage"`
	Message string           `json:"message"`
	Hop     *HopProgressData `json:"hop,omitempty"`
}

// ProgressHook is an optional callback that Runners invoke to report progress.
// Implementations must be safe for concurrent use.
type ProgressHook func(ProgressEvent)
