//go:build !windows

package wincred

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"StreamSignal/internal/ports"
)

type Store struct{}

func NewStore() ports.SecretStore {
	return &Store{}
}

const serviceName = "StreamSignal"

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("credential storage is only supported on Windows and macOS")
	}

	// security find-generic-password -s "StreamSignal" -a <key> -w
	cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-s", serviceName, "-a", key, "-w")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 44 {
			return "", fmt.Errorf("secret %q not found", key)
		}
		return "", fmt.Errorf("read macOS keychain credential %q: %w", key, err)
	}

	return strings.TrimRight(string(out), "\n"), nil
}

func (s *Store) Put(ctx context.Context, key string, value string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("credential storage is only supported on Windows and macOS")
	}

	// security add-generic-password -s "StreamSignal" -a <key> -w <value> -U
	cmd := exec.CommandContext(ctx, "security", "add-generic-password", "-s", serviceName, "-a", key, "-w", value, "-U")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write macOS keychain credential %q: %s: %w", key, string(out), err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("credential storage is only supported on Windows and macOS")
	}

	// security delete-generic-password -s "StreamSignal" -a <key>
	cmd := exec.CommandContext(ctx, "security", "delete-generic-password", "-s", serviceName, "-a", key)
	if out, err := cmd.CombinedOutput(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && (exitErr.ExitCode() == 44 || strings.Contains(string(out), "could not be found")) {
			return nil
		}
		return fmt.Errorf("delete macOS keychain credential %q: %s: %w", key, string(out), err)
	}

	return nil
}
