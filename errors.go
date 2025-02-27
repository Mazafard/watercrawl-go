package watercrawl

import (
	"fmt"
)

// APIError represents an error returned by the WaterCrawl API
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("watercrawl: API error (status %d): %s", e.StatusCode, e.Message)
}

// ValidationError represents a validation error in the SDK
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("watercrawl: validation error: %s: %s", e.Field, e.Message)
}

// TimeoutError represents a timeout error in the SDK
type TimeoutError struct {
	Operation string
	Message   string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("watercrawl: timeout error during %s: %s", e.Operation, e.Message)
} 