//go:build !windows

package wincred

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestMacKeychainStore(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS Keychain test on non-darwin OS")
	}

	store := NewStore()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("streamsignal/test/%d", time.Now().UnixNano())
	value := "supersecretvalue123"

	// 1. Clean up key in case it exists from a failed run
	_ = store.Delete(ctx, key)

	// 2. Try to get non-existent key, should fail with "not found"
	_, err := store.Get(ctx, key)
	if err == nil {
		t.Fatalf("expected error getting non-existent key, got nil")
	}
	expectedNotFoundMsg := fmt.Sprintf("secret %q not found", key)
	if err.Error() != expectedNotFoundMsg {
		t.Errorf("expected error %q, got %q", expectedNotFoundMsg, err.Error())
	}

	// 3. Put value
	err = store.Put(ctx, key, value)
	if err != nil {
		t.Fatalf("failed to store secret: %v", err)
	}

	// 4. Get value and verify
	gotValue, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if gotValue != value {
		t.Errorf("expected value %q, got %q", value, gotValue)
	}

	// 5. Update value (verifying the -U flag update functionality)
	newValue := "anothersecret456"
	err = store.Put(ctx, key, newValue)
	if err != nil {
		t.Fatalf("failed to update secret: %v", err)
	}

	gotNewValue, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get updated secret: %v", err)
	}
	if gotNewValue != newValue {
		t.Errorf("expected updated value %q, got %q", newValue, gotNewValue)
	}

	// 6. Delete key
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete secret: %v", err)
	}

	// 7. Try to get deleted key, should fail with "not found"
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Errorf("expected error getting deleted key, got nil")
	}

	// 8. Delete again (should not fail)
	err = store.Delete(ctx, key)
	if err != nil {
		t.Errorf("expected second delete to be a no-op, got error: %v", err)
	}
}

func TestMacKeychainStore_PreservesWhitespace(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS Keychain test on non-darwin OS")
	}

	store := NewStore()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("streamsignal/test/ws/%d", time.Now().UnixNano())
	defer store.Delete(ctx, key)

	// Value with leading and trailing spaces that must be preserved.
	value := " token-with-spaces "

	err := store.Put(ctx, key, value)
	if err != nil {
		t.Fatalf("failed to store secret with whitespace: %v", err)
	}

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get secret with whitespace: %v", err)
	}
	if got != value {
		t.Errorf("whitespace not preserved: expected %q, got %q", value, got)
	}
}
