package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/server"
)

// ?? SSE test helpers ??????????????????????????????????????????????????????

type sseEvent struct {
	Event string
	Data  string
}

// parseSSEEvents splits a raw SSE body into individual named events.
func parseSSEEvents(body string) []sseEvent {
	var events []sseEvent
	for _, msg := range strings.Split(body, "\n\n") {
		if strings.TrimSpace(msg) == "" {
			continue
		}
		var ev sseEvent
		for _, line := range strings.Split(msg, "\n") {
			if strings.HasPrefix(line, "event: ") {
				ev.Event = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				ev.Data = strings.TrimPrefix(line, "data: ")
			}
		}
		if ev.Event != "" {
			events = append(events, ev)
		}
	}
	return events
}

// stubProgressRunner is a Runner that emits a fixed set of progress stages
// before returning successfully (or a preset error).
type stubProgressRunner struct {
	stages []string
	err    error
}

func (s *stubProgressRunner) Run(_ context.Context, req diag.Request) error {
	if s.err != nil {
		return s.err
	}
	for _, stage := range s.stages {
		req.Emit(stage, "test: "+stage)
	}
	return nil
}

// postStream POSTs req to /api/diag/stream and returns status + raw body.
func postStream(t *testing.T, h http.Handler, req server.DiagRequest) (int, string) {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag/stream", bytes.NewReader(b)))
	return rec.Code, rec.Body.String()
}

// ?? Content-Type / header tests ???????????????????????????????????????????

// TestStreamEndpoint_ContentTypeIsSSE verifies that the streaming endpoint
// returns text/event-stream content type.
func TestStreamEndpoint_ContentTypeIsSSE(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetSMTP: &stubProgressRunner{stages: []string{"smtp"}},
	})
	h := newHandler(t, d)

	b, _ := json.Marshal(server.DiagRequest{Target: "smtp"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag/stream", bytes.NewReader(b)))

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
}

// TestStreamEndpoint_HTTPStatus200 verifies the streaming endpoint always
// returns 200 OK (SSE error events are sent in the body, not via status codes).
func TestStreamEndpoint_HTTPStatus200(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetWeb: &stubProgressRunner{},
	})
	h := newHandler(t, d)

	code, _ := postStream(t, h, server.DiagRequest{Target: "web"})
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
}

// ?? Progress event tests ??????????????????????????????????????????????????

// TestStreamEndpoint_EmitsProgressEvents verifies that progress events emitted
// by the runner appear in the SSE stream before the result event.
func TestStreamEndpoint_EmitsProgressEvents(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetFTP: &stubProgressRunner{stages: []string{"ftp", "ftp_result"}},
	})
	h := newHandler(t, d)

	_, body := postStream(t, h, server.DiagRequest{Target: "ftp"})
	events := parseSSEEvents(body)

	var progressCount int
	var hasResult bool
	for _, ev := range events {
		switch ev.Event {
		case "progress":
			progressCount++
		case "result":
			hasResult = true
		}
	}

	if progressCount != 2 {
		t.Errorf("progress events = %d, want 2; all events: %+v", progressCount, events)
	}
	if !hasResult {
		t.Errorf("no 'result' event in stream; all events: %+v", events)
	}
}

