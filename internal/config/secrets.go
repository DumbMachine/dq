package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const keyringService = "dq"

// ResolvePassword resolves a password value from various sources.
// Supported formats:
//   - "env:VAR_NAME" → reads from environment variable
//   - "keyring:name" → reads from OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
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
		val, err := keyring.Get(keyringService, name)
		if err != nil {
			return "", false, fmt.Errorf("reading password from keyring for %q: %w", name, err)
		}
		return val, false, nil
	}

	// Plain text password
	return raw, true, nil
}

// StoreInKeyring stores a password in the OS keychain under the given name.
func StoreInKeyring(name, password string) error {
	return keyring.Set(keyringService, name, password)
}

// DeleteFromKeyring removes a password from the OS keychain.
// Returns nil if the entry does not exist.
func DeleteFromKeyring(name string) error {
	err := keyring.Delete(keyringService, name)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
