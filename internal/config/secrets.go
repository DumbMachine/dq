package config

import (
	"fmt"
	"os"
	"strings"
)

// ResolvePassword resolves a password value from various sources.
// Supported formats:
//   - "env:VAR_NAME" → reads from environment variable
//   - "keyring:name" → reads from OS keychain (not yet implemented)
//   - plain text → returns as-is (warns)
func ResolvePassword(raw string) (string, bool, error) {
	if raw == "" {
		return "", false, nil
	}

	if strings.HasPrefix(raw, "env:") {
		envVar := strings.TrimPrefix(raw, "env:")
		val := os.Getenv(envVar)
		if val == "" {
			return "", false, fmt.Errorf("environment variable %s is not set", envVar)
		}
		return val, false, nil
	}

	if strings.HasPrefix(raw, "keyring:") {
		name := strings.TrimPrefix(raw, "keyring:")
		return "", false, fmt.Errorf("keyring support not yet implemented (requested: %s)", name)
	}

	// Plain text password
	return raw, true, nil
}
