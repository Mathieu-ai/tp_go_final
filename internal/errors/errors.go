package errors

import (
	"errors"
	"fmt"
)

// Custom error types for the URL shortener application

// ErrShortCodeNotFound is returned when a short code doesn't exist in the database
var ErrShortCodeNotFound = errors.New("short code not found")

// ErrInvalidURL is returned when the provided URL is invalid
var ErrInvalidURL = errors.New("invalid URL format")

// ErrDatabaseConnection is returned when database connection fails
var ErrDatabaseConnection = errors.New("database connection failed")

// ErrShortCodeGenerationFailed is returned when we can't generate a unique short code
var ErrShortCodeGenerationFailed = errors.New("failed to generate unique short code")

// ErrClickRecordingFailed is returned when click recording fails
type ErrClickRecordingFailed struct {
	LinkID uint
	Reason string
}

func (e ErrClickRecordingFailed) Error() string {
	return fmt.Sprintf("failed to record click for link %d: %s", e.LinkID, e.Reason)
}

// ErrURLCheckFailed is returned when URL health check fails
type ErrURLCheckFailed struct {
	URL    string
	Reason string
}

func (e ErrURLCheckFailed) Error() string {
	return fmt.Sprintf("failed to check URL %s: %s", e.URL, e.Reason)
}

// ErrConfigLoad is returned when configuration loading fails
type ErrConfigLoad struct {
	Path   string
	Reason string
}

func (e ErrConfigLoad) Error() string {
	return fmt.Sprintf("failed to load config from %s: %s", e.Path, e.Reason)
}

// ErrInvalidShortCode is returned when the short code format is invalid
var ErrInvalidShortCode = errors.New("invalid short code format")
