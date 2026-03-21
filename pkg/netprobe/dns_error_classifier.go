package netprobe

import "strings"

// DNSErrorCategory classifies a DNS lookup failure for user-facing display.
// The value is a plain string so it serialises transparently to JSON and can
// be consumed directly by the renderer without a mapping step.
type DNSErrorCategory string

const (
	// DNSErrInput indicates the query domain is malformed (e.g. a URL was
	// entered instead of a bare hostname, or the name contains characters
	// that are illegal in DNS labels).  The error originates with the caller.
	DNSErrInput DNSErrorCategory = "input"

	// DNSErrNXDomain indicates the domain name does not exist in DNS.
	// The response comes from the authoritative server and the domain format
	// itself is valid.
	DNSErrNXDomain DNSErrorCategory = "nxdomain"

	// DNSErrNetwork indicates a transit-level failure: the probe could not
	// reach the DNS resolver (timeout, connection refused, unreachable route).
	DNSErrNetwork DNSErrorCategory = "network"

	// DNSErrResolver indicates the DNS resolver (DoH endpoint or recursive
	// resolver) responded but returned an error status such as SERVFAIL.
	DNSErrResolver DNSErrorCategory = "resolver"

	// DNSErrUnknown is the fallback for unrecognised error messages.
	DNSErrUnknown DNSErrorCategory = "unknown"
)

// ClassifyDNSLookupError returns the DNSErrorCategory that best explains the
// given raw Go resolver error string.  domain is also examined so that a
// "no such host" response for a URL-format domain is classified as an input
// error rather than NXDOMAIN.
//
// Classification priority (highest → lowest):
//
//  1. input   — domain contains "://" or errMsg contains "invalid character"
//  2. network — errMsg matches deadline / timeout / connection failure patterns
//  3. nxdomain — errMsg contains "no such host"
//  4. resolver — errMsg matches DoH / SERVFAIL patterns
//  5. unknown  — all other non-empty error messages
//
// Returns an empty DNSErrorCategory when errMsg is empty (no error).
func ClassifyDNSLookupError(domain, errMsg string) DNSErrorCategory {
	if errMsg == "" {
		return ""
	}
	lower := strings.ToLower(errMsg)

	// Priority 1 — Input format errors.
	// A URL-style domain (contains "://") is never a valid DNS name; any
	// error it produces is an input error regardless of the raw message.
	if strings.Contains(domain, "://") ||
		strings.Contains(lower, "invalid character") {
		return DNSErrInput
	}

	// Priority 2 — Network transit failures (cannot reach the resolver).
	if strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "network unreachable") ||
		strings.Contains(lower, "no route to host") ||
		strings.Contains(lower, "dial ") {
		return DNSErrNetwork
	}

	// Priority 3 — Domain not found (NXDOMAIN).
	if strings.Contains(lower, "no such host") {
		return DNSErrNXDomain
	}

	// Priority 4 — Resolver returned an error status (DoH HTTP error,
	// SERVFAIL, or similar DNS RCODE ≠ NOERROR).
	if strings.Contains(lower, "resolver returned error") ||
		strings.Contains(lower, "servfail") {
		return DNSErrResolver
	}

	return DNSErrUnknown
}
