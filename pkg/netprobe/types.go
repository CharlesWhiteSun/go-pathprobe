package netprobe

import "time"

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
