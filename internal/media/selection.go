package media

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type Selection struct {
	Request   Request
	Info      Info
	MessageID string
	ChannelID string
	UserID    string
	CreatedAt time.Time
	Quality   string
}

type SelectionStore struct {
	mu    sync.Mutex
	items map[string]*Selection
	TTL   time.Duration
}

func NewSelectionStore(ttl time.Duration) *SelectionStore {
	return &SelectionStore{items: make(map[string]*Selection), TTL: ttl}
}

func (s *SelectionStore) Create(selection Selection) (string, error) {
	nonce := make([]byte, 9)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("create selection ID: %w", err)
	}
	id := hex.EncodeToString(nonce)
	selection.CreatedAt = time.Now()
	s.mu.Lock()
	s.items[id] = &selection
	s.mu.Unlock()
	return id, nil
}

func (s *SelectionStore) Get(id string) (Selection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	selection, ok := s.items[id]
	if !ok {
		return Selection{}, false
	}
	if s.TTL > 0 && time.Since(selection.CreatedAt) > s.TTL {
		delete(s.items, id)
		return Selection{}, false
	}
	return *selection, true
}

func (s *SelectionStore) SetQuality(id, userID, quality string) (Selection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	selection, ok := s.items[id]
	if !ok || (selection.UserID != "" && selection.UserID != userID) {
		return Selection{}, false
	}
	if s.TTL > 0 && time.Since(selection.CreatedAt) > s.TTL {
		delete(s.items, id)
		return Selection{}, false
	}
	selection.Quality = quality
	return *selection, true
}

func (s *SelectionStore) Take(id, userID string) (Selection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	selection, ok := s.items[id]
	if !ok || (selection.UserID != "" && selection.UserID != userID) {
		return Selection{}, false
	}
	if s.TTL > 0 && time.Since(selection.CreatedAt) > s.TTL {
		delete(s.items, id)
		return Selection{}, false
	}
	if selection.Quality == "" {
		return Selection{}, false
	}
	result := *selection
	delete(s.items, id)
	return result, true
}

func (s *SelectionStore) Delete(id, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	selection, ok := s.items[id]
	if !ok || (selection.UserID != "" && selection.UserID != userID) {
		return false
	}
	delete(s.items, id)
	return true
}

func (s *SelectionStore) SetMessageID(id, userID, messageID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	selection, ok := s.items[id]
	if !ok || (selection.UserID != "" && selection.UserID != userID) {
		return false
	}
	selection.MessageID = messageID
	return true
}
