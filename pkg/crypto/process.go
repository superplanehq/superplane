package crypto

import (
	"errors"
	"sync"
)

var processEncryptor struct {
	sync.RWMutex
	value Encryptor
}

// SetProcessEncryptor makes the server's initialized encryptor available to
// code paths that run outside dependency-injected request and worker contexts.
func SetProcessEncryptor(encryptor Encryptor) {
	processEncryptor.Lock()
	processEncryptor.value = encryptor
	processEncryptor.Unlock()
}

func ProcessEncryptor() (Encryptor, error) {
	processEncryptor.RLock()
	encryptor := processEncryptor.value
	processEncryptor.RUnlock()
	if encryptor != nil {
		return encryptor, nil
	}
	return nil, errors.New("process encryptor is not initialized")
}
