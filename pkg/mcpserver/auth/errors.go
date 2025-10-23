package auth

import "errors"

var (
	// ErrInvalidAPIKey is returned when the API key is not found
	ErrInvalidAPIKey = errors.New("invalid or missing API key")

	// ErrExpiredAPIKey is returned when the API key has expired
	ErrExpiredAPIKey = errors.New("API key has expired")

	// ErrUnauthorized is returned when authentication is required but not provided
	ErrUnauthorized = errors.New("authentication required")

	// ErrForbidden is returned when the authenticated user lacks permission
	ErrForbidden = errors.New("access denied")
)
