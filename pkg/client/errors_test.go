package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantType string
		wantMsg  string
	}{
		{
			name:     "not_found",
			body:     `{"error":{"type":"not_found","message":"Agent not found","request_id":"req_123"}}`,
			wantType: "not_found",
			wantMsg:  "Agent not found",
		},
		{
			name:     "validation_error",
			body:     `{"error":{"type":"validation_error","message":"name is required"}}`,
			wantType: "validation_error",
			wantMsg:  "name is required",
		},
		{
			name:     "quota_exceeded",
			body:     `{"error":{"type":"quota_exceeded","message":"Agent limit reached."}}`,
			wantType: "quota_exceeded",
			wantMsg:  "Agent limit reached.",
		},
		{
			name:     "unauthorized",
			body:     `{"error":{"type":"unauthorized","message":"Invalid API key"}}`,
			wantType: "unauthorized",
			wantMsg:  "Invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: 400}
			apiErr := parseAPIError(resp, []byte(tt.body))
			assert.NotNil(t, apiErr)
			assert.Equal(t, tt.wantType, apiErr.Type)
			assert.Equal(t, tt.wantMsg, apiErr.Message)
		})
	}
}

func TestParseAPIErrorInvalidJSON(t *testing.T) {
	resp := &http.Response{StatusCode: 500}
	apiErr := parseAPIError(resp, []byte("not json"))
	assert.Nil(t, apiErr)
}

func TestParseAPIErrorNoType(t *testing.T) {
	resp := &http.Response{StatusCode: 400}
	apiErr := parseAPIError(resp, []byte(`{"error":{"message":"oops"}}`))
	assert.Nil(t, apiErr)
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{Type: "not_found", Message: "Agent not found", RequestID: "req_abc"}
	assert.Contains(t, err.Error(), "not_found")
	assert.Contains(t, err.Error(), "req_abc")
}

func TestAPIErrorStringNoRequestID(t *testing.T) {
	err := &APIError{Type: "not_found", Message: "Agent not found"}
	assert.Equal(t, "not_found: Agent not found", err.Error())
}

func TestFriendlyMessages(t *testing.T) {
	tests := []struct {
		errType  string
		contains string
	}{
		{"not_found", "not found"},
		{"validation_error", "Invalid request"},
		{"quota_exceeded", "Quota exceeded"},
		{"unauthorized", "API key"},
		{"rate_limited", "Rate limited"},
		{"authentication_error", "Authentication failed"},
		{"unknown_type", ""},
	}

	for _, tt := range tests {
		t.Run(tt.errType, func(t *testing.T) {
			err := &APIError{Type: tt.errType, Message: "test message"}
			msg := err.FriendlyMessage()
			if tt.contains != "" {
				assert.Contains(t, msg, tt.contains)
			}
			assert.NotEmpty(t, msg)
		})
	}
}

func TestHTTPErrorFriendlyMessages(t *testing.T) {
	tests := []struct {
		code     int
		contains string
	}{
		{502, "temporarily unavailable"},
		{503, "temporarily unavailable"},
		{504, "timed out"},
		{500, "Server error"},
		{418, "Unexpected response"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			err := &HTTPError{StatusCode: tt.code, Status: http.StatusText(tt.code)}
			assert.Contains(t, err.FriendlyMessage(), tt.contains)
		})
	}
}
