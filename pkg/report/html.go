package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"text/template"
)

//go:embed templates/report.html
var reportHTMLTemplate string

// HTMLWriter renders an AnnotatedReport as a self-contained HTML report
// using the embedded Leaflet+OSM template.
type HTMLWriter struct{}

// Write renders the HTML report to w.
func (HTMLWriter) Write(w io.Writer, r *AnnotatedReport) error {
	tmpl, err := template.New("report").Parse(reportHTMLTemplate)
	if err != nil {
		return fmt.Errorf("parse HTML template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r); err != nil {
		return fmt.Errorf("render HTML template: %w", err)
	}
	_, err = w.Write(buf.Bytes())
	return err
}

// WriteFile renders the HTML report and writes it to the given file path.
func (h HTMLWriter) WriteFile(path string, r *AnnotatedReport) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report file %q: %w", path, err)
	}
	defer f.Close()
	return h.Write(f, r)
}
