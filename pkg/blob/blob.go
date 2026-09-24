package blob

import (
	"sync"
)

var (
	currentMu sync.RWMutex
	current   Provider
)

func SetCurrent(provider Provider) {
	currentMu.Lock()
	defer currentMu.Unlock()
	current = provider
}

func Current() Provider {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}
