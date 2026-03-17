package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

// DiagHandler serves POST /api/diag.
// It decodes the request body, delegates to diagPipeline.runDiag, and returns
// a JSON-encoded AnnotatedReport.
type DiagHandler struct {
	pipeline diagPipeline
}

// ServeHTTP handles POST /api/diag.
func (h *DiagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req DiagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	ar, err := h.pipeline.runDiag(r.Context(), req, nil)
	if err != nil {
		var pe *pipelineError
		errors.As(err, &pe)
		writeError(w, pe.code, pe.msg)
		return
	}

	writeJSON(w, http.StatusOK, ar)
}
