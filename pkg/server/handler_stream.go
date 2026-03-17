package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go-pathprobe/pkg/diag"
)

// StreamDiagHandler serves POST /api/diag/stream.
// It runs the same diagnostic pipeline as DiagHandler, but streams each
// ProgressEvent as a Server-Sent Events "progress" event and emits the final
// AnnotatedReport (or an error) as a "result" / "error" event.
//
// SSE event shape:
//
//	event: progress
//	data: {"stage":"network","message":"Probing 1 port(s) on example.com …"}
//
//	event: result
//	data: {<AnnotatedReport JSON>}
//
//	event: error
//	data: {"error":"<message>"}
type StreamDiagHandler struct {
	pipeline diagPipeline
}

// ServeHTTP handles POST /api/diag/stream.
func (h *StreamDiagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported by server")
		return
	}

	// SSE headers must be set before the first write.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // prevent nginx / proxy buffering
	w.WriteHeader(http.StatusOK)

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req DiagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSSEError(w, flusher, "invalid request body: "+err.Error())
		return
	}

	hook := func(ev diag.ProgressEvent) {
		writeSSEEvent(w, flusher, "progress", ev)
	}

	ar, err := h.pipeline.runDiag(r.Context(), req, hook)
	if err != nil {
		var pe *pipelineError
		errors.As(err, &pe)
		writeSSEError(w, flusher, pe.msg)
		return
	}

	writeSSEEvent(w, flusher, "result", ar)
}

// writeSSEEvent encodes payload as JSON and writes a named SSE event, then
// flushes the response buffer so the client receives the data immediately.
func writeSSEEvent(w http.ResponseWriter, f http.Flusher, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	f.Flush()
}

// writeSSEError writes an SSE "error" event with a plain message.
func writeSSEError(w http.ResponseWriter, f http.Flusher, msg string) {
	writeSSEEvent(w, f, "error", map[string]string{"error": msg})
}
