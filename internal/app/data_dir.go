package app

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	dataDir := filepath.Join(configDir, "StreamSignal")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", fmt.Errorf("create app data dir: %w", err)
	}

	return dataDir, nil
}
