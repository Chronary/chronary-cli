package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// These guard against the CLI response models silently dropping live API
// fields from `-o json` output (the P3 surface-audit finding). Each test
// unmarshals an API-shaped response and asserts the previously-missing fields
// round-trip with the correct (mostly camelCase) JSON tags.

func TestCalendarPreservesLiveFields(t *testing.T) {
	const body = `{
		"id": "cal_1",
		"orgId": "org_1",
		"agentId": "agt_1",
		"name": "Work",
		"timezone": "UTC",
		"agent_status": "working",
		"ical_url": "https://api.chronary.ai/v1/ical/feeds/tok.ics",
		"externalId": "ext_1",
		"provider": "google",
		"default_reminders": [10, 1440],
		"metadata": {},
		"deletedAt": null,
		"createdAt": "2026-04-14T00:00:00Z",
		"updatedAt": "2026-04-15T00:00:00Z"
	}`
	var cal Calendar
	require.NoError(t, json.Unmarshal([]byte(body), &cal))

	require.Equal(t, "org_1", cal.OrgID)
	require.Equal(t, "working", cal.AgentStatus)
	require.NotNil(t, cal.ExternalID)
	require.Equal(t, "ext_1", *cal.ExternalID)
	require.NotNil(t, cal.Provider)
	require.Equal(t, "google", *cal.Provider)
	require.False(t, cal.UpdatedAt.IsZero(), "updatedAt must deserialize")
}

func TestWebhookPreservesCircuitBreakerFields(t *testing.T) {
	const body = `{
		"id": "whk_1",
		"orgId": "org_1",
		"url": "https://example.com/hook",
		"events": ["event.created"],
		"active": true,
		"consecutiveFailures": 2,
		"firstFailureAt": "2026-04-14T00:00:00Z",
		"createdAt": "2026-04-14T00:00:00Z"
	}`
	var wh Webhook
	require.NoError(t, json.Unmarshal([]byte(body), &wh))

	require.Equal(t, "org_1", wh.OrgID)
	require.Equal(t, 2, wh.ConsecutiveFailures)
	require.NotNil(t, wh.FirstFailureAt)
	require.Equal(t, "2026-04-14T00:00:00Z", *wh.FirstFailureAt)
}

func TestAgentPreservesOrgID(t *testing.T) {
	const body = `{
		"id": "agt_1",
		"orgId": "org_1",
		"name": "Bot",
		"type": "ai",
		"status": "active",
		"metadata": {},
		"createdAt": "2026-04-14T00:00:00Z",
		"updatedAt": "2026-04-14T00:00:00Z"
	}`
	var agent Agent
	require.NoError(t, json.Unmarshal([]byte(body), &agent))

	require.Equal(t, "org_1", agent.OrgID)
	require.False(t, agent.CreatedAt.IsZero())
}
