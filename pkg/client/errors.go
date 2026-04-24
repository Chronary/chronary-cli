package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError represents a structured error from the Chronary API.
type APIError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// apiErrorEnvelope is the JSON envelope: {"error": {...}}
type apiErrorEnvelope struct {
	Error APIError `json:"error"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%s: %s (request_id: %s)", e.Type, e.Message, e.RequestID)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// FriendlyMessage returns a user-facing message for the error type.
func (e *APIError) FriendlyMessage() string {
	switch e.Type {
	case "not_found":
		return "Resource not found. Check the ID and try again."
	case "validation_error":
		return fmt.Sprintf("Invalid request: %s", e.Message)
	case "quota_exceeded":
		return "Quota exceeded. Upgrade your plan or wait for the next billing cycle."
	case "unauthorized":
		return "Invalid or missing API key. Run `chronary auth login` to configure."
	case "rate_limited":
		return "Rate limited. Wait a moment and try again."
	case "authentication_error":
		return "Authentication failed. Run `chronary auth login` to reconfigure your API key."
	default:
		return e.Message
	}
}

// parseAPIError attempts to parse an API error from an HTTP response.
// Returns nil if the response is not an error or can't be parsed.
func parseAPIError(resp *http.Response, body []byte) *APIError {
	var envelope apiErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil
	}
	if envelope.Error.Type == "" {
		return nil
	}
	return &envelope.Error
}

// HTTPError represents a non-JSON HTTP error (e.g., 502, 503).
type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Status)
}

// FriendlyMessage returns a user-facing message for HTTP status codes.
func (e *HTTPError) FriendlyMessage() string {
	switch {
	case e.StatusCode == 502 || e.StatusCode == 503:
		return "The API is temporarily unavailable. Try again in a moment."
	case e.StatusCode == 504:
		return "The API request timed out. Try again."
	case e.StatusCode >= 500:
		return fmt.Sprintf("Server error (%d). If this persists, check https://status.chronary.ai", e.StatusCode)
	default:
		return fmt.Sprintf("Unexpected response: HTTP %d", e.StatusCode)
	}
}
