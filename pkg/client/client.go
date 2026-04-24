package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the Chronary API HTTP client.
type Client struct {
	BaseURL    string
	APIKey     string
	Debug      bool
	HTTPClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL, apiKey string, debug bool) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Debug:   debug,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request to the given path.
func (c *Client) Get(path string) ([]byte, int, error) {
	return c.do("GET", path, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body any) ([]byte, int, error) {
	return c.doJSON("POST", path, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(path string, body any) ([]byte, int, error) {
	return c.doJSON("PATCH", path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body any) ([]byte, int, error) {
	return c.doJSON("PUT", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) ([]byte, int, error) {
	return c.do("DELETE", path, nil)
}

func (c *Client) doJSON(method, path string, body any) ([]byte, int, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, 0, fmt.Errorf("encoding request body: %w", err)
		}
	}
	return c.do(method, path, &buf)
}

func (c *Client) do(method, path string, body io.Reader) ([]byte, int, error) {
	url := c.BaseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	req.Header.Set("User-Agent", "chronary-cli")

	if c.Debug {
		fmt.Printf(">>> %s %s\n", method, url)
		if c.APIKey != "" {
			masked := c.APIKey[:min(16, len(c.APIKey))] + "..."
			fmt.Printf(">>> Authorization: Bearer %s\n", masked)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if c.Debug {
		fmt.Printf("<<< %d %s\n", resp.StatusCode, resp.Status)
		if len(respBody) > 0 {
			fmt.Printf("<<< %s\n", string(respBody))
		}
	}

	// Success responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, resp.StatusCode, nil
	}

	// Try to parse structured API error
	if apiErr := parseAPIError(resp, respBody); apiErr != nil {
		return nil, resp.StatusCode, apiErr
	}

	// Fallback to generic HTTP error
	return nil, resp.StatusCode, &HTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(respBody),
	}
}
