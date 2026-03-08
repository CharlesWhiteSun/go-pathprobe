// Package store defines the persistence interface for diagnostic history.
package store

import (
	"time"

	"go-pathprobe/pkg/report"
)

// HistoryEntry wraps an AnnotatedReport with an opaque string ID and the
// timestamp at which it was stored.
type HistoryEntry struct {
	ID        string
	CreatedAt time.Time
	Report    *report.AnnotatedReport
}

// Store is the read/write interface for diagnostic history entries.
// Implementations are expected to be safe for concurrent use.
type Store interface {
	// Save persists entry and returns the same value (for convenient chaining).
	Save(entry HistoryEntry) HistoryEntry

	// List returns all stored entries, newest first.
	List() []HistoryEntry

	// Get retrieves a single entry by its ID.
	// Returns the zero value and false when no entry with that ID exists.
	Get(id string) (HistoryEntry, bool)
}
