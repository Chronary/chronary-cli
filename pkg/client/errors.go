package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents a structured error from the Chronary API.
type APIError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`

	// Plan / quota guidance, present on plan_required and quota_exceeded errors.
	UpgradeURL   string `json:"upgrade_url,omitempty"`
	DocsURL      string `json:"docs_url,omitempty"`
	RequiredPlan string `json:"required_plan,omitempty"`
	UpgradePlan  string `json:"upgrade_plan,omitempty"`
	Feature      string `json:"feature,omitempty"`
	ContactURL   string `json:"contact_url,omitempty"`
	Metric       string `json:"metric,omitempty"`
	Limit        int    `json:"limit,omitempty"`
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
		msg := "Quota exceeded"
		if e.Metric != "" {
			msg += fmt.Sprintf(" for %s", e.Metric)
			if e.Limit > 0 {
				msg += fmt.Sprintf(" (limit %d)", e.Limit)
			}
		}
		msg += ". Quotas reset at the start of each UTC calendar month; upgrade your plan to continue sooner."
		return msg + e.upgradeHint()
	case "plan_required":
		msg := "This feature is not included in your current plan"
		if e.RequiredPlan != "" {
			msg = fmt.Sprintf("This feature requires the %s plan", e.RequiredPlan)
		}
		if e.Feature != "" {
			msg += fmt.Sprintf(" (%s)", e.Feature)
		}
		return msg + "." + e.upgradeHint()
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

// upgradeHint renders the "how to fix this" lines for plan/quota errors:
// the upgrade URL when the API supplied one, plus the agent-friendly path of
// asking a human to approve an upgrade.
func (e *APIError) upgradeHint() string {
	var b strings.Builder
	if e.UpgradeURL != "" {
		if e.UpgradePlan != "" {
			fmt.Fprintf(&b, "\nUpgrade to %s: %s", e.UpgradePlan, e.UpgradeURL)
		} else {
			fmt.Fprintf(&b, "\nUpgrade: %s", e.UpgradeURL)
		}
	}
	if e.ContactURL != "" {
		fmt.Fprintf(&b, "\nContact: %s", e.ContactURL)
	}
	if e.DocsURL != "" {
		fmt.Fprintf(&b, "\nDocs: %s", e.DocsURL)
	}
	b.WriteString("\nRun `chronary billing request-upgrade` to ask your human to approve an upgrade.")
	return b.String()
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
