package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func upgradeRequestFixture(status string) map[string]any {
	return map[string]any{
		"id": "upr_1", "status": status, "plan": "pro", "interval": "month",
		"reason": nil, "feature": nil, "metric": nil, "requested_via": "api",
		"requested_by_key_prefix": "chr_sk_abcdefghijkl",
		"approval_url":            "https://console.test/settings/billing/requests/upr_1",
		"created_at":              "2026-10-19T00:00:00Z", "expires_at": "2026-10-26T00:00:00Z",
		"completed_at": nil, "declined_at": nil, "canceled_at": nil,
	}
}

func TestBillingStatusCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/billing", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode(map[string]any{
			"plan": "free-agent", "plan_source": nil, "plan_expires_at": nil, "status": nil,
			"interval": nil, "current_period_end": nil, "cancel_at": nil, "cancel_at_period_end": false,
			"renews_at": nil, "price": nil, "has_billing_account": false, "self_serve_upgrade": true,
			"upgrade_options": []map[string]any{
				{"plan": "pro", "interval": "month", "amount": 2900, "currency": "usd", "savings_vs_monthly": 0},
				{"plan": "pro", "interval": "year", "amount": 29000, "currency": "usd", "savings_vs_monthly": 5800},
			},
			"upgrade_url": "https://console.test/settings/billing", "manage_url": "https://console.test/settings/billing",
			"contact_url": nil, "pending_upgrade_request": nil, "offer": nil,
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "free-agent")
	assert.Contains(t, out, "29.00 USD")
	assert.Contains(t, out, "saves 58.00 USD vs monthly")
	assert.Contains(t, out, "Upgrade: https://console.test/settings/billing")
	assert.Contains(t, out, "chronary billing request-upgrade")
}

func TestBillingStatusJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"plan": "pro", "status": "active", "interval": "month", "cancel_at_period_end": false,
			"has_billing_account": true, "self_serve_upgrade": false, "upgrade_options": []any{},
			"manage_url": "https://console.test/settings/billing", "pending_upgrade_request": upgradeRequestFixture("pending"),
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
		require.NoError(t, rootCmd.Execute())
	})
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "pro", got["plan"])
	assert.NotNil(t, got["pending_upgrade_request"])
}

func TestBillingRequestUpgrade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/billing/upgrade-requests", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "year", body["interval"])
		assert.Equal(t, "need more events", body["reason"])
		w.WriteHeader(http.StatusCreated)
		resp := upgradeRequestFixture("pending")
		resp["reused"] = true
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "request-upgrade", "--interval", "year", "--reason", "need more events",
			"--api-key", "chr_sk_xxx", "--base-url", srv.URL})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "upr_1")
	assert.Contains(t, out, "Approval URL:    https://console.test/settings/billing/requests/upr_1")
	assert.Contains(t, out, "already pending")
}

func TestBillingRequestUpgradeRejectsBadInterval(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"billing", "request-upgrade", "--interval", "week", "--api-key", "chr_sk_xxx", "--base-url", "http://127.0.0.1:1"})
	rootCmd.SilenceErrors = true
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--interval must be month or year")
}

func TestBillingRequestUpgradeWaitPollsUntilTerminal(t *testing.T) {
	origInterval, origSleep := upgradeRequestPollInterval, upgradeRequestSleep
	slept := 0
	upgradeRequestPollInterval = time.Minute
	upgradeRequestSleep = func(d time.Duration) {
		assert.Equal(t, time.Minute, d)
		slept++
	}
	defer func() { upgradeRequestPollInterval, upgradeRequestSleep = origInterval, origSleep }()

	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(upgradeRequestFixture("pending"))
			return
		}
		assert.Equal(t, "/v1/billing/upgrade-requests/upr_1", r.URL.Path)
		polls++
		status := "pending"
		if polls >= 3 {
			status = "completed"
		}
		json.NewEncoder(w).Encode(upgradeRequestFixture(status))
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "request-upgrade", "--wait", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Equal(t, 3, polls)
	assert.Equal(t, 3, slept)
	assert.Contains(t, out, "is completed")
	assert.Contains(t, out, "Retry the blocked call in about 15 seconds")
}

func TestUpgradeRequestGetAndCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/billing/upgrade-requests/upr_1":
			json.NewEncoder(w).Encode(upgradeRequestFixture("pending"))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/billing/upgrade-requests/upr_1/cancel":
			json.NewEncoder(w).Encode(upgradeRequestFixture("canceled"))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "upgrade-request", "get", "upr_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "upr_1 (pending")

	out = captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "upgrade-request", "cancel", "upr_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "-o", "json"})
		require.NoError(t, rootCmd.Execute())
	})
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "canceled", got["status"])
}

func TestPlanRequiredErrorPrintsUpgradeLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
			"type": "plan_required", "message": "Pro required", "request_id": "req_1",
			"required_plan": "pro", "feature": "scoped_keys",
			"upgrade_url": "https://console.test/settings/billing", "docs_url": "https://docs.test/plans",
		}})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SilenceErrors = true
	rootCmd.SetArgs([]string{"billing", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires the pro plan (scoped_keys)")
	assert.Contains(t, err.Error(), "Upgrade: https://console.test/settings/billing")
	assert.Contains(t, err.Error(), "Docs: https://docs.test/plans")
	assert.Contains(t, err.Error(), "chronary billing request-upgrade")
}

func TestQuotaExceededErrorPrintsUpgradeLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
			"type": "quota_exceeded", "message": "Quota hit", "request_id": "req_2",
			"metric": "api_calls", "limit": 50000, "upgrade_url": "https://console.test/settings/billing",
		}})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SilenceErrors = true
	rootCmd.SetArgs([]string{"usage", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Quota exceeded for api_calls (limit 50000)")
	assert.Contains(t, err.Error(), "UTC calendar month")
	assert.NotContains(t, err.Error(), "billing cycle")
	assert.Contains(t, err.Error(), "Upgrade: https://console.test/settings/billing")
}

func TestBillingStatusShowsPendingPlanAndDeveloperOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"plan": "pro", "status": "active", "interval": "month", "cancel_at_period_end": false,
			"has_billing_account": true, "self_serve_upgrade": false,
			"pending_plan": "developer", "pending_plan_interval": "month", "pending_plan_at": "2026-12-01T00:00:00Z",
			"upgrade_options": []map[string]any{
				{"plan": "developer", "interval": "month", "amount": 900, "currency": "usd", "savings_vs_monthly": 0, "via": "switch_plan"},
			},
			"manage_url": "https://console.test/settings/billing", "pending_upgrade_request": nil,
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"billing", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "Switching to")
	assert.Contains(t, out, "developer (monthly) from 2026-12-01T00:00:00Z")
	assert.Contains(t, out, "developer (per month): 9.00 USD")
}

func TestQuotaExceededErrorDecodesUpgradePlan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
			"type": "quota_exceeded", "message": "Quota hit", "metric": "api_calls", "limit": 50000,
			"upgrade_url": "https://console.test/settings/billing", "upgrade_plan": "developer",
		}})
	}))
	defer srv.Close()
	rootCmd := NewRootCmd("test")
	rootCmd.SilenceErrors = true
	rootCmd.SetArgs([]string{"billing", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Upgrade to developer: https://console.test/settings/billing")
}
