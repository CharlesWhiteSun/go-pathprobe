package geo_test

import (
	"strings"
	"testing"

	"go-pathprobe/pkg/geo"
)

// TestAutoLocator_EmbeddedReturnsFunctionalLocator verifies that AutoLocator
// returns a working locator (not NoopLocator) when no file paths are given,
// because the build-time embedded databases are present.
func TestAutoLocator_EmbeddedReturnsFunctionalLocator(t *testing.T) {
	loc, err := geo.AutoLocator("", "")
	if err != nil {
		t.Fatalf("AutoLocator returned unexpected error: %v", err)
	}
	defer loc.Close()

	if _, ok := loc.(geo.NoopLocator); ok {
		t.Fatal("AutoLocator should not return NoopLocator when embedded DBs are present")
	}
}

// TestAutoLocator_WellKnownPublicIP probes a well-known public IP (Google DNS
// 8.8.8.8 = US) and verifies that the embedded locator resolves its country.
func TestAutoLocator_WellKnownPublicIP(t *testing.T) {
	loc, err := geo.AutoLocator("", "")
	if err != nil {
		t.Fatalf("AutoLocator: %v", err)
	}
	defer loc.Close()

	info, err := loc.LocateIP("8.8.8.8")
	if err != nil {
		t.Fatalf("LocateIP(8.8.8.8): %v", err)
	}
	if info.CountryCode == "" {
		t.Error("expected non-empty CountryCode for 8.8.8.8")
	}
	if !info.HasLocation {
		t.Error("expected HasLocation=true for 8.8.8.8 (US has centroid)")
	}
	if info.Lat == 0 && info.Lon == 0 {
		t.Error("expected non-zero lat/lon for 8.8.8.8")
	}
}

// TestAutoLocator_ASNPopulated verifies that the embedded ASN database
// provides a non-empty autonomous system organisation for 8.8.8.8 (Google).
func TestAutoLocator_ASNPopulated(t *testing.T) {
	loc, err := geo.AutoLocator("", "")
	if err != nil {
		t.Fatalf("AutoLocator: %v", err)
	}
	defer loc.Close()

	info, err := loc.LocateIP("8.8.8.8")
	if err != nil {
		t.Fatalf("LocateIP: %v", err)
	}
	if info.OrgName == "" && info.ASN == 0 {
		t.Error("expected ASN info for 8.8.8.8 (Google DNS)")
	}
}

// TestAutoLocator_InvalidIPReturnsError checks that an invalid IP string
// returns an error without panicking.
func TestAutoLocator_InvalidIPReturnsError(t *testing.T) {
	loc, err := geo.AutoLocator("", "")
	if err != nil {
		t.Fatalf("AutoLocator: %v", err)
	}
	defer loc.Close()

	_, err = loc.LocateIP("not-an-ip")
	if err == nil {
		t.Error("expected error for invalid IP, got nil")
	}
	if !strings.Contains(err.Error(), "not-an-ip") {
		t.Errorf("error should mention the bad input, got: %v", err)
	}
}

// TestAutoLocator_WithFilePaths_DelegatesToOpen verifies that AutoLocator
// delegates to Open when explicit paths are given (returning an error for
// non-existent paths, just as Open does).
func TestAutoLocator_WithFilePaths_DelegatesToOpen(t *testing.T) {
	_, err := geo.AutoLocator("/nonexistent/city.mmdb", "")
	if err == nil {
		t.Fatal("expected error when non-existent city path is provided")
	}
}

// TestAutoLocator_PrivateIPReturnsEmptyGeo confirms that private/RFC-1918
// addresses are not matched (no country info) but no error is returned.
func TestAutoLocator_PrivateIPReturnsEmptyGeo(t *testing.T) {
	loc, err := geo.AutoLocator("", "")
	if err != nil {
		t.Fatalf("AutoLocator: %v", err)
	}
	defer loc.Close()

	// 192.168.0.1 is private; the DB should return no country record.
	info, err := loc.LocateIP("192.168.0.1")
	if err != nil {
		t.Fatalf("LocateIP(192.168.0.1): unexpected error: %v", err)
	}
	// Private IPs have no country entry; HasLocation and CountryCode should
	// remain at their zero values.
	if info.HasLocation {
		t.Errorf("private IP should not HasLocation, got info=%+v", info)
	}
}

// TestCountryCentroid_USHasExpectedCoords provides a lightweight sanity check
// that the US centroid is in the Western Hemisphere at reasonable latitude.
func TestCountryCentroid_USHasExpectedCoords(t *testing.T) {
	loc, err := geo.AutoLocator("", "")
	if err != nil {
		t.Fatalf("AutoLocator: %v", err)
	}
	defer loc.Close()

	info, err := loc.LocateIP("8.8.8.8") // Google — resolves to US
	if err != nil {
		t.Fatalf("LocateIP: %v", err)
	}
	if info.CountryCode != "US" {
		t.Skipf("8.8.8.8 resolved to %q instead of US — skipping centroid check", info.CountryCode)
	}
	if info.Lon >= 0 {
		t.Errorf("US centroid longitude should be negative (Western Hemisphere), got %v", info.Lon)
	}
	if info.Lat < 20 || info.Lat > 60 {
		t.Errorf("US latitude %v is out of expected range [20,60]", info.Lat)
	}
}
