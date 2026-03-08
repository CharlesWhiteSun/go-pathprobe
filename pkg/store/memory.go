package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go-pathprobe/pkg/report"
)

// DefaultMaxHistory is the default capacity of an in-memory store.
// Oldest entries are evicted once the limit is reached.
const DefaultMaxHistory = 100

// MemoryStore is a thread-safe, bounded in-memory implementation of Store.
// When the store is at capacity the oldest entry is evicted (FIFO).
type MemoryStore struct {
	mu      sync.RWMutex
	entries []HistoryEntry
	maxSize int
	seq     atomic.Uint64
}

// NewMemoryStore returns a MemoryStore that retains at most maxSize entries.
// If maxSize <= 0 it uses DefaultMaxHistory.
func NewMemoryStore(maxSize int) *MemoryStore {
	if maxSize <= 0 {
		maxSize = DefaultMaxHistory
	}
	return &MemoryStore{maxSize: maxSize}
}

// Save stores the entry, evicting the oldest one when at capacity.
// The entry's ID is set by the store; any caller-supplied ID is ignored.
func (m *MemoryStore) Save(e HistoryEntry) HistoryEntry {
	e.ID = fmt.Sprintf("%d", m.seq.Add(1))
	e.CreatedAt = time.Now()
	if e.Report == nil {
		e.Report = &report.AnnotatedReport{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.entries) >= m.maxSize {
		// Evict oldest (index 0); shift remaining left.
		copy(m.entries, m.entries[1:])
		m.entries = m.entries[:len(m.entries)-1]
	}
	m.entries = append(m.entries, e)
	return e
}

// List returns a snapshot of all entries, newest first.
func (m *MemoryStore) List() []HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]HistoryEntry, len(m.entries))
	for i, e := range m.entries {
		out[len(m.entries)-1-i] = e
	}
	return out
}

// Get retrieves a single entry by its opaque ID.
func (m *MemoryStore) Get(id string) (HistoryEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, e := range m.entries {
		if e.ID == id {
			return e, true
		}
	}
	return HistoryEntry{}, false
}
