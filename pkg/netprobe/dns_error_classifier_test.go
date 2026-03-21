package netprobe

import "testing"

func TestClassifyDNSLookupError(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		errMsg string
		want   DNSErrorCategory
	}{
		// ── Empty error (no failure) ────────────────────────────────────────
		{
			name:   "empty error returns empty category",
			domain: "example.com",
			errMsg: "",
			want:   "",
		},

		// ── Input errors ────────────────────────────────────────────────────
		{
			name:   "URL domain with no-such-host classifies as input",
			domain: "https://www.example.com/",
			errMsg: "lookup https://www.example.com/: no such host",
			want:   DNSErrInput,
		},
		{
			name:   "URL domain with resolver error classifies as input",
			domain: "https://www.rakuten.co.jp/",
			errMsg: "resolver returned error status",
			want:   DNSErrInput,
		},
		{
			name:   "URL domain with deadline exceeded classifies as input",
			domain: "http://example.com",
			errMsg: "context deadline exceeded",
			want:   DNSErrInput,
		},
		{
			name:   "invalid character in error message classifies as input",
			domain: "example.com",
			errMsg: "lookup example.com: dnsquery: DNS name contains an invalid character",
			want:   DNSErrInput,
		},
		{
			name:   "invalid character case-insensitive",
			domain: "example.com",
			errMsg: "INVALID CHARACTER in DNS label",
			want:   DNSErrInput,
		},

		// ── Network transit failures ─────────────────────────────────────────
		{
			name:   "deadline exceeded classifies as network",
			domain: "example.com",
			errMsg: "lookup example.com on 8.8.8.8:53: read udp: context deadline exceeded",
			want:   DNSErrNetwork,
		},
		{
			name:   "timed out classifies as network",
			domain: "example.com",
			errMsg: "lookup example.com: timed out",
			want:   DNSErrNetwork,
		},
		{
			name:   "i/o timeout classifies as network",
			domain: "example.com",
			errMsg: "read udp: i/o timeout",
			want:   DNSErrNetwork,
		},
		{
			name:   "connection refused classifies as network",
			domain: "example.com",
			errMsg: "dial tcp 1.1.1.1:53: connection refused",
			want:   DNSErrNetwork,
		},
		{
			name:   "network unreachable classifies as network",
			domain: "example.com",
			errMsg: "dial udp: network unreachable",
			want:   DNSErrNetwork,
		},
		{
			name:   "no route to host classifies as network",
			domain: "example.com",
			errMsg: "dial tcp: no route to host",
			want:   DNSErrNetwork,
		},
		{
			name:   "dial prefix classifies as network",
			domain: "example.com",
			errMsg: "dial tcp 192.0.2.1:853: connection reset by peer",
			want:   DNSErrNetwork,
		},

		// ── NXDOMAIN ─────────────────────────────────────────────────────────
		{
			name:   "no such host with valid domain classifies as nxdomain",
			domain: "notexist.example.com",
			errMsg: "lookup notexist.example.com: no such host",
			want:   DNSErrNXDomain,
		},
		{
			name:   "no such host case-insensitive",
			domain: "gone.example",
			errMsg: "lookup gone.example: NO SUCH HOST",
			want:   DNSErrNXDomain,
		},

		// ── Resolver errors ───────────────────────────────────────────────────
		{
			name:   "resolver returned error classifies as resolver",
			domain: "example.com",
			errMsg: "resolver returned error status 500",
			want:   DNSErrResolver,
		},
		{
			name:   "servfail classifies as resolver",
			domain: "example.com",
			errMsg: "DNS SERVFAIL for example.com",
			want:   DNSErrResolver,
		},
		{
			name:   "servfail lowercase classifies as resolver",
			domain: "example.com",
			errMsg: "servfail",
			want:   DNSErrResolver,
		},

		// ── Unknown fallback ──────────────────────────────────────────────────
		{
			name:   "unrecognised error falls back to unknown",
			domain: "example.com",
			errMsg: "some unexpected resolver condition",
			want:   DNSErrUnknown,
		},
		{
			name:   "NXDOMAIN substring (not servfail) falls back to unknown when not matching",
			domain: "example.com",
			errMsg: "NXDOMAIN response received",
			want:   DNSErrUnknown, // "nxdomain" substring does not match "no such host" pattern
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyDNSLookupError(tc.domain, tc.errMsg)
			if got != tc.want {
				t.Errorf("ClassifyDNSLookupError(%q, %q) = %q; want %q",
					tc.domain, tc.errMsg, got, tc.want)
			}
		})
	}
}
