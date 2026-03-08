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
type DNSAnswer struct {
	Name   string
	Type   RecordType
	Values []string
	RTT    time.Duration
	Source string
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
