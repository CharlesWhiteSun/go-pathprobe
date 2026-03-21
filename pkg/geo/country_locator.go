package geo

import (
	"errors"
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// CountryLocator resolves geo information using a country-level MMDB combined
// with hardcoded country centroid coordinates.  Geographic precision is at the
// country level (centre of the country polygon), but the information is fully
// self-contained—no external files or network requests are needed.
//
// It is used by AutoLocator when no explicit database paths are provided.
type CountryLocator struct {
	countryDB *geoip2.Reader
	asnDB     *geoip2.Reader
}

// openEmbedded creates a CountryLocator from the bytes embedded at build time.
// It returns NoopLocator (never an error) when the embedded data is absent or
// cannot be parsed, so callers always get a valid Locator.
func openEmbedded() (Locator, error) {
	if len(embeddedCountryDB) == 0 && len(embeddedAsnDB) == 0 {
		return NoopLocator{}, nil
	}
	loc := &CountryLocator{}
	if len(embeddedCountryDB) > 0 {
		r, err := geoip2.FromBytes(embeddedCountryDB)
		if err != nil {
			return NoopLocator{}, nil
		}
		loc.countryDB = r
	}
	if len(embeddedAsnDB) > 0 {
		r, err := geoip2.FromBytes(embeddedAsnDB)
		if err != nil {
			if loc.countryDB != nil {
				loc.countryDB.Close()
			}
			return NoopLocator{}, nil
		}
		loc.asnDB = r
	}
	return loc, nil
}

// LocateIP implements Locator for CountryLocator.
func (l *CountryLocator) LocateIP(ipStr string) (GeoInfo, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoInfo{}, fmt.Errorf("invalid IP address: %q", ipStr)
	}
	info := GeoInfo{IP: ipStr}

	if l.countryDB != nil {
		rec, err := l.countryDB.Country(ip)
		if err == nil && rec.Country.IsoCode != "" {
			info.CountryCode = rec.Country.IsoCode
			info.CountryName = rec.Country.Names["en"]
			if coords, ok := countryCentroids[rec.Country.IsoCode]; ok {
				info.Lat = coords[0]
				info.Lon = coords[1]
				info.HasLocation = true
				info.LocationPrecision = "country"
			}
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

// Close releases the underlying database readers.
func (l *CountryLocator) Close() error {
	var errs []error
	if l.countryDB != nil {
		if err := l.countryDB.Close(); err != nil {
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

// countryCentroids maps ISO 3166-1 alpha-2 country codes to approximate
// geographic centres [latitude, longitude].  Used to provide map coordinates
// for country-level lookups when a city database is not available.
//
// Source: https://developers.google.com/public-data/docs/canonical/countries_csv
var countryCentroids = map[string][2]float64{
	"AD": {42.55, 1.57},
	"AE": {23.42, 53.85},
	"AF": {33.94, 67.71},
	"AG": {17.06, -61.80},
	"AI": {18.22, -63.07},
	"AL": {41.15, 20.17},
	"AM": {40.07, 45.04},
	"AO": {-11.20, 17.87},
	"AQ": {-75.00, 0.00},
	"AR": {-38.42, -63.62},
	"AS": {-14.27, -170.13},
	"AT": {47.52, 14.55},
	"AU": {-25.27, 133.78},
	"AW": {12.52, -69.97},
	"AX": {60.18, 19.92},
	"AZ": {40.14, 47.58},
	"BA": {43.92, 17.68},
	"BB": {13.19, -59.54},
	"BD": {23.68, 90.35},
	"BE": {50.50, 4.47},
	"BF": {12.36, -1.56},
	"BG": {42.73, 25.49},
	"BH": {25.93, 50.64},
	"BI": {-3.37, 29.92},
	"BJ": {9.31, 2.32},
	"BL": {17.90, -62.83},
	"BM": {32.32, -64.76},
	"BN": {4.54, 114.73},
	"BO": {-16.29, -63.59},
	"BQ": {12.15, -68.27},
	"BR": {-14.24, -51.93},
	"BS": {25.03, -77.40},
	"BT": {27.51, 90.43},
	"BV": {-54.42, 3.41},
	"BW": {-22.33, 24.68},
	"BY": {53.71, 28.05},
	"BZ": {17.19, -88.49},
	"CA": {56.13, -106.35},
	"CC": {-12.16, 96.87},
	"CD": {-4.04, 21.76},
	"CF": {6.61, 20.94},
	"CG": {-0.23, 15.83},
	"CH": {46.82, 8.23},
	"CI": {7.54, -5.55},
	"CK": {-21.24, -159.78},
	"CL": {-35.68, -71.54},
	"CM": {3.85, 11.50},
	"CN": {35.86, 104.19},
	"CO": {4.57, -74.30},
	"CR": {9.75, -83.75},
	"CU": {21.52, -77.78},
	"CV": {16.54, -23.04},
	"CW": {12.17, -68.99},
	"CX": {-10.49, 105.69},
	"CY": {35.13, 33.43},
	"CZ": {49.82, 15.47},
	"DE": {51.17, 10.45},
	"DJ": {11.83, 42.59},
	"DK": {56.26, 9.50},
	"DM": {15.41, -61.37},
	"DO": {18.74, -70.16},
	"DZ": {28.03, 1.66},
	"EC": {-1.83, -78.18},
	"EE": {58.60, 25.01},
	"EG": {25.80, 28.83},
	"EH": {24.22, -12.89},
	"ER": {15.18, 39.78},
	"ES": {40.46, -3.75},
	"ET": {9.15, 40.49},
	"FI": {61.92, 25.75},
	"FJ": {-16.58, 179.41},
	"FK": {-51.80, -59.52},
	"FM": {6.92, 158.18},
	"FO": {61.89, -6.91},
	"FR": {46.23, 2.21},
	"GA": {-0.80, 11.61},
	"GB": {55.38, -3.44},
	"GD": {12.11, -61.68},
	"GE": {42.32, 43.36},
	"GF": {3.93, -53.13},
	"GG": {49.47, -2.58},
	"GH": {7.95, -1.02},
	"GI": {36.14, -5.35},
	"GL": {71.71, -42.60},
	"GM": {13.44, -15.31},
	"GN": {9.95, -11.61},
	"GP": {16.99, -62.07},
	"GQ": {1.65, 10.27},
	"GR": {39.07, 21.82},
	"GS": {-54.43, -36.59},
	"GT": {15.78, -90.23},
	"GU": {13.44, 144.79},
	"GW": {11.80, -15.18},
	"GY": {4.86, -58.93},
	"HK": {22.40, 114.11},
	"HM": {-53.08, 73.50},
	"HN": {15.20, -86.24},
	"HR": {45.10, 15.20},
	"HT": {18.97, -72.29},
	"HU": {47.16, 19.50},
	"ID": {-0.79, 113.92},
	"IE": {53.41, -8.24},
	"IL": {31.05, 34.85},
	"IM": {54.24, -4.57},
	"IN": {20.59, 78.96},
	"IO": {-6.34, 71.88},
	"IQ": {33.22, 43.68},
	"IR": {32.43, 53.69},
	"IS": {64.96, -19.02},
	"IT": {41.87, 12.57},
	"JE": {49.21, -2.13},
	"JM": {18.11, -77.30},
	"JO": {30.59, 36.24},
	"JP": {36.20, 138.25},
	"KE": {-0.02, 37.91},
	"KG": {41.20, 74.77},
	"KH": {12.57, 104.99},
	"KI": {-3.37, -168.73},
	"KM": {-11.88, 43.87},
	"KN": {17.36, -62.78},
	"KP": {40.34, 127.51},
	"KR": {35.91, 127.77},
	"KW": {29.31, 47.48},
	"KY": {19.51, -80.57},
	"KZ": {48.02, 66.92},
	"LA": {19.86, 102.50},
	"LB": {33.85, 35.86},
	"LC": {13.91, -60.98},
	"LI": {47.14, 9.55},
	"LK": {7.87, 80.77},
	"LR": {6.43, -9.43},
	"LS": {-29.61, 28.23},
	"LT": {55.17, 23.88},
	"LU": {49.82, 6.13},
	"LV": {56.88, 24.60},
	"LY": {26.34, 17.23},
	"MA": {31.79, -7.09},
	"MC": {43.75, 7.41},
	"MD": {47.41, 28.37},
	"ME": {42.71, 19.37},
	"MF": {18.08, -63.05},
	"MG": {-18.77, 46.87},
	"MH": {7.13, 171.18},
	"MK": {41.61, 21.75},
	"ML": {17.57, -3.99},
	"MM": {16.87, 96.11},
	"MN": {46.86, 103.85},
	"MO": {22.16, 113.55},
	"MP": {17.33, 145.38},
	"MQ": {14.64, -61.02},
	"MR": {21.00, -10.94},
	"MS": {16.74, -62.19},
	"MT": {35.94, 14.37},
	"MU": {-20.35, 57.55},
	"MV": {3.20, 73.22},
	"MW": {-13.25, 34.30},
	"MX": {23.63, -102.55},
	"MY": {4.21, 101.98},
	"MZ": {-18.67, 35.53},
	"NA": {-22.96, 18.49},
	"NC": {-20.90, 165.62},
	"NE": {17.61, 8.08},
	"NF": {-29.03, 167.95},
	"NG": {9.08, 8.68},
	"NI": {12.87, -85.21},
	"NL": {52.13, 5.29},
	"NO": {60.47, 8.47},
	"NP": {28.39, 84.12},
	"NR": {-0.52, 166.93},
	"NU": {-19.05, -169.87},
	"NZ": {-40.90, 174.89},
	"OM": {21.51, 55.92},
	"PA": {8.54, -80.78},
	"PE": {-9.19, -75.02},
	"PF": {-17.68, -149.41},
	"PG": {-6.31, 143.96},
	"PH": {12.88, 121.77},
	"PK": {30.38, 69.35},
	"PL": {51.92, 19.15},
	"PM": {46.94, -56.27},
	"PN": {-24.70, -127.44},
	"PR": {18.22, -66.59},
	"PS": {31.95, 35.23},
	"PT": {39.40, -8.22},
	"PW": {7.51, 134.58},
	"PY": {-23.44, -58.44},
	"QA": {25.35, 51.18},
	"RE": {-21.12, 55.54},
	"RO": {45.94, 24.97},
	"RS": {44.02, 21.01},
	"RU": {61.52, 105.32},
	"RW": {-1.94, 29.87},
	"SA": {23.89, 45.08},
	"SB": {-9.46, 160.16},
	"SC": {-4.68, 55.49},
	"SD": {12.86, 30.22},
	"SE": {60.13, 18.64},
	"SG": {1.35, 103.82},
	"SH": {-24.14, -10.03},
	"SI": {46.15, 14.99},
	"SJ": {77.55, 23.67},
	"SK": {48.67, 19.70},
	"SL": {8.46, -11.78},
	"SM": {43.94, 12.46},
	"SN": {14.50, -14.45},
	"SO": {5.15, 46.20},
	"SR": {3.92, -56.03},
	"SS": {4.86, 31.57},
	"ST": {0.19, 6.61},
	"SV": {13.79, -88.90},
	"SX": {18.04, -63.07},
	"SY": {34.80, 38.99},
	"SZ": {-26.52, 31.47},
	"TC": {21.69, -71.80},
	"TD": {15.45, 18.73},
	"TF": {-49.28, 69.35},
	"TG": {8.62, 0.82},
	"TH": {15.87, 100.99},
	"TJ": {38.86, 71.28},
	"TK": {-8.97, -171.86},
	"TL": {-8.87, 125.73},
	"TM": {38.97, 59.56},
	"TN": {33.89, 9.54},
	"TO": {-21.18, -175.20},
	"TR": {38.96, 35.24},
	"TT": {10.69, -61.22},
	"TV": {-7.11, 177.65},
	"TW": {23.70, 121.00},
	"TZ": {-6.37, 34.89},
	"UA": {48.38, 31.17},
	"UG": {1.37, 32.29},
	"UM": {19.30, 166.65},
	"US": {37.09, -95.71},
	"UY": {-32.52, -55.77},
	"UZ": {41.38, 64.59},
	"VA": {41.90, 12.45},
	"VC": {12.98, -61.29},
	"VE": {6.42, -66.59},
	"VG": {18.43, -64.62},
	"VI": {18.34, -64.90},
	"VN": {14.06, 108.28},
	"VU": {-15.38, 166.96},
	"WF": {-13.77, -177.16},
	"WS": {-13.76, -172.10},
	"XK": {42.60, 20.90},
	"YE": {15.55, 48.52},
	"YT": {-12.83, 45.17},
	"ZA": {-30.56, 22.94},
	"ZM": {-13.13, 27.85},
	"ZW": {-19.02, 29.15},
}
