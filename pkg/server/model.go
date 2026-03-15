// Package server provides the embedded HTTP REST API for PathProbe.
// It wires the existing pkg/diag Dispatcher to HTTP handlers without coupling
// to the CLI layer.
package server

import "time"

// DiagRequest is the JSON body accepted by POST /api/diag.
type DiagRequest struct {
	Target  string     `json:"target"`
	Options ReqOptions `json:"options"`
}

// ReqOptions carries per-request configuration for a diagnostic run.
// Timeout is a Go duration string (e.g. "5s"); zero or missing defaults to
// defaultDiagTimeout.
// SFTP private-key authentication is intentionally not exposed via the HTTP
// API  use the CLI with a local key file instead.
type ReqOptions struct {
	MTRCount   int      `json:"mtr_count,omitempty"`
	Timeout    string   `json:"timeout,omitempty"`
	Insecure   bool     `json:"insecure,omitempty"`
	DisableGeo bool     `json:"disable_geo,omitempty"`
	Web        *ReqWeb  `json:"web,omitempty"`
	Net        *ReqNet  `json:"net,omitempty"`
	SMTP       *ReqSMTP `json:"smtp,omitempty"`
	FTP        *ReqFTP  `json:"ftp,omitempty"`
	SFTP       *ReqSFTP `json:"sftp,omitempty"`
}

// ReqWeb configures Web / DNS diagnostic parameters.
type ReqWeb struct {
	Mode    string   `json:"mode,omitempty"` // "" | "public-ip" | "dns" | "http" | "port" | "traceroute"
	Domains []string `json:"domains,omitempty"`
	Types   []string `json:"types,omitempty"` // e.g. ["A","AAAA","MX"]
	URL     string   `json:"url,omitempty"`
	MaxHops int      `json:"max_hops,omitempty"` // traceroute maximum TTL; 0 uses DefaultMaxHops
}

// ReqNet configures network connectivity probe parameters.
type ReqNet struct {
	Host  string `json:"host"`
	Ports []int  `json:"ports,omitempty"`
}

// ReqSMTP configures SMTP-layer probe parameters.
type ReqSMTP struct {
	Mode        string   `json:"mode,omitempty"` // "" | "handshake" | "auth" | "send"
	Domain      string   `json:"domain,omitempty"`
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"password,omitempty"`
	From        string   `json:"from,omitempty"`
	To          []string `json:"to,omitempty"`
	UseTLS      bool     `json:"use_tls,omitempty"`
	StartTLS    bool     `json:"start_tls,omitempty"`
	AuthMethods []string `json:"auth_methods,omitempty"`
	MXProbeAll  bool     `json:"mx_probe_all,omitempty"`
}

// ReqFTP configures FTP / FTPS probe parameters.
type ReqFTP struct {
	Mode     string `json:"mode,omitempty"` // "" | "login" | "list"
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	UseTLS   bool   `json:"use_tls,omitempty"`
	AuthTLS  bool   `json:"auth_tls,omitempty"`
	RunLIST  bool   `json:"run_list,omitempty"`
}

// ReqSFTP configures SFTP / SSH probe parameters.
// Private-key authentication is intentionally omitted from the HTTP API.
type ReqSFTP struct {
	Mode     string `json:"mode,omitempty"` // "" | "auth" | "ls"
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	RunLS    bool   `json:"run_ls,omitempty"`
}

// ErrorResponse is the JSON body for all 4xx / 5xx responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse is the JSON body returned by GET /api/health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HistoryListItem is a summary row returned by GET /api/history.
type HistoryListItem struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	Target      string `json:"target"`
	Host        string `json:"host"`
	GeneratedAt string `json:"generated_at"`
}

// defaultDiagTimeout is applied when ReqOptions.Timeout is empty or unparseable.
const defaultDiagTimeout = 30 * time.Second

// maxBodyBytes guards against oversized request payloads.
const maxBodyBytes = 64 * 1024 // 64 KB
