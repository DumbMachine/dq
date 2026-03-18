package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/pkg/types"
)

func CacheDir(connection string) string {
	return filepath.Join(config.ConfigDir(), "cache", connection)
}

func DiscoverCachePath(connection string) string {
	return filepath.Join(CacheDir(connection), "discover.json")
}

func LoadDiscover(connection string) (*types.DiscoverResult, error) {
	path := DiscoverCachePath(connection)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache: %w", err)
	}

	var result types.DiscoverResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}
	return &result, nil
}

func SaveDiscover(connection string, result *types.DiscoverResult) error {
	dir := CacheDir(connection)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	return os.WriteFile(DiscoverCachePath(connection), data, 0644)
}

func Invalidate(connection string) error {
	path := DiscoverCachePath(connection)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("invalidating cache: %w", err)
	}
	return nil
}
