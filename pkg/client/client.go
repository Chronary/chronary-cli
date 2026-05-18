package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Version is the CLI version reported in the User-Agent header so the API
// can attribute traffic to the CLI surface (vs. SDKs and raw REST). Set at
// build time via -ldflags "-X github.com/Chronary/chronary-cli/pkg/client.Version=$VERSION".
// Defaults to "dev" for unbuilt and local-test runs.
var Version = "dev"

// Client is the Chronary API HTTP client.
type Client struct {
	BaseURL    string
	APIKey     string
	Debug      bool
	HTTPClient *http.Client
}

var (
	chronaryKeyPattern = regexp.MustCompile(`chr_(?:sk|ak)_(?:live_)?[A-Za-z0-9_-]+`)
	jsonSecretPattern  = regexp.MustCompile(`"((?:api_)?key|secret|revocation_token|ical_token|ical_feed_url)"\s*:\s*"[^"]*"`)
	icalURLPattern     = regexp.MustCompile(`https://[^"\s]+/ical/[^"\s]+\.ics`)
	retrySleep         = time.Sleep
)

const maxGetRetries = 2

func redactDebugBody(body []byte) string {
	redacted := chronaryKeyPattern.ReplaceAllString(string(body), "[redacted_key]")
	redacted = icalURLPattern.ReplaceAllString(redacted, "[redacted_ical_url]")
	return jsonSecretPattern.ReplaceAllString(redacted, `"$1":"[redacted]"`)
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

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, 0, fmt.Errorf("reading request body: %w", err)
		}
	}

	attempts := 1
	if method == "GET" {
		attempts += maxGetRetries
	}

	var lastStatus int
	for attempt := 0; attempt < attempts; attempt++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		respBody, status, retryAfter, err := c.doOnce(method, url, bodyReader)
		lastStatus = status
		if err != nil {
			if method == "GET" && attempt < attempts-1 {
				retrySleep(retryDelay(attempt+1, ""))
				continue
			}
			return nil, status, err
		}

		if isRetryableStatus(status) && method == "GET" && attempt < attempts-1 {
			retrySleep(retryDelay(attempt+1, retryAfter))
			continue
		}
		if status >= 200 && status < 300 {
			return respBody, status, nil
		}

		return c.responseError(status, respBody)
	}

	return nil, lastStatus, fmt.Errorf("request failed after retries")
}

func (c *Client) doOnce(method, url string, body io.Reader) ([]byte, int, string, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, "", fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	req.Header.Set("User-Agent", "chronary-cli/"+Version)

	if c.Debug {
		fmt.Printf(">>> %s %s\n", method, url)
		if c.APIKey != "" {
			masked := c.APIKey[:min(16, len(c.APIKey))] + "..."
			fmt.Printf(">>> Authorization: Bearer %s\n", masked)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("network error: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, resp.StatusCode, resp.Header.Get("Retry-After"), fmt.Errorf("reading response: %w", err)
	}

	if c.Debug {
		fmt.Printf("<<< %d %s\n", resp.StatusCode, resp.Status)
		if len(respBody) > 0 {
			fmt.Printf("<<< %s\n", redactDebugBody(respBody))
		}
	}

	return respBody, resp.StatusCode, resp.Header.Get("Retry-After"), nil
}

func (c *Client) responseError(status int, respBody []byte) ([]byte, int, error) {
	statusText := http.StatusText(status)
	statusLine := fmt.Sprintf("%d %s", status, statusText)
	if statusText == "" {
		statusLine = fmt.Sprintf("HTTP %d", status)
	}
	resp := &http.Response{StatusCode: status, Status: statusLine}
	if apiErr := parseAPIError(resp, respBody); apiErr != nil {
		return nil, status, apiErr
	}
	return nil, status, &HTTPError{
		StatusCode: status,
		Status:     resp.Status,
		Body:       string(respBody),
	}
}

func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

func retryDelay(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return minDuration(time.Duration(seconds)*time.Second, 5*time.Second)
		}
		if retryAt, err := http.ParseTime(retryAfter); err == nil {
			delay := time.Until(retryAt)
			if delay < 0 {
				return 0
			}
			return minDuration(delay, 5*time.Second)
		}
	}
	delay := time.Duration(100*(1<<(attempt-1))) * time.Millisecond
	return minDuration(delay, 2*time.Second)
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
