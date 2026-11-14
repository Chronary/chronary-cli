package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Chronary/chronary-cli/pkg/client"
)

// Page sizes for `--all` (fetchAllPages). Each is the server's maximum
// `limit` for that endpoint, mirrored from the list query schema in
// packages/shared/src/schemas. A page size above the cap gets a 400 from the
// API (#1992); TestFetchAllPagesPageSizes pins every value.
const (
	pageSizeAgents            = 200 // ListAgentsQuerySchema (agents.ts)
	pageSizeCalendars         = 200 // ListCalendarsQuerySchema (calendars.ts)
	pageSizeEvents            = 200 // ListEventsQuerySchema (events.ts)
	pageSizeOrgEvents         = 100 // ListOrgEventsQuerySchema (events.ts)
	pageSizeBookingPages      = 200 // ListBookingPagesQuerySchema (booking-pages.ts)
	pageSizeICalSubscriptions = 200 // ListICalSubscriptionsQuerySchema (ical-subscriptions.ts)
	pageSizeProposals         = 200 // ListProposalsQuerySchema (scheduling.ts)
	pageSizeWebhooks          = 100 // ListWebhooksQuerySchema (webhooks.ts)
)

// paginatedResponse is the generic shape of a list API response for pagination.
type paginatedResponse struct {
	Data   []json.RawMessage `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

// fetchAllPages fetches all pages from a paginated endpoint and returns the
// combined raw JSON items. The basePath should not include limit/offset params.
func fetchAllPages(c *client.Client, basePath string, limit int) ([]json.RawMessage, int, error) {
	// Non-nil, so an empty result marshals as [] rather than null (#2095).
	allData := []json.RawMessage{}
	offset := 0
	total := 0

	for {
		params := url.Values{}
		params.Set("limit", strconv.Itoa(limit))
		params.Set("offset", strconv.Itoa(offset))
		path := appendQueryParams(basePath, params)

		body, _, err := c.Get(path)
		if err != nil {
			return nil, 0, err
		}

		var resp paginatedResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, 0, fmt.Errorf("parsing response: %w", err)
		}

		total = resp.Total
		allData = append(allData, resp.Data...)

		if offset+len(resp.Data) >= total {
			break
		}
		offset += len(resp.Data)
	}

	return allData, total, nil
}

// rewrapList reconstructs a ListResponse JSON from collected raw items: one
// page holding everything, so `limit` is the item count. An empty result
// keeps the list contract every other response follows (#2095): `data` is
// [] (never null) and `limit` is the page size that was asked for (the
// endpoint's cap, always >= 1), not 0.
func rewrapList(items []json.RawMessage, total, pageSize int) ([]byte, error) {
	if items == nil {
		items = []json.RawMessage{}
	}
	limit := len(items)
	if limit == 0 {
		limit = pageSize
	}
	wrapped := map[string]any{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": 0,
	}
	return json.Marshal(wrapped)
}

func appendQueryParams(path string, params url.Values) string {
	if len(params) == 0 {
		return path
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + params.Encode()
}
