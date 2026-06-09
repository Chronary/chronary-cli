package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/stretchr/testify/require"
)

const plansCatalogFixture = `{
  "plans": [
    {
      "id": "free",
      "name": "Free",
      "tagline": "For prototyping and small agents",
      "price": 0,
      "currency": "usd",
      "limits": {
        "agents": 5, "calendars": 10, "events": 5000,
        "api_calls": 50000, "webhook_deliveries": 10000,
        "availability_queries": 10000, "ical_subscriptions": 3, "proposals": 500,
        "webhook_endpoints": 3, "scoped_keys": 0
      },
      "display_features": ["5 agents"],
      "recommended": false
    },
    {
      "id": "pro",
      "name": "Pro",
      "tagline": "For production agent workflows",
      "price": 2900,
      "currency": "usd",
      "limits": {
        "agents": 500, "calendars": 1000, "events": 500000,
        "api_calls": 1000000, "webhook_deliveries": 1000000,
        "availability_queries": 1000000, "ical_subscriptions": 100, "proposals": null
      },
      "display_features": ["500 agents"],
      "recommended": true
    },
    {
      "id": "custom",
      "name": "Custom",
      "tagline": "For multi-agent platforms and enterprise",
      "price": null,
      "currency": null,
      "limits": null,
      "display_features": ["Custom volume limits"],
      "recommended": false,
      "custom_pricing": true,
      "contact_url": "https://chronary.ai/contact"
    }
  ]
}`

func TestPlansListCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/plans", r.URL.Path)
		// Public endpoint — should work without an Authorization header.
		require.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(plansCatalogFixture))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"plans", "list", "--base-url", srv.URL, "--output", "json"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

// Guards against the CLI plan model dropping live limit fields from `-o json`
// output. webhook_endpoints and scoped_keys were previously absent from
// PlanLimits, so they silently vanished from the machine-readable output.
func TestPlanLimitsPreservesEndpointAndKeyCaps(t *testing.T) {
	var resp client.PlansListResponse
	require.NoError(t, json.Unmarshal([]byte(plansCatalogFixture), &resp))
	require.NotEmpty(t, resp.Plans)

	free := resp.Plans[0]
	require.NotNil(t, free.Limits)
	require.NotNil(t, free.Limits.WebhookEndpoints, "webhook_endpoints must round-trip")
	require.Equal(t, 3, *free.Limits.WebhookEndpoints)
	require.NotNil(t, free.Limits.ScopedKeys, "scoped_keys must round-trip")
	require.Equal(t, 0, *free.Limits.ScopedKeys)
}

func TestFormatPlanPrice(t *testing.T) {
	zero := 0
	pro := 2900
	usd := "usd"

	cases := []struct {
		name string
		plan client.Plan
		want string
	}{
		{"free tier renders 'Free'", client.Plan{Price: &zero, Currency: &usd}, "Free"},
		{"pro tier renders dollars.cents", client.Plan{Price: &pro, Currency: &usd}, "$29.00/mo usd"},
		{"custom pricing renders 'Contact sales'", client.Plan{CustomPricing: true}, "Contact sales"},
		{"missing price renders 'Contact sales'", client.Plan{}, "Contact sales"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatPlanPrice(tc.plan))
		})
	}
}
