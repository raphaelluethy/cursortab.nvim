package sweep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	return &Client{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+DefaultAutocompletePath, &compressedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Connection", "keep-alive")
	httpReq.Header.Set("Content-Encoding", "br")

	logger.Debug("sweep autocomplete request: URL=%s, file_path=%s, body_len=%d, compressed_len=%d", c.BaseURL+DefaultAutocompletePath, req.FilePath, len(jsonBody), compressedBody.Len())

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var autoResp AutocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&autoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Debug("sweep autocomplete response: id=%s, completion_len=%d", autoResp.AutocompleteID, len(autoResp.Completion))

	return &autoResp, nil
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
