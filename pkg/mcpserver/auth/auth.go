package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// KeyInfo holds information about an API key
type KeyInfo struct {
	Key       string
	OwnerID   uuid.UUID
	TenantID  uuid.UUID
	ExpiresAt *time.Time
	Scopes    []string // Optional: content:read, content:write, etc.
}

// Authenticator validates API keys and manages authentication
type Authenticator interface {
	// Validate checks if an API key is valid and returns associated key info
	Validate(ctx context.Context, apiKey string) (*KeyInfo, error)
}

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// keyInfoContextKey is the context key for storing KeyInfo
	keyInfoContextKey contextKey = "auth:keyInfo"
)

// WithKeyInfo adds KeyInfo to the context
func WithKeyInfo(ctx context.Context, keyInfo *KeyInfo) context.Context {
	return context.WithValue(ctx, keyInfoContextKey, keyInfo)
}

// GetKeyInfo retrieves KeyInfo from the context
func GetKeyInfo(ctx context.Context) (*KeyInfo, bool) {
	keyInfo, ok := ctx.Value(keyInfoContextKey).(*KeyInfo)
	return keyInfo, ok
}

// EnforceOwnership checks if the authenticated key has access to the specified owner
func EnforceOwnership(ctx context.Context, ownerID uuid.UUID) error {
	keyInfo, ok := GetKeyInfo(ctx)
	if !ok {
		return ErrUnauthorized
	}

	if keyInfo.OwnerID != ownerID {
		return ErrForbidden
	}

	return nil
}

// EnforceTenant checks if the authenticated key has access to the specified tenant
func EnforceTenant(ctx context.Context, tenantID uuid.UUID) error {
	keyInfo, ok := GetKeyInfo(ctx)
	if !ok {
		return ErrUnauthorized
	}

	if keyInfo.TenantID != uuid.Nil && keyInfo.TenantID != tenantID {
		return ErrForbidden
	}

	return nil
}
