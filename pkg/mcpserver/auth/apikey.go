package auth

import (
	"context"
	"sync"
	"time"
)

// APIKeyAuthenticator implements Authenticator using API keys
type APIKeyAuthenticator struct {
	mu   sync.RWMutex
	keys map[string]*KeyInfo
}

// NewAPIKeyAuthenticator creates a new API key authenticator
func NewAPIKeyAuthenticator() *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		keys: make(map[string]*KeyInfo),
	}
}

// AddKey registers a new API key
func (a *APIKeyAuthenticator) AddKey(keyInfo *KeyInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.keys[keyInfo.Key] = keyInfo
}

// RemoveKey removes an API key
func (a *APIKeyAuthenticator) RemoveKey(apiKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.keys, apiKey)
}

// Validate checks if an API key is valid
func (a *APIKeyAuthenticator) Validate(ctx context.Context, apiKey string) (*KeyInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	keyInfo, ok := a.keys[apiKey]
	if !ok {
		return nil, ErrInvalidAPIKey
	}

	// Check expiration
	if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
		return nil, ErrExpiredAPIKey
	}

	return keyInfo, nil
}

// ListKeys returns all registered API keys (for admin purposes)
func (a *APIKeyAuthenticator) ListKeys() []*KeyInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	keys := make([]*KeyInfo, 0, len(a.keys))
	for _, keyInfo := range a.keys {
		keys = append(keys, keyInfo)
	}
	return keys
}
