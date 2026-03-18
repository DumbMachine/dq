package database

import (
	"fmt"
	"sync"
)

var (
	mu      sync.RWMutex
	drivers = make(map[string]Driver)
)

func Register(name string, driver Driver) {
	mu.Lock()
	defer mu.Unlock()
	drivers[name] = driver
}

func Get(name string) (Driver, error) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", name)
	}
	return d, nil
}

func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	return names
}
