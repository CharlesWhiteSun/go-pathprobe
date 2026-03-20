package geo_test

import (
	"testing"

	"go-pathprobe/pkg/geo"
)

// TestNoopLocatorReturnsEmpty verifies that NoopLocator never errors and
// always returns a zero-value GeoInfo.
func TestNoopLocatorReturnsEmpty(t *testing.T) {
	loc := geo.NoopLocator{}
	info, err := loc.LocateIP("1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.IP != "" || info.City != "" || info.HasLocation {
		t.Fatalf("expected empty GeoInfo, got %+v", info)
	}
}

// TestNoopLocatorClose verifies that Close is a valid no-op.
func TestNoopLocatorClose(t *testing.T) {
	loc := geo.NoopLocator{}
	if err := loc.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// TestOpenNoPaths verifies that passing both empty paths returns a NoopLocator
// without error.
func TestOpenNoPaths(t *testing.T) {
	loc, err := geo.Open("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := loc.(geo.NoopLocator); !ok {
		t.Fatalf("expected NoopLocator when paths are empty")
	}
	_ = loc.Close()
}

// TestOpenInvalidCityPath verifies that an invalid city DB path returns an error.
func TestOpenInvalidCityPath(t *testing.T) {
	_, err := geo.Open("/nonexistent/GeoLite2-City.mmdb", "")
	if err == nil {
		t.Fatal("expected error for non-existent city DB path")
	}
}

// TestOpenInvalidASNPath verifies that an invalid ASN DB path returns an error.
func TestOpenInvalidASNPath(t *testing.T) {
	_, err := geo.Open("", "/nonexistent/GeoLite2-ASN.mmdb")
	if err == nil {
		t.Fatal("expected error for non-existent ASN DB path")
	}
}

// Compile-time assertions: concrete types satisfy the narrow IPLocator
// interface without requiring the full Locator (which includes Close).
var (
	_ geo.IPLocator = geo.NoopLocator{}
	_ geo.IPLocator = (*geo.GeoLite2Locator)(nil)
	_ geo.IPLocator = (*geo.CountryLocator)(nil)
)

// TestIPLocatorContract verifies that NoopLocator satisfies IPLocator at
// runtime and returns the expected zero-value GeoInfo.
func TestIPLocatorContract(t *testing.T) {
	var loc geo.IPLocator = geo.NoopLocator{}
	info, err := loc.LocateIP("8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error from NoopLocator.LocateIP: %v", err)
	}
	if info != (geo.GeoInfo{}) {
		t.Errorf("expected zero GeoInfo, got %+v", info)
	}
}
