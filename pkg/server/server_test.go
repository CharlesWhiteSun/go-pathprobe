package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/geo"
	"go-pathprobe/pkg/report"
	"go-pathprobe/pkg/server"
)

// ---- test doubles -------------------------------------------------------

// stubRunner is a diag.Runner test double that returns a preset error.
type stubRunner struct{ err error }

func (s stubRunner) Run(_ context.Context, _ diag.Request) error { return s.err }

// newHandler returns the http.Handler of a test Server wired with the given
// dispatcher and a NoopLocator.
func newHandler(t *testing.T, dispatcher *diag.Dispatcher) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := server.New(server.DefaultConfig(), dispatcher, geo.NoopLocator{}, logger)
	return srv.Handler()
}

// diagBody serialises a DiagRequest to a JSON *bytes.Buffer.
func diagBody(t *testing.T, req server.DiagRequest) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal DiagRequest: %v", err)
	}
	return bytes.NewBuffer(b)
}

// ---- GET /api/health ----------------------------------------------------

func TestHealthEndpoint_ReturnsOKWithVersion(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("want Content-Type application/json, got %q", ct)
	}

	var resp server.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.Version == "" {
		t.Error("version must not be empty")
	}
}

func TestHealthEndpoint_MethodNotAllowed(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/health", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rec.Code)
	}
}

// ---- POST /api/diag — request validation --------------------------------

func TestDiagEndpoint_InvalidJSON_Returns400(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag",
		strings.NewReader(`{not valid json`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	assertErrorField(t, rec.Body)
}

func TestDiagEndpoint_EmptyBody_Returns400(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag",
		strings.NewReader("")))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestDiagEndpoint_UnknownTarget_Returns400(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	body := diagBody(t, server.DiagRequest{Target: "unknown"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	assertErrorField(t, rec.Body)
}

func TestDiagEndpoint_InvalidWebTypes_Returns400(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	body := diagBody(t, server.DiagRequest{
		Target: "web",
		Options: server.ReqOptions{
			Web: &server.ReqWeb{Types: []string{"TXT"}}, // unsupported type
		},
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestDiagEndpoint_BodyTooLarge_Returns400(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	// Send 65 KB of zeros — exceeds maxBodyBytes
	body := strings.NewReader(strings.Repeat("a", 65*1024))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestDiagEndpoint_MethodNotAllowed(t *testing.T) {
	// Use POST /api/health (a GET-only endpoint with no POST catch-all) so that
	// the expected 405 is unaffected by the GET / static catch-all handler.
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/health", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rec.Code)
	}
}

// ---- POST /api/diag — dispatcher outcomes -------------------------------

func TestDiagEndpoint_RunnerNotFound_Returns404(t *testing.T) {
	// Empty dispatcher — no runners registered for any target.
	h := newHandler(t, diag.NewDispatcher(nil))
	body := diagBody(t, server.DiagRequest{
		Target: "web",
		Options: server.ReqOptions{
			Net: &server.ReqNet{Host: "127.0.0.1"},
		},
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
	assertErrorField(t, rec.Body)
}

func TestDiagEndpoint_RunnerError_Returns500(t *testing.T) {
	d := diag.NewDispatcher(nil)
	d.Register(diag.TargetWeb, stubRunner{err: io.ErrUnexpectedEOF})

	h := newHandler(t, d)
	body := diagBody(t, server.DiagRequest{
		Target: "web",
		Options: server.ReqOptions{
			Net: &server.ReqNet{Host: "127.0.0.1"},
		},
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	assertErrorField(t, rec.Body)
}

func TestDiagEndpoint_Success_Returns200WithAnnotatedReport(t *testing.T) {
	d := diag.NewDispatcher(nil)
	// Stub runner: succeeds without touching the network.
	d.Register(diag.TargetWeb, stubRunner{})

	h := newHandler(t, d)
	body := diagBody(t, server.DiagRequest{
		Target: "web",
		Options: server.ReqOptions{
			// Use a loopback IP to avoid DNS resolution inside report.Build.
			Net: &server.ReqNet{Host: "127.0.0.1", Ports: []int{80}},
			Web: &server.ReqWeb{
				Domains: []string{"example.com"},
				Types:   []string{"A"},
			},
		},
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}

	var ar report.AnnotatedReport
	if err := json.NewDecoder(rec.Body).Decode(&ar); err != nil {
		t.Fatalf("decode AnnotatedReport: %v", err)
	}
	if ar.Target != "web" {
		t.Errorf("AnnotatedReport.Target = %q, want %q", ar.Target, "web")
	}
	if ar.GeneratedAt == "" {
		t.Error("AnnotatedReport.GeneratedAt must not be empty")
	}
}

func TestDiagEndpoint_AllTargets_AreRoutable(t *testing.T) {
	// Verifies that every element of diag.AllTargets is recognised as valid
	// by the handler (does not return 400 "unknown target").  Each target
	// has no runner registered so we expect 404, not 400.
	h := newHandler(t, diag.NewDispatcher(nil))

	for _, target := range diag.AllTargets {
		t.Run(target.String(), func(t *testing.T) {
			body := diagBody(t, server.DiagRequest{Target: target.String()})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag", body))

			if rec.Code == http.StatusBadRequest {
				t.Errorf("target %q must not return 400 (would indicate an unrecognised target)", target)
			}
		})
	}
}

// ---- helpers ------------------------------------------------------------

// assertErrorField decodes the body and asserts the "error" key is non-empty.
func assertErrorField(t *testing.T, body io.Reader) {
	t.Helper()
	var resp server.ErrorResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decode ErrorResponse: %v", err)
	}
	if resp.Error == "" {
		t.Error("error field must not be empty")
	}
}
