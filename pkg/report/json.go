package report

import (
	"encoding/json"
	"io"
)

// JSONWriter renders an AnnotatedReport as compact, indented JSON.
type JSONWriter struct{}

// jsonPayload mirrors AnnotatedReport with JSON-friendly field tags.
type jsonPayload struct {
	Target      string        `json:"target"`
	Host        string        `json:"host"`
	GeneratedAt string        `json:"generated_at"`
	PublicGeo   GeoAnnotation `json:"public_geo"`
	TargetGeo   GeoAnnotation `json:"target_geo"`
	Ports       []PortEntry   `json:"ports"`
	Protos      []ProtoEntry  `json:"protos"`
}

// Write outputs the report as indented JSON to w.
func (JSONWriter) Write(w io.Writer, r *AnnotatedReport) error {
	payload := jsonPayload{
		Target:      r.Target,
		Host:        r.Host,
		GeneratedAt: r.GeneratedAt,
		PublicGeo:   r.PublicGeo,
		TargetGeo:   r.TargetGeo,
		Ports:       r.Ports,
		Protos:      r.Protos,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
