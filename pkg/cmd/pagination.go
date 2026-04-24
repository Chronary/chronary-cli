package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Chronary/chronary-cli/pkg/client"
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
	var allData []json.RawMessage
	offset := 0
	total := 0

	for {
		sep := "?"
		if strings.Contains(basePath, "?") {
			sep = "&"
		}
		path := fmt.Sprintf("%s%slimit=%d&offset=%d", basePath, sep, limit, offset)

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

// rewrapList reconstructs a ListResponse JSON from collected raw items.
func rewrapList(items []json.RawMessage, total int) ([]byte, error) {
	wrapped := map[string]any{
		"data":   items,
		"total":  total,
		"limit":  total,
		"offset": 0,
	}
	return json.Marshal(wrapped)
}
