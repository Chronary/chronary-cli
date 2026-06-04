package cmd

import "testing"

// validWebhookEvents must mirror the server's WEBHOOK_EVENT_TYPES (18 events).
// webhook.deactivated was previously missing — guard against that drift.
func TestValidWebhookEventsParity(t *testing.T) {
	want := []string{
		"agent.created", "agent.updated",
		"event.created", "event.updated", "event.deleted", "event.started", "event.ended", "event.reminder",
		"event.hold_created", "event.hold_expired", "event.hold_released", "event.hold_confirmed",
		"proposal.created", "proposal.responded", "proposal.confirmed", "proposal.expired", "proposal.cancelled",
		"webhook.deactivated",
	}

	if len(validWebhookEvents) != len(want) {
		t.Fatalf("expected %d valid webhook events, got %d", len(want), len(validWebhookEvents))
	}

	have := make(map[string]bool, len(validWebhookEvents))
	for _, e := range validWebhookEvents {
		have[e] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("validWebhookEvents is missing %q", w)
		}
	}
	if !have["webhook.deactivated"] {
		t.Error("validWebhookEvents must include webhook.deactivated")
	}
}
