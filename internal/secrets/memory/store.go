package memory

import (
	"context"
	"fmt"
	"sync"

	"StreamSignal/internal/ports"
)

type Store struct {
	mu    sync.RWMutex
	items map[string]string
}

func NewStore() ports.SecretStore {
	return &Store{
		items: map[string]string{},
	}
}

func (s *Store) Get(_ context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.items[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found", key)
	}
	return value, nil
}

func (s *Store) Put(_ context.Context, key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = value
	return nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, key)
	return nil
}
