package cmd

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendQueryParamsEncodesSpecialCharacters(t *testing.T) {
	params := url.Values{}
	params.Set("start", "2026-04-20T10:00:00+05:30")
	params.Set("title", "team sync")
	params.Set("cursor", "evt_%ready")

	path := appendQueryParams("/v1/events", params)
	query := path[len("/v1/events?"):]

	assert.Contains(t, query, "start=2026-04-20T10%3A00%3A00%2B05%3A30")
	assert.Contains(t, query, "title=team+sync")
	assert.Contains(t, query, "cursor=evt_%25ready")
}

func TestAppendQueryParamsPreservesExistingQuery(t *testing.T) {
	params := url.Values{}
	params.Set("limit", "50")

	assert.Equal(t, "/v1/events?status=hold&limit=50", appendQueryParams("/v1/events?status=hold", params))
}
