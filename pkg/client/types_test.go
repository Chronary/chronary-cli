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

// Nullable response fields must render as JSON `null` (not be omitted) so
// `-o json` mirrors the REST response shape exactly. Guards against omitempty
// creeping back onto these pointer fields.
func TestCalendarNullableFieldsRenderAsNull(t *testing.T) {
	// All nullable pointers nil — as REST returns for an org-level calendar
	// with no external provider and not soft-deleted.
	cal := Calendar{ID: "cal_1", OrgID: "org_1", Name: "Work", Timezone: "UTC"}
	out, err := json.Marshal(cal)
	require.NoError(t, err)
	s := string(out)
	for _, key := range []string{`"agentId":null`, `"externalId":null`, `"provider":null`, `"deletedAt":null`} {
		require.Contains(t, s, key, "nullable field must serialize as null, not be omitted")
	}
}

func TestWebhookFirstFailureAtRendersAsNull(t *testing.T) {
	wh := Webhook{ID: "whk_1", OrgID: "org_1", URL: "https://e.com/h", Active: true}
	out, err := json.Marshal(wh)
	require.NoError(t, err)
	require.Contains(t, string(out), `"firstFailureAt":null`)
	// secret keeps omitempty — must NOT appear when empty.
	require.NotContains(t, string(out), `"secret"`)
}

func TestEventAndAgentNullableFieldsRenderAsNull(t *testing.T) {
	ev := Event{ID: "evt_1", CalendarID: "cal_1", Title: "Standup"}
	out, err := json.Marshal(ev)
	require.NoError(t, err)
	s := string(out)
	for _, key := range []string{`"description":null`, `"holdExpiresAt":null`, `"holdPriority":null`} {
		require.Contains(t, s, key, "nullable event field must serialize as null")
	}

	ag := Agent{ID: "agt_1", OrgID: "org_1", Name: "Bot", Type: "ai", Status: "active"}
	aout, err := json.Marshal(ag)
	require.NoError(t, err)
	require.Contains(t, string(aout), `"description":null`)
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
