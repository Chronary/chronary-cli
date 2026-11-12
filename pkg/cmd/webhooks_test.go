package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWebhook(id, url string, events []string) client.Webhook {
	return client.Webhook{
		ID:        id,
		URL:       url,
		Events:    events,
		Active:    true,
		CreatedAt: time.Now(),
	}
}

func TestWebhooksListCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/webhooks", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"data":  []client.Webhook{newTestWebhook("whk_1", "https://example.com/hook", []string{"agent.created"})},
			"total": 1, "limit": 20, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "list", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "https://example.com/hook", body["url"])
		events := body["events"].([]any)
		assert.Len(t, events, 2)
		w.WriteHeader(201)
		wh := newTestWebhook("whk_new", "https://example.com/hook", []string{"agent.created", "event.created"})
		wh.Secret = "whsec_abc123"
		json.NewEncoder(w).Encode(wh)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"webhooks", "create",
		"--url", "https://example.com/hook",
		"--events", "agent.created,event.created",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/webhooks/whk_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestWebhook("whk_1", "https://example.com/hook", []string{"agent.created"}))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "get", "whk_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksUpdateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, false, body["active"])
		json.NewEncoder(w).Encode(newTestWebhook("whk_1", "https://example.com/hook", []string{"agent.created"}))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "update", "whk_1", "--active=false", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksDeliveryCommand(t *testing.T) {
	status := 503
	dur := 120
	code := "non_2xx"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/webhooks/whk_1/deliveries/whd_1", r.URL.Path)
		json.NewEncoder(w).Encode(client.WebhookDeliveryDetail{
			ID: "whd_1", SubscriptionID: "whk_1", EventType: "event.created",
			Status: "failed", Attempts: 2, MaxAttempts: 4, CreatedAt: time.Now(),
			AttemptLog: []client.WebhookDeliveryAttempt{
				{ID: "wha_1", AttemptNumber: 1, StartedAt: time.Now()},
				{ID: "wha_2", AttemptNumber: 2, StartedAt: time.Now(), ResponseStatus: &status, DurationMs: &dur, ErrorCode: &code},
			},
		})
	}))
	defer srv.Close()

	for _, format := range []string{"json", "table"} {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"webhooks", "delivery", "whk_1", "whd_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", format})
		require.NoError(t, rootCmd.Execute())
	}
}

func TestWebhooksDeliveriesCommandFilters(t *testing.T) {
	next := "WyJ4IiwieSJd"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/webhooks/whk_1/deliveries", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "event.", q.Get("event_type"))
		assert.Equal(t, "2026-04-18T00:00:00Z", q.Get("created_after"))
		assert.Equal(t, "2026-04-19T00:00:00Z", q.Get("created_before"))
		assert.Equal(t, "whd_1", q.Get("delivery_id"))
		assert.Equal(t, "false", q.Get("is_test"))
		assert.Equal(t, "abc", q.Get("cursor"))
		assert.Equal(t, "dropped", q.Get("status"))
		json.NewEncoder(w).Encode(client.WebhookDeliveryListResponse{
			Data:       []client.WebhookDelivery{{ID: "whd_1", EventType: "event.created", Status: "dropped", CreatedAt: time.Now()}},
			Total:      3,
			Limit:      1,
			NextCursor: &next,
			HasMore:    true,
			Stats:      client.WebhookDeliveryStats{Delivered: 1, Dropped: 2},
		})
	}))
	defer srv.Close()

	for _, format := range []string{"json", "table"} {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{
			"webhooks", "deliveries", "whk_1",
			"--status", "dropped", "--event-type", "event.",
			"--created-after", "2026-04-18T00:00:00Z", "--created-before", "2026-04-19T00:00:00Z",
			"--delivery-id", "whd_1", "--test=false", "--cursor", "abc",
			"--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", format,
		})
		require.NoError(t, rootCmd.Execute())
	}
}

func TestWebhooksDeliveriesCommandOmitsUnsetTestFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, r.URL.Query().Has("is_test"))
		json.NewEncoder(w).Encode(client.WebhookDeliveryListResponse{})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "deliveries", "whk_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksDeliveryCommandRequiresTwoArgs(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "delivery", "whk_1", "--api-key", "chr_sk_xxx"})
	require.Error(t, rootCmd.Execute())
}

func TestWebhooksTestCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/webhooks/whk_1/test", r.URL.Path)
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "event.created", body["event_type"])
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(client.WebhookTestSent{
			DeliveryID: "whd_t", WebhookID: "whk_1", EventType: "event.created", Status: "pending", IsTest: true,
		})
	}))
	defer srv.Close()

	for _, format := range []string{"json", "table"} {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"webhooks", "test", "whk_1", "--event", "event.created", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", format})
		require.NoError(t, rootCmd.Execute())
	}
}

func TestWebhooksTestCommandRequiresEvent(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "test", "whk_1", "--api-key", "chr_sk_xxx"})
	require.Error(t, rootCmd.Execute())
}

func TestWebhooksRotateSecretCommand(t *testing.T) {
	expiry := "2026-11-05T10:00:00.000Z"
	var bodies []map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/webhooks/whk_1/secret/rotate", r.URL.Path)
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		resp := client.RotatedWebhookSecret{WebhookID: "whk_1", Secret: "whsec_new"}
		if body["expire_old_in"] != "now" {
			resp.PreviousSecretExpiresAt = &expiry
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cases := [][]string{
		{"--output", "json"},
		{"--output", "table"},
		{"--expire-old-in", "now", "--output", "table"},
	}
	for _, extra := range cases {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs(append([]string{"webhooks", "rotate-secret", "whk_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL}, extra...))
		require.NoError(t, rootCmd.Execute())
	}
	require.Len(t, bodies, 3)
	assert.Equal(t, "24h", bodies[0]["expire_old_in"])
	assert.Equal(t, "now", bodies[2]["expire_old_in"])
}

func TestWebhooksRotateSecretCommandRejectsUnknownExpiry(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "rotate-secret", "whk_1", "--expire-old-in", "30d", "--api-key", "chr_sk_xxx"})
	require.Error(t, rootCmd.Execute())
}

func TestWebhooksDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "delete", "whk_1", "--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksReplayCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/webhooks/whk_1/deliveries/whd_orig/replay", r.URL.Path)
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(client.WebhookReplaySent{
			DeliveryID: "whd_new", WebhookID: "whk_1", EventType: "event.created", Status: "pending",
			ReplayOfDeliveryID: "whd_orig",
		})
	}))
	defer srv.Close()

	for _, format := range []string{"json", "table"} {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"webhooks", "replay", "whk_1", "whd_orig", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", format})
		require.NoError(t, rootCmd.Execute())
	}
}

func TestWebhooksReplayCommandPlanRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"type":"plan_required","message":"Replaying webhook deliveries requires a Pro plan","required_plan":"pro","feature":"webhook_replay"}}`))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "replay", "whk_1", "whd_orig", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook_replay")
}

func TestWebhooksDeliveryShowsReplayOf(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orig := "whd_orig"
		json.NewEncoder(w).Encode(client.WebhookDeliveryDetail{
			ID: "whd_new", SubscriptionID: "whk_1", EventType: "event.created", Status: "pending",
			MaxAttempts: 8, ReplayOfDeliveryID: &orig,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "delivery", "whk_1", "whd_new", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "table"})
	require.NoError(t, rootCmd.Execute())
}
