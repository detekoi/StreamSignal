//go:build !windows

package wincred

import (
	"context"
	"fmt"

	"StreamSignal/internal/ports"
)

type Store struct{}

func NewStore() ports.SecretStore {
	return &Store{}
}

func (s *Store) Get(context.Context, string) (string, error) {
	return "", fmt.Errorf("Windows Credential Manager is only available on Windows")
}

func (s *Store) Put(context.Context, string, string) error {
	return fmt.Errorf("Windows Credential Manager is only available on Windows")
}

func (s *Store) Delete(context.Context, string) error {
	return fmt.Errorf("Windows Credential Manager is only available on Windows")
}
