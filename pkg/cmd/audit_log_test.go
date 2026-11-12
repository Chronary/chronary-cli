package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func auditLogServer(t *testing.T, gotQuery *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/audit-log", r.URL.Path)
		q := map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		*gotQuery = q
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"id": "aud_1", "action": "agent_key.create", "actor_key_prefix": nil,
					"actor_type": "session", "actor_id": "org_1", "actor_label": "owner@example.com",
					"agent_id": nil, "resource": "", "ip": nil, "status": 201, "method": "POST",
					"path": "/v1/auth/keys/scoped", "duration_ms": 12, "request_id": "req_1",
					"created_at": "2026-10-30T00:00:00.000Z",
				},
			},
			"pagination":     map[string]any{"next_cursor": nil},
			"retention_days": 90,
			"range_clamped":  false,
		})
	}))
}

func TestAuditLogListActorFlags(t *testing.T) {
	var got map[string]string
	srv := auditLogServer(t, &got)
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{
			"audit-log", "list", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json",
			"--from", "2026-10-01T00:00:00+02:00",
			"--actor-type", "session", "--actor-id", "org_1", "--request-id", "req_1",
			"--resource", "cal_", "--status-class", "2xx", "--category", "keys", "--limit", "10",
		})
		require.NoError(t, rootCmd.Execute())
	})

	assert.Equal(t, map[string]string{
		// URL-encoded on the wire, so the "+" offset survives.
		"from":         "2026-10-01T00:00:00+02:00",
		"actor_type":   "session",
		"actor_id":     "org_1",
		"request_id":   "req_1",
		"resource":     "cal_",
		"status_class": "2xx",
		"category":     "keys",
		"limit":        "10",
	}, got)

	var resp client.AuditLogResponse
	require.NoError(t, json.Unmarshal([]byte(out), &resp))
	require.Len(t, resp.Data, 1)
	require.NotNil(t, resp.Data[0].ActorType)
	assert.Equal(t, "session", *resp.Data[0].ActorType)
	require.NotNil(t, resp.Data[0].ActorLabel)
	assert.Equal(t, "owner@example.com", *resp.Data[0].ActorLabel)
}

func TestAuditLogListTableShowsActorLabel(t *testing.T) {
	var got map[string]string
	srv := auditLogServer(t, &got)
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"audit-log", "list", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})

	assert.Equal(t, map[string]string{"limit": "50"}, got)
	assert.Contains(t, out, "owner@example.com [session]")
}
