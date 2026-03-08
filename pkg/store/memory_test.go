package store_test

import (
	"fmt"
	"testing"

	"go-pathprobe/pkg/report"
	"go-pathprobe/pkg/store"
)

func makeEntry(host string) store.HistoryEntry {
	return store.HistoryEntry{
		Report: &report.AnnotatedReport{Host: host},
	}
}

// ---- Save / List / Get --------------------------------------------------

func TestMemoryStore_SaveAssignsID(t *testing.T) {
	s := store.NewMemoryStore(10)
	e := s.Save(makeEntry("a.com"))
	if e.ID == "" {
		t.Error("Save must assign a non-empty ID")
	}
	if e.CreatedAt.IsZero() {
		t.Error("Save must set CreatedAt")
	}
}

func TestMemoryStore_ListNewestFirst(t *testing.T) {
	s := store.NewMemoryStore(10)
	for i := 0; i < 3; i++ {
		s.Save(makeEntry(fmt.Sprintf("host%d.com", i)))
	}
	got := s.List()
	if len(got) != 3 {
		t.Fatalf("List() len = %d, want 3", len(got))
	}
	// Newest (host2) should be first.
	if got[0].Report.Host != "host2.com" {
		t.Errorf("List()[0].Host = %q, want host2.com", got[0].Report.Host)
	}
	if got[2].Report.Host != "host0.com" {
		t.Errorf("List()[2].Host = %q, want host0.com", got[2].Report.Host)
	}
}

func TestMemoryStore_GetByID(t *testing.T) {
	s := store.NewMemoryStore(10)
	saved := s.Save(makeEntry("example.com"))
	got, ok := s.Get(saved.ID)
	if !ok {
		t.Fatalf("Get(%q) not found", saved.ID)
	}
	if got.Report.Host != "example.com" {
		t.Errorf("Got host = %q, want example.com", got.Report.Host)
	}
}

func TestMemoryStore_GetMissing(t *testing.T) {
	s := store.NewMemoryStore(10)
	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("Get on empty store must return false")
	}
}

// ---- FIFO eviction ------------------------------------------------------

func TestMemoryStore_FIFOEviction(t *testing.T) {
	const cap = 3
	s := store.NewMemoryStore(cap)
	var saved [5]store.HistoryEntry
	for i := range saved {
		saved[i] = s.Save(makeEntry(fmt.Sprintf("h%d.com", i)))
	}

	list := s.List()
	if len(list) != cap {
		t.Fatalf("after eviction List() len = %d, want %d", len(list), cap)
	}

	// Oldest two entries (index 0, 1) must have been evicted.
	_, ok0 := s.Get(saved[0].ID)
	_, ok1 := s.Get(saved[1].ID)
	if ok0 || ok1 {
		t.Error("evicted entries should not be findable by ID")
	}

	// Newest three must still be present.
	for i := 2; i <= 4; i++ {
		if _, ok := s.Get(saved[i].ID); !ok {
			t.Errorf("entry %d missing after partial eviction", i)
		}
	}
}

// ---- DefaultMaxHistory default ------------------------------------------

func TestNewMemoryStore_ZeroUsesDefault(t *testing.T) {
	s := store.NewMemoryStore(0)
	for i := 0; i < store.DefaultMaxHistory+5; i++ {
		s.Save(makeEntry("x.com"))
	}
	if len(s.List()) != store.DefaultMaxHistory {
		t.Errorf("List() len = %d, want %d", len(s.List()), store.DefaultMaxHistory)
	}
}
