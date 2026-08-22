package media

import (
	"testing"
	"time"
)

func TestSelectionStoreFlow(t *testing.T) {
	store := NewSelectionStore(time.Minute)
	id, err := store.Create(Selection{UserID: "user-1", Quality: ""})
	if err != nil {
		t.Fatalf("create selection: %v", err)
	}
	if _, ok := store.SetQuality(id, "user-2", "251"); ok {
		t.Fatal("expected another user to be rejected")
	}
	selection, ok := store.SetQuality(id, "user-1", "251")
	if !ok || selection.Quality != "251" {
		t.Fatalf("set quality failed: %#v ok=%t", selection, ok)
	}
	selection, ok = store.Take(id, "user-1")
	if !ok || selection.Quality != "251" {
		t.Fatalf("take failed: %#v ok=%t", selection, ok)
	}
	if _, ok := store.Take(id, "user-1"); ok {
		t.Fatal("expected selection to be one-time")
	}
}

func TestSelectionStoreExpires(t *testing.T) {
	store := NewSelectionStore(time.Millisecond)
	id, err := store.Create(Selection{UserID: "user-1"})
	if err != nil {
		t.Fatalf("create selection: %v", err)
	}
	time.Sleep(3 * time.Millisecond)
	if _, ok := store.Get(id); ok {
		t.Fatal("expected expired selection")
	}
}
