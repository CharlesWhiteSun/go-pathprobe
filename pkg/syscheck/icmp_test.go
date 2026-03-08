package syscheck_test

import (
	"strings"
	"testing"

	"go-pathprobe/pkg/syscheck"
)

// Compile-time interface assertion.
var _ syscheck.ICMPChecker = syscheck.RawICMPChecker{}

func TestICMPAvailabilityNoticeAvailable(t *testing.T) {
	a := syscheck.ICMPAvailability{Available: true}
	notice := a.Notice()
	if !strings.Contains(strings.ToLower(notice), "available") {
		t.Errorf("expected notice to contain 'available', got %q", notice)
	}
}

func TestICMPAvailabilityNoticeUnavailable(t *testing.T) {
	a := syscheck.ICMPAvailability{Available: false}
	notice := a.Notice()
	if !strings.Contains(strings.ToLower(notice), "tcp") {
		t.Errorf("expected notice to mention 'tcp' fallback, got %q", notice)
	}
}

// TestICMPAvailabilityNoticeConsistency verifies Notice never returns an empty string.
func TestICMPAvailabilityNoticeConsistency(t *testing.T) {
	cases := []syscheck.ICMPAvailability{
		{Available: true},
		{Available: false},
	}
	for _, c := range cases {
		if c.Notice() == "" {
			t.Errorf("Notice() returned empty string for %+v", c)
		}
	}
}

// TestRawICMPCheckerDoesNotPanic exercises the real Check() call.
// On CI environments without raw-socket privileges, it will return
// Available=false — which is acceptable; we only verify the contract.
func TestRawICMPCheckerDoesNotPanic(t *testing.T) {
	avail := syscheck.RawICMPChecker{}.Check()
	if avail.Available && avail.Err != nil {
		t.Errorf("Available=true must have Err==nil, got %v", avail.Err)
	}
	if !avail.Available && avail.Err == nil {
		t.Error("Available=false must have a non-nil Err")
	}
}
