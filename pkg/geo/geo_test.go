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
