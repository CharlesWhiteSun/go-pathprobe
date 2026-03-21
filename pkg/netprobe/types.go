package netprobe

import (
	"sort"
	"time"
)

// RecordType enumerates DNS record types we support.
type RecordType string

const (
	RecordTypeA    RecordType = "A"
	RecordTypeAAAA RecordType = "AAAA"
	RecordTypeMX   RecordType = "MX"
)

// PublicIPResult captures the observed public IP from a probe.
type PublicIPResult struct {
	IP     string
	Source string
	RTT    time.Duration
}

// DNSAnswer stores answers for a single DNS lookup.
// LookupError is non-empty when the resolver returned an error; Values will be nil in that case.
type DNSAnswer struct {
	Name        string
	Type        RecordType
	Values      []string
	RTT         time.Duration
	Source      string
	LookupError string
}

// DNSComparison aggregates lookup results across resolvers and flags divergence.
type DNSComparison struct {
	Name    string
	Type    RecordType
	Results []DNSAnswer
}

// HasDivergence reports whether any two resolvers returned different answer sets.
// Answer order is normalised before comparison, so ordering differences alone are
// not flagged as divergence.
func (c DNSComparison) HasDivergence() bool {
	if len(c.Results) < 2 {
		return false
	}
	ref := sortedStringSlice(c.Results[0].Values)
	for _, r := range c.Results[1:] {
		if !equalStringSlices(ref, sortedStringSlice(r.Values)) {
			return true
		}
	}
	return false
}

// AllFailed reports whether every resolver returned a LookupError.
// This is the highest-priority status: it distinguishes "all resolvers errored"
// from divergence (some disagree), all-empty (none had records), or consistent.
// Any resolver that succeeded without error causes AllFailed to return false.
func (c DNSComparison) AllFailed() bool {
	if len(c.Results) == 0 {
		return false
	}
	for _, r := range c.Results {
		if r.LookupError == "" {
			return false
		}
	}
	return true
}

// AllEmpty reports whether every resolver successfully responded with no records.
// This distinguishes "domain has no records of this type" from "results are consistent".
// A result with a LookupError is considered a failure, not an empty response,
// so any LookupError causes AllEmpty to return false.
func (c DNSComparison) AllEmpty() bool {
	if len(c.Results) == 0 {
		return false
	}
	for _, r := range c.Results {
		if r.LookupError != "" {
			return false
		}
		if len(r.Values) > 0 {
			return false
		}
	}
	return true
}

// NoneFound reports whether no resolver returned any actual records, regardless
// of whether individual resolvers succeeded with empty results or failed with an
// error (e.g. NXDOMAIN).  It returns true when every resolver either returned an
// empty Values slice or a LookupError, AND the entry is not AllFailed (at least
// one resolver responded without error) AND resolvers do not actively disagree on
// records (HasDivergence is false).
//
// This is the fifth badge state, sitting between HasDivergence and the implicit
// "consistent non-empty" state in priority:
//
//	AllFailed     — every resolver errored         (highest priority)
//	HasDivergence — resolvers disagree on values
//	NoneFound     — no records found at all (mix of errors + empty OK)
//	consistent    — resolvers agree on non-empty records
//
// Typical trigger: system resolver returns NXDOMAIN while DoH resolvers return
// empty without error for a record type the domain does not have (e.g. AAAA or MX).
// Previously this fell through to "Consistent" because Values were all empty, but
// NoneFound separates it correctly as a "no records" outcome.
func (c DNSComparison) NoneFound() bool {
	if len(c.Results) == 0 {
		return false
	}
	// At least one resolver must have responded without error (otherwise AllFailed).
	if c.AllFailed() {
		return false
	}
	// Resolvers must not disagree on actual record values.
	if c.HasDivergence() {
		return false
	}
	// True only when no resolver produced any actual records.
	for _, r := range c.Results {
		if len(r.Values) > 0 {
			return false
		}
	}
	return true
}

func sortedStringSlice(vs []string) []string {
	out := make([]string, len(vs))
	copy(out, vs)
	sort.Strings(out)
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
