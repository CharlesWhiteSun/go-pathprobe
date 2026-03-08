package server

import (
	"net/http"

	"go-pathprobe/pkg/store"
)

// HistoryHandler serves GET /api/history.
// It returns a JSON array of HistoryListItem records, newest first.
type HistoryHandler struct {
	store store.Store
}

func (h *HistoryHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	entries := h.store.List()
	items := make([]HistoryListItem, len(entries))
	for i, e := range entries {
		items[i] = HistoryListItem{
			ID:        e.ID,
			CreatedAt: e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if e.Report != nil {
			items[i].Target = e.Report.Target
			items[i].Host = e.Report.Host
			items[i].GeneratedAt = e.Report.GeneratedAt
		}
	}
	writeJSON(w, http.StatusOK, items)
}

// HistoryDetailHandler serves GET /api/history/{id}.
// It returns the full AnnotatedReport for the matching entry.
type HistoryDetailHandler struct {
	store store.Store
}

func (h *HistoryDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing history id")
		return
	}
	e, ok := h.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "history entry not found: "+id)
		return
	}
	writeJSON(w, http.StatusOK, e.Report)
}
