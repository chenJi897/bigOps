package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func resolveAgentStateFile(configPath, configuredStateFile string) string {
	if strings.TrimSpace(configuredStateFile) != "" {
		return configuredStateFile
	}
	configDir := filepath.Dir(configPath)
	if configDir == "." || configDir == "" {
		configDir = "config"
	}
	return filepath.Join(configDir, ".agent-id")
}

func resolveAgentID(configuredID, stateFile string) (string, error) {
	if strings.TrimSpace(configuredID) != "" {
		return strings.TrimSpace(configuredID), nil
	}
	if data, err := os.ReadFile(stateFile); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id, nil
		}
	}

	id := uuid.NewString()
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(stateFile, []byte(id+"\n"), 0o600); err != nil {
		return "", err
	}
	return id, nil
}
