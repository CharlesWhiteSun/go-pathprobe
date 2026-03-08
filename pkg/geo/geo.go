// Package geo provides geographic and ASN annotation for IP addresses using
// MaxMind GeoLite2 offline databases.  All public types implement graceful
// degradation: when no database is configured a NoopLocator is used and
// callers receive empty GeoInfo values without errors.
package geo

import (
	"errors"
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// GeoInfo holds the geographic and network data resolved for a single IP.
type GeoInfo struct {
	IP          string
	Lat         float64
	Lon         float64
	City        string
	CountryCode string
	CountryName string
	ASN         uint
	OrgName     string
	// HasLocation is true when Lat/Lon are populated from a city database.
	HasLocation bool
}

// Locator resolves geographic information for IP addresses.
type Locator interface {
	// LocateIP returns GeoInfo for the given IP string.
	// Implementations return empty GeoInfo (not an error) when the IP is not
	// found in the database.
	LocateIP(ip string) (GeoInfo, error)
	// Close releases any underlying file handles.
	Close() error
}

// NoopLocator implements Locator and always returns empty GeoInfo.
// It is used when no GeoLite2 database paths are configured.
type NoopLocator struct{}

// LocateIP always returns an empty GeoInfo with no error.
func (NoopLocator) LocateIP(_ string) (GeoInfo, error) { return GeoInfo{}, nil }

// Close is a no-op.
func (NoopLocator) Close() error { return nil }

// GeoLite2Locator reads from MaxMind GeoLite2 City and/or ASN .mmdb files.
// Either database may be nil (omitted) for partial lookups.
type GeoLite2Locator struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

// Open loads the GeoLite2 database files at the given paths.
// Either path may be empty, in which case that database is skipped.
// If both paths are empty, Open returns a NoopLocator.
func Open(cityPath, asnPath string) (Locator, error) {
	if cityPath == "" && asnPath == "" {
		return NoopLocator{}, nil
	}
	loc := &GeoLite2Locator{}
	if cityPath != "" {
		r, err := geoip2.Open(cityPath)
		if err != nil {
			return nil, fmt.Errorf("open city db %q: %w", cityPath, err)
		}
		loc.cityDB = r
	}
	if asnPath != "" {
		r, err := geoip2.Open(asnPath)
		if err != nil {
			if loc.cityDB != nil {
				loc.cityDB.Close()
			}
			return nil, fmt.Errorf("open asn db %q: %w", asnPath, err)
		}
		loc.asnDB = r
	}
	return loc, nil
}

// LocateIP resolves geographic and ASN information for the given IP string.
// Parse failures and database misses are returned as non-fatal empty GeoInfo.
func (l *GeoLite2Locator) LocateIP(ipStr string) (GeoInfo, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoInfo{}, fmt.Errorf("invalid IP address: %q", ipStr)
	}
	info := GeoInfo{IP: ipStr}

	if l.cityDB != nil {
		rec, err := l.cityDB.City(ip)
		if err == nil {
			info.Lat = rec.Location.Latitude
			info.Lon = rec.Location.Longitude
			info.City = rec.City.Names["en"]
			info.CountryCode = rec.Country.IsoCode
			info.CountryName = rec.Country.Names["en"]
			// HasLocation only when at least one coordinate is non-zero.
			info.HasLocation = rec.Location.Latitude != 0 || rec.Location.Longitude != 0
		}
	}

	if l.asnDB != nil {
		rec, err := l.asnDB.ASN(ip)
		if err == nil {
			info.ASN = rec.AutonomousSystemNumber
			info.OrgName = rec.AutonomousSystemOrganization
		}
	}

	return info, nil
}

// Close releases the database file handles.
func (l *GeoLite2Locator) Close() error {
	var errs []error
	if l.cityDB != nil {
		if err := l.cityDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if l.asnDB != nil {
		if err := l.asnDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
