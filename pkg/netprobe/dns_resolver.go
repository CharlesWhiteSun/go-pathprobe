package netprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DNSResolver resolves DNS records for a name/type pair.
type DNSResolver interface {
	Lookup(ctx context.Context, name string, rtype RecordType) (DNSAnswer, error)
}

// HTTPDNSResolver implements DNSResolver via DNS-over-HTTPS JSON API (Google/Cloudflare style).
type HTTPDNSResolver struct {
	Client   *http.Client
	Endpoint string // e.g., https://dns.google/resolve or https://cloudflare-dns.com/dns-query
	Name     string
}

// dohResponse represents minimal fields from DNS JSON API.
type dohResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Data string `json:"data"`
	} `json:"Answer"`
}

// Lookup performs a DoH lookup.
func (r *HTTPDNSResolver) Lookup(ctx context.Context, name string, rtype RecordType) (DNSAnswer, error) {
	if r.Client == nil {
		r.Client = http.DefaultClient
	}
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.Endpoint, nil)
	if err != nil {
		return DNSAnswer{}, err
	}
	q := req.URL.Query()
	q.Set("name", name)
	q.Set("type", string(rtype))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return DNSAnswer{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DNSAnswer{}, fmt.Errorf("doh status %d", resp.StatusCode)
	}

	var dr dohResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return DNSAnswer{}, err
	}
	if dr.Status != 0 {
		return DNSAnswer{}, errors.New("resolver returned error status")
	}

	values := make([]string, 0, len(dr.Answer))
	for _, ans := range dr.Answer {
		trimmed := strings.TrimSpace(ans.Data)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return DNSAnswer{
		Name:   name,
		Type:   rtype,
		Values: values,
		RTT:    time.Since(start),
		Source: r.Name,
	}, nil
}

// SystemResolver wraps net.Resolver for system DNS.
type SystemResolver struct {
	Name     string
	Resolver *net.Resolver
}

func (r *SystemResolver) Lookup(ctx context.Context, name string, rtype RecordType) (DNSAnswer, error) {
	res := r.Resolver
	if res == nil {
		res = net.DefaultResolver
	}
	start := time.Now()
	var records []string
	var err error

	switch rtype {
	case RecordTypeA:
		var ips []net.IP
		ips, err = res.LookupIP(ctx, "ip4", name)
		if err == nil {
			records = stringifyIPs(ips)
		}
	case RecordTypeAAAA:
		var ips []net.IP
		ips, err = res.LookupIP(ctx, "ip6", name)
		if err == nil {
			records = stringifyIPs(ips)
		}
	case RecordTypeMX:
		var mx []*net.MX
		mx, err = res.LookupMX(ctx, name)
		if err == nil {
			records = stringifyMX(mx)
		}
	default:
		return DNSAnswer{}, fmt.Errorf("unsupported record type: %s", rtype)
	}

	if err != nil {
		return DNSAnswer{}, err
	}

	return DNSAnswer{
		Name:   name,
		Type:   rtype,
		Values: records,
		RTT:    time.Since(start),
		Source: r.Name,
	}, nil
}

func stringifyIPs(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

func stringifyMX(mx []*net.MX) []string {
	out := make([]string, 0, len(mx))
	for _, rec := range mx {
		out = append(out, fmt.Sprintf("%s:%d", strings.TrimSuffix(rec.Host, "."), rec.Pref))
	}
	return out
}

// DNSComparator compares answers across resolvers to detect divergence.
type DNSComparator struct {
	Resolvers []DNSResolver
}

// Compare evaluates a set of domains and record types.
func (c DNSComparator) Compare(ctx context.Context, domains []string, types []RecordType) ([]DNSComparison, error) {
	var comparisons []DNSComparison
	for _, domain := range domains {
		for _, rt := range types {
			comp, err := c.compareSingle(ctx, domain, rt)
			if err != nil {
				return nil, err
			}
			comparisons = append(comparisons, comp)
		}
	}
	return comparisons, nil
}

func (c DNSComparator) compareSingle(ctx context.Context, domain string, rt RecordType) (DNSComparison, error) {
	results := make([]DNSAnswer, 0, len(c.Resolvers))
	for _, resolver := range c.Resolvers {
		ans, err := resolver.Lookup(ctx, domain, rt)
		if err != nil {
			return DNSComparison{}, err
		}
		results = append(results, ans)
	}
	return DNSComparison{Name: domain, Type: rt, Results: results}, nil
}

// ParseRecordTypes converts textual record types into constants.
func ParseRecordTypes(inputs []string) ([]RecordType, error) {
	var out []RecordType
	for _, in := range inputs {
		normalized := strings.ToUpper(strings.TrimSpace(in))
		switch normalized {
		case "A":
			out = append(out, RecordTypeA)
		case "AAAA":
			out = append(out, RecordTypeAAAA)
		case "MX":
			out = append(out, RecordTypeMX)
		case "", " ":
			continue
		default:
			return nil, fmt.Errorf("unsupported record type: %s", normalized)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("at least one record type is required")
	}
	return out, nil
}

// TTLFromSeconds parses numeric TTL strings for potential future use.
func TTLFromSeconds(value string) (time.Duration, error) {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if seconds < 0 {
		return 0, errors.New("ttl cannot be negative")
	}
	return time.Duration(seconds) * time.Second, nil
}