// TestStreamEndpoint_ProgressEventPayload verifies the JSON structure of a
// single progress event contains non-empty stage and message fields.
func TestStreamEndpoint_ProgressEventPayload(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetSFTP: &stubProgressRunner{stages: []string{"sftp"}},
	})
	h := newHandler(t, d)

	_, body := postStream(t, h, server.DiagRequest{Target: "sftp"})
	events := parseSSEEvents(body)

	for _, ev := range events {
		if ev.Event != "progress" {
			continue
		}
		var payload struct {
			Stage   string `json:"stage"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &payload); err != nil {
			t.Fatalf("progress payload parse error: %v (data=%q)", err, ev.Data)
		}
		if payload.Stage == "" {
			t.Error("progress event has empty stage")
		}
		if payload.Message == "" {
			t.Error("progress event has empty message")
		}
		return // first progress event verified ??sufficient
	}
	t.Error("no progress event found in stream")
}

// TestStreamEndpoint_ResultEventIsValidJSON verifies the result event payload
// is a parseable JSON object.
func TestStreamEndpoint_ResultEventIsValidJSON(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetIMAP: &stubProgressRunner{},
	})
	h := newHandler(t, d)

	_, body := postStream(t, h, server.DiagRequest{Target: "imap"})
	events := parseSSEEvents(body)

	for _, ev := range events {
		if ev.Event != "result" {
			continue
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(ev.Data), &out); err != nil {
			t.Fatalf("result payload is not valid JSON: %v (data=%q)", err, ev.Data)
		}
		return
	}
	t.Error("no 'result' event found in stream")
}

// ?? Error handling tests ??????????????????????????????????????????????????

// TestStreamEndpoint_InvalidBody verifies an SSE "error" event is emitted for
// a malformed JSON request body.
func TestStreamEndpoint_InvalidBody(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diag/stream",
		strings.NewReader("{bad json")))

	events := parseSSEEvents(rec.Body.String())
	if len(events) == 0 || events[0].Event != "error" {
		t.Errorf("expected an 'error' event; got: %+v", events)
	}
}

// TestStreamEndpoint_UnknownTarget verifies an SSE "error" event is emitted
// for an unrecognised target string.
func TestStreamEndpoint_UnknownTarget(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))

	_, body := postStream(t, h, server.DiagRequest{Target: "bogus"})
	events := parseSSEEvents(body)

	if len(events) == 0 || events[0].Event != "error" {
		t.Errorf("expected an 'error' event; got: %+v", events)
	}
}

// TestStreamEndpoint_RunnerNotFound verifies an SSE "error" event is emitted
// when no runner is registered for the requested target.
func TestStreamEndpoint_RunnerNotFound(t *testing.T) {
	// Empty dispatcher ??no runners registered.
	h := newHandler(t, diag.NewDispatcher(nil))

	_, body := postStream(t, h, server.DiagRequest{Target: "web"})
	events := parseSSEEvents(body)

	var hasError bool
	for _, ev := range events {
		if ev.Event == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected an 'error' event; got: %+v", events)
	}
}

// TestStreamEndpoint_RunnerError verifies an SSE "error" event is emitted when
// the runner itself returns a non-nil error.
func TestStreamEndpoint_RunnerError(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetPOP: &stubProgressRunner{err: diag.ErrRunnerNotFound},
	})
	h := newHandler(t, d)

	_, body := postStream(t, h, server.DiagRequest{Target: "pop"})
	events := parseSSEEvents(body)

	var hasError bool
	for _, ev := range events {
		if ev.Event == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected an 'error' event when runner returns error; got: %+v", events)
	}
}

// ?? Route isolation ???????????????????????????????????????????????????????

// TestStreamEndpoint_IsDistinctFromDiagEndpoint verifies that both
// POST /api/diag and POST /api/diag/stream are reachable simultaneously and
// return the expected content types.
func TestStreamEndpoint_IsDistinctFromDiagEndpoint(t *testing.T) {
	d := diag.NewDispatcher(map[diag.Target]diag.Runner{
		diag.TargetWeb: &stubProgressRunner{},
	})
	h := newHandler(t, d)

	// POST /api/diag must return JSON (not SSE).
	b, _ := json.Marshal(server.DiagRequest{Target: "web"})
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest(http.MethodPost, "/api/diag", bytes.NewReader(b)))
	if rec1.Code != http.StatusOK {
		t.Errorf("/api/diag: status = %d, want 200", rec1.Code)
	}
	if strings.Contains(rec1.Header().Get("Content-Type"), "event-stream") {
		t.Error("/api/diag must not return text/event-stream")
	}

	// POST /api/diag/stream must return SSE.
	b, _ = json.Marshal(server.DiagRequest{Target: "web"})
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/api/diag/stream", bytes.NewReader(b)))
	if !strings.Contains(rec2.Header().Get("Content-Type"), "event-stream") {
		t.Errorf("/api/diag/stream Content-Type = %q, want text/event-stream",
			rec2.Header().Get("Content-Type"))
	}
}
