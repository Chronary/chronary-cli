package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #2519: a caller-supplied ID stays inside one escaped path segment, and "",
// "." and ".." are refused before any request is sent.

func TestPathSegmentEscapes(t *testing.T) {
	cases := map[string]string{
		"evt_abc-123_XYZ":                      "evt_abc-123_XYZ",
		"11111111-2222-3333-4444-555555555555": "11111111-2222-3333-4444-555555555555",
		"../x":                                 "..%2Fx",
		"%2e%2e":                               "%252e%252e",
		"a#b":                                  "a%23b",
		"a?b":                                  "a%3Fb",
		"a/b":                                  "a%2Fb",
		"...":                                  "...",
	}
	for in, want := range cases {
		assert.Equal(t, want, PathSegment(in), in)
	}
}

func TestPathSegmentRefusedValuesNeverReachTheServer(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(204)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "chr_sk_x", false)

	for _, bad := range []string{"", ".", ".."} {
		_, status, err := c.Get("/v1/events/" + PathSegment(bad))
		require.Error(t, err, bad)
		assert.Equal(t, 0, status)
		assert.Contains(t, err.Error(), "No request was sent")
		_, _, err = c.Delete("/v1/agents/" + PathSegment(bad))
		require.Error(t, err, bad)
		_, _, err = c.Post("/v1/webhooks/"+PathSegment(bad)+"/test", map[string]string{})
		require.Error(t, err, bad)
	}
	assert.Equal(t, 0, requests)
}

func TestPathSegmentKeepsTheIDInItsResource(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		w.WriteHeader(204)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "chr_sk_x", false)

	_, _, err := c.Delete("/v1/events/" + PathSegment("../agents/agt_1"))
	require.NoError(t, err)
	_, _, err = c.Get("/v1/events/" + PathSegment("evt_1?expand=all"))
	require.NoError(t, err)
	assert.Equal(t, []string{
		"DELETE /v1/events/..%2Fagents%2Fagt_1?",
		"GET /v1/events/evt_1%3Fexpand=all?",
	}, got)
	for _, g := range got {
		assert.False(t, strings.Contains(g, "/v1/agents/"), g)
	}
}
