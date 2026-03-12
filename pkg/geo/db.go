package geo

import _ "embed"

// embeddedCountryDB holds the DB-IP Country Lite MMDB (CC BY 4.0).
// It provides country-level annotations (ISO code, country name) for IPv4/IPv6
// addresses without any runtime file-system dependency.
//
//go:embed db/country.mmdb
var embeddedCountryDB []byte

// embeddedAsnDB holds the DB-IP ASN Lite MMDB (CC BY 4.0).
// It provides AS number and organisation name for IPv4/IPv6 addresses.
//
//go:embed db/asn.mmdb
var embeddedAsnDB []byte
