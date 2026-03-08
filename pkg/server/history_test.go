package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-pathprobe/pkg/diag"
	"go-pathprobe/pkg/server"
)

// ---- GET /api/history -------------------------------------------------------

func TestHistoryList_EmptyReturnsEmptyArray(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))

	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var items []server.HistoryListItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

// ---- GET /api/history/{id} --------------------------------------------------

func TestHistoryDetail_MissingIDReturns404(t *testing.T) {
	h := newHandler(t, diag.NewDispatcher(nil))

	req := httptest.NewRequest(http.MethodGet, "/api/history/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var resp server.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(resp.Error, "nonexistent") {
		t.Errorf("error message %q does not mention the missing id", resp.Error)
	}
}
