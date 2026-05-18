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

func newTestProposal(id, status string) client.Proposal {
	return client.Proposal{
		ID:                  id,
		Title:               "Project sync",
		OrganizerAgentID:    "agt_org",
		ParticipantAgentIDs: []string{"agt_a", "agt_b"},
		CalendarID:          "cal_team",
		Status:              status,
		Metadata:            json.RawMessage(`{}`),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

func TestSchedulingCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/scheduling/proposals", r.URL.Path)
		w.WriteHeader(201)
		// Emit raw snake_case JSON that matches the real API shape, so we catch
		// regressions in struct tag casing.
		_, _ = w.Write([]byte(`{
			"id": "spr_new",
			"title": "Project sync",
			"description": null,
			"organizer_agent_id": "agt_org",
			"participant_agent_ids": ["agt_a", "agt_b"],
			"calendar_id": "cal_team",
			"status": "pending",
			"expires_at": null,
			"resolved_slot": null,
			"created_event_id": null,
			"metadata": {},
			"created_at": "2026-04-16T12:00:00Z",
			"updated_at": "2026-04-16T12:00:00Z"
		}`))
	}))
	defer srv.Close()

	fileArg := writeTempJSON(t, "proposal.json", map[string]any{
		"title":                  "Project sync",
		"organizer_agent_id":     "agt_org",
		"participant_agent_ids":  []string{"agt_a", "agt_b"},
		"calendar_id":            "cal_team",
		"slots":                  []map[string]any{{"start_time": "2026-04-20T14:00:00Z", "end_time": "2026-04-20T15:00:00Z"}},
	})

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"scheduling", "create", fileArg,
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestSchedulingListCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/scheduling/proposals", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":   []client.Proposal{newTestProposal("spr_1", "pending")},
			"total":  1,
			"limit":  50,
			"offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"scheduling", "list", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestSchedulingGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/scheduling/proposals/spr_1", r.URL.Path)
		_ = json.NewEncoder(w).Encode(newTestProposal("spr_1", "pending"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"scheduling", "get", "spr_1", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestSchedulingRespondCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/scheduling/proposals/spr_1/respond", r.URL.Path)
		_ = json.NewEncoder(w).Encode(client.ProposalResponse{
			ID: "rsp_1", AgentID: "agt_a", Response: "accept",
			CreatedAt: time.Now(),
		})
	}))
	defer srv.Close()

	fileArg := writeTempJSON(t, "respond.json", map[string]any{
		"agent_id":         "agt_a",
		"response":         "accept",
		"selected_slot_id": "slt_1",
	})

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"scheduling", "respond", "spr_1", fileArg,
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestSchedulingResolveCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/scheduling/proposals/spr_1/resolve", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "confirmed",
			"resolved_slot": map[string]any{
				"id":          "slt_1",
				"start_time":  "2026-04-20T14:00:00Z",
				"end_time":    "2026-04-20T15:00:00Z",
				"weight":      1.0,
				"calendar_id": nil,
			},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"scheduling", "resolve", "spr_1", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

// TestProposalSnakeCaseUnmarshal guards against regressions where Go struct
// tags drift from the scheduling API's snake_case response shape. See
// apps/api/src/services/scheduling.ts formatProposal / formatSlot / formatResponse.
func TestProposalSnakeCaseUnmarshal(t *testing.T) {
	raw := []byte(`{
		"id": "spr_1",
		"title": "Sync",
		"organizer_agent_id": "agt_org",
		"participant_agent_ids": ["agt_a"],
		"calendar_id": "cal_team",
		"status": "confirmed",
		"created_event_id": "evt_x",
		"created_at": "2026-04-16T12:00:00Z",
		"updated_at": "2026-04-16T12:00:00Z",
		"metadata": {},
		"slots": [
			{"id": "slt_1", "start_time": "2026-04-20T14:00:00Z", "end_time": "2026-04-20T15:00:00Z", "weight": 2.0, "calendar_id": null}
		],
		"responses": [
			{"id": "rsp_1", "agent_id": "agt_a", "response": "accept", "selected_slot_id": "slt_1", "counter_slots": null, "message": null, "created_at": "2026-04-16T13:00:00Z"}
		]
	}`)

	var p client.Proposal
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, "agt_org", p.OrganizerAgentID)
	assert.Equal(t, []string{"agt_a"}, p.ParticipantAgentIDs)
	assert.Equal(t, "cal_team", p.CalendarID)
	require.NotNil(t, p.CreatedEventID)
	assert.Equal(t, "evt_x", *p.CreatedEventID)
	require.Len(t, p.Slots, 1)
	assert.Equal(t, "slt_1", *p.Slots[0].ID)
	assert.Equal(t, 2.0, p.Slots[0].Weight)
	require.Len(t, p.Responses, 1)
	assert.Equal(t, "agt_a", p.Responses[0].AgentID)
	require.NotNil(t, p.Responses[0].SelectedSlotID)
	assert.Equal(t, "slt_1", *p.Responses[0].SelectedSlotID)
}

func TestSchedulingCancelCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/scheduling/proposals/spr_1/cancel", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "cancelled"})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"scheduling", "cancel", "spr_1", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}
