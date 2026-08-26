//go:build windows

package auth

import "sync"

var secretsMu sync.Mutex

// withSecretsLock serializes secrets file access within this process only.
func withSecretsLock(fn func() error) error {
	secretsMu.Lock()
	defer secretsMu.Unlock()
	return fn()
}
