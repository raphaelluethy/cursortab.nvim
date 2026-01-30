package sweep

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andybalholm/brotli"

	"cursortab/logger"
)

const (
	DefaultAutocompletePath = "/backend/next_edit_autocomplete"
	DefaultMetricsPath      = "/backend/track_autocomplete_metrics"
	DefaultAPIKeyEnv        = "SWEEP_AI_TOKEN"
)

// Client is a reusable Sweep API client for hosted Sweep
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	APIKey     string
}

// NewClient creates a new Sweep client with the given base URL and API key configuration
// If apiKey is empty, it will attempt to read from the environment variable specified by apiKeyEnv
// (or SWEEP_AI_TOKEN if apiKeyEnv is empty)
func NewClient(baseURL, apiKey, apiKeyEnv string) (*Client, error) {
	// Resolve API key
	resolvedKey := apiKey
	if resolvedKey == "" {
		envVar := apiKeyEnv
		if envVar == "" {
			envVar = DefaultAPIKeyEnv
		}
		resolvedKey = os.Getenv(envVar)
	}

	if resolvedKey == "" {
		return nil, fmt.Errorf("sweep API key not found: set %s environment variable or provide api_key in config", getEnvVarName(apiKeyEnv))
	}

	// Use custom transport to force HTTP/1.1 (avoid HTTP/2 stream errors)
	transport := &http.Transport{
		ForceAttemptHTTP2: false,
		// Setting TLSNextProto to empty map disables HTTP/2 for HTTPS
		TLSNextProto:        make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true, // We handle compression ourselves (Brotli)
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 2,
	}

	return &Client{
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		BaseURL: baseURL,
		APIKey:  resolvedKey,
	}, nil
}

// getEnvVarName returns the environment variable name to use
func getEnvVarName(apiKeyEnv string) string {
	if apiKeyEnv != "" {
		return apiKeyEnv
	}
	return DefaultAPIKeyEnv
}

// DoAutocomplete sends an autocomplete request to Sweep's hosted API
func (c *Client) DoAutocomplete(ctx context.Context, req *AutocompleteRequest) (*AutocompleteResponse, error) {
	defer logger.Trace("sweep.DoAutocomplete")()

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Compress with Brotli (quality 1, window size 22 - same as Zed)
	var compressedBody bytes.Buffer
	brotliWriter := brotli.NewWriterOptions(&compressedBody, brotli.WriterOptions{
		Quality: 1,
		LGWin:   22,
	})
	if _, err := brotliWriter.Write(jsonBody); err != nil {
		return nil, fmt.Errorf("failed to compress request: %w", err)
	}
	if err := brotliWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close brotli writer: %w", err)
	}

	url := c.BaseURL + DefaultAutocompletePath
	compressedBytes := compressedBody.Bytes()

	logger.Debug("sweep autocomplete request: URL=%s, file_path=%s, body_len=%d, compressed_len=%d", url, req.FilePath, len(jsonBody), len(compressedBytes))

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Small backoff to avoid hammering on transient transport issues.
			backoff := time.Duration(attempt-1) * 150 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(compressedBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
		httpReq.Header.Set("Connection", "keep-alive")
		httpReq.Header.Set("Content-Encoding", "br")

		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts && isRetryableTransportError(err) {
				logger.Debug("sweep autocomplete transient transport error (attempt %d/%d): %v", attempt, maxAttempts, err)
				continue
			}
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		respBody, statusCode, err := readSweepResponse(resp)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts && isRetryableResponseError(statusCode, err) {
				logger.Debug("sweep autocomplete transient response error (attempt %d/%d): %v", attempt, maxAttempts, err)
				continue
			}
			return nil, err
		}

		logger.Debug("sweep autocomplete raw response: %s", string(respBody))

		var autoResp AutocompleteResponse
		if err := json.Unmarshal(respBody, &autoResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		logger.Debug("sweep autocomplete response: id=%s, start=%d, end=%d, completion_len=%d", autoResp.AutocompleteID, autoResp.StartIndex, autoResp.EndIndex, len(autoResp.Completion))
		return &autoResp, nil
	}

	return nil, fmt.Errorf("failed to complete request after %d attempts: %w", maxAttempts, lastErr)
}

func readSweepResponse(resp *http.Response) ([]byte, int, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return body, resp.StatusCode, nil
}

func isRetryableResponseError(statusCode int, err error) bool {
	// Retry on transient read failures even when status is 200.
	if isRetryableTransportError(err) {
		return true
	}

	// Retry on 429 and 5xx.
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode >= 500 && statusCode <= 599 {
		return true
	}

	return false
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// net/http2 stream errors bubble up as stringy errors through net/http.
	msg := err.Error()
	if strings.Contains(msg, "stream error: stream ID") {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}

	return false
}

// SendMetrics sends metrics to Sweep's metrics endpoint (fire-and-forget)
func (c *Client) SendMetrics(ctx context.Context, req *MetricsRequest) {
	defer logger.Trace("sweep.SendMetrics")()

	body, err := json.Marshal(req)
	if err != nil {
		logger.Debug("sweep metrics: failed to marshal request: %v", err)
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+DefaultMetricsPath, bytes.NewReader(body))
	if err != nil {
		logger.Debug("sweep metrics: failed to create request: %v", err)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Fire-and-forget: don't wait for response
	go func() {
		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			logger.Debug("sweep metrics: failed to send: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			logger.Debug("sweep metrics: request failed with status %d: %s", resp.StatusCode, string(body))
		}
	}()
}
