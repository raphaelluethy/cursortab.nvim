package sweep

import (
	"context"
	"fmt"
	"strings"

	"cursortab/client/openai"
	"cursortab/client/sweep"
	"cursortab/provider"
	"cursortab/types"
)

// isHostedSweep checks if the URL indicates hosted Sweep (not localhost)
func isHostedSweep(url string) bool {
	return !strings.Contains(url, "localhost") && !strings.Contains(url, "127.0.0.1")
}

// NewProvider creates a new Sweep provider (local or hosted)
func NewProvider(config *types.ProviderConfig) (*provider.Provider, error) {
	if isHostedSweep(config.ProviderURL) {
		return newHostedProvider(config)
	}
	return newLocalProvider(config), nil
}

// newLocalProvider creates a provider for local Sweep (OpenAI-compatible)
func newLocalProvider(config *types.ProviderConfig) *provider.Provider {
	return &provider.Provider{
		Name:      "sweep-local",
		Config:    config,
		Client:    openai.NewClient(config.ProviderURL, config.CompletionPath),
		Streaming: true,
		Preprocessors: []provider.Preprocessor{
			provider.TrimContent(),
		},
		PromptBuilder: buildLocalPrompt,
		Postprocessors: []provider.Postprocessor{
			provider.RejectEmpty(),
			provider.ValidateAnchorPosition(0.25),
			provider.AnchorTruncation(0.75),
			parseLocalCompletion,
		},
	}
}

// newHostedProvider creates a provider for hosted Sweep (sweep.dev)
func newHostedProvider(config *types.ProviderConfig) (*provider.Provider, error) {
	client, err := sweep.NewClient(config.ProviderURL, config.APIKey, config.APIKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create sweep client: %w", err)
	}

	return &provider.Provider{
		Name:      "sweep-hosted",
		Config:    config,
		Client:    &hostedSweepClient{client: client},
		Streaming: false, // Hosted Sweep doesn't use streaming
		Preprocessors: []provider.Preprocessor{
			provider.TrimContent(),
		},
		PromptBuilder: buildHostedPrompt,
		Postprocessors: []provider.Postprocessor{
			provider.RejectEmpty(),
			provider.ValidateAnchorPosition(0.25),
			provider.AnchorTruncation(0.75),
			parseHostedCompletion,
		},
	}, nil
}

// hostedSweepClient wraps the sweep.Client to implement provider.Client interface
type hostedSweepClient struct {
	client *sweep.Client
}

func (h *hostedSweepClient) DoCompletion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error) {
	// Convert OpenAI request to Sweep request
	sweepReq := convertToSweepRequest(req)

	sweepResp, err := h.client.DoAutocomplete(ctx, sweepReq)
	if err != nil {
		return nil, err
	}

	// Convert Sweep response to OpenAI response format
	return convertToOpenAIResponse(sweepResp), nil
}

func (h *hostedSweepClient) DoStreamingCompletion(ctx context.Context, req *openai.CompletionRequest, maxLines int) (*openai.StreamResult, error) {
	// Hosted Sweep doesn't support streaming, so we use regular completion
	resp, err := h.DoCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &openai.StreamResult{}
	if len(resp.Choices) > 0 {
		result.Text = resp.Choices[0].Text
		result.FinishReason = resp.Choices[0].FinishReason
	}
	return result, nil
}

// convertToSweepRequest converts an OpenAI completion request to Sweep format
func convertToSweepRequest(req *openai.CompletionRequest) *sweep.AutocompleteRequest {
	// Extract file path and content from the prompt
	// The prompt format is specific to how Sweep expects it
	return &sweep.AutocompleteRequest{
		FilePath: extractFilePath(req.Prompt),
		Prefix:   req.Prompt,
		// Other fields will be populated based on context
	}
}

// convertToOpenAIResponse converts a Sweep response to OpenAI format
func convertToOpenAIResponse(sweepResp *sweep.AutocompleteResponse) *openai.CompletionResponse {
	return &openai.CompletionResponse{
		ID:      sweepResp.AutocompleteID,
		Object:  "text_completion",
		Created: 0,
		Model:   "sweep",
		Choices: []struct {
			Index        int    `json:"index"`
			Text         string `json:"text"`
			Logprobs     any    `json:"logprobs"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index:        0,
				Text:         sweepResp.Completion,
				Logprobs:     sweepResp.Logprobs,
				FinishReason: getFinishReason(sweepResp.FinishReason),
			},
		},
	}
}

func getFinishReason(reason *string) string {
	if reason != nil {
		return *reason
	}
	return "stop"
}

func extractFilePath(prompt string) string {
	// Extract file path from prompt if present
	// This is a simple extraction - may need refinement
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "<|file_sep|>") {
			parts := strings.Split(line, "/")
			if len(parts) > 1 {
				return parts[len(parts)-1]
			}
		}
	}
	return ""
}

// buildLocalPrompt builds the prompt for local Sweep (OpenAI-compatible)
func buildLocalPrompt(p *provider.Provider, ctx *provider.Context) *openai.CompletionRequest {
	req := ctx.Request
	var promptBuilder strings.Builder

	if len(req.Lines) == 0 {
		promptBuilder.WriteString("<|file_sep|>original/")
		promptBuilder.WriteString(req.FilePath)
		promptBuilder.WriteString("\n\n")
		promptBuilder.WriteString("<|file_sep|>current/")
		promptBuilder.WriteString(req.FilePath)
		promptBuilder.WriteString("\n\n")
		promptBuilder.WriteString("<|file_sep|>updated/")
		promptBuilder.WriteString(req.FilePath)
		promptBuilder.WriteString("\n")

		return &openai.CompletionRequest{
			Model:       p.Config.ProviderModel,
			Prompt:      promptBuilder.String(),
			Temperature: p.Config.ProviderTemperature,
			MaxTokens:   p.Config.ProviderMaxTokens,
			TopK:        p.Config.ProviderTopK,
			Stop:        []string{"<|file_sep|>", "</s>"},
			N:           1,
			Echo:        false,
		}
	}

	diffSection := buildDiffSection(req)
	originalLines := getTrimmedOriginalContent(req, ctx.WindowStart, len(ctx.TrimmedLines))

	if diffSection != "" {
		promptBuilder.WriteString(diffSection)
	}

	promptBuilder.WriteString("<|file_sep|>original/")
	promptBuilder.WriteString(req.FilePath)
	promptBuilder.WriteString("\n")
	promptBuilder.WriteString(strings.Join(originalLines, "\n"))
	promptBuilder.WriteString("\n")

	promptBuilder.WriteString("<|file_sep|>current/")
	promptBuilder.WriteString(req.FilePath)
	promptBuilder.WriteString("\n")
	promptBuilder.WriteString(strings.Join(ctx.TrimmedLines, "\n"))
	promptBuilder.WriteString("\n")

	promptBuilder.WriteString("<|file_sep|>updated/")
	promptBuilder.WriteString(req.FilePath)
	promptBuilder.WriteString("\n")

	return &openai.CompletionRequest{
		Model:       p.Config.ProviderModel,
		Prompt:      promptBuilder.String(),
		Temperature: p.Config.ProviderTemperature,
		MaxTokens:   p.Config.ProviderMaxTokens,
		TopK:        p.Config.ProviderTopK,
		Stop:        []string{"<|file_sep|>", "</s>"},
		N:           1,
		Echo:        false,
	}
}

// buildHostedPrompt builds the prompt for hosted Sweep
func buildHostedPrompt(p *provider.Provider, ctx *provider.Context) *openai.CompletionRequest {
	// For hosted Sweep, we need to build a different prompt format
	// that the sweep client will convert to the proper API request
	var promptBuilder strings.Builder

	// Write the current file content
	promptBuilder.WriteString(strings.Join(ctx.TrimmedLines, "\n"))

	return &openai.CompletionRequest{
		Model:       "sweep",
		Prompt:      promptBuilder.String(),
		Temperature: p.Config.ProviderTemperature,
		MaxTokens:   p.Config.ProviderMaxTokens,
		TopK:        p.Config.ProviderTopK,
		Stop:        []string{},
		N:           1,
		Echo:        false,
	}
}

func buildDiffSection(req *types.CompletionRequest) string {
	if len(req.FileDiffHistories) == 0 {
		return ""
	}

	var builder strings.Builder

	for _, fileHistory := range req.FileDiffHistories {
		for _, diffEntry := range fileHistory.DiffHistory {
			if diffEntry.Original == "" && diffEntry.Updated == "" {
				continue
			}

			builder.WriteString("<|file_sep|>")
			builder.WriteString(fileHistory.FileName)
			builder.WriteString(".diff\n")
			builder.WriteString("original:\n")
			builder.WriteString(diffEntry.Original)
			builder.WriteString("\nupdated:\n")
			builder.WriteString(diffEntry.Updated)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func getTrimmedOriginalContent(req *types.CompletionRequest, trimOffset, lineCount int) []string {
	sourceLines := req.PreviousLines
	if len(sourceLines) == 0 {
		sourceLines = req.Lines
	}

	windowStart := trimOffset
	windowEnd := trimOffset + lineCount

	if windowStart >= len(sourceLines) {
		return []string{}
	}
	if windowEnd > len(sourceLines) {
		windowEnd = len(sourceLines)
	}

	return sourceLines[windowStart:windowEnd]
}

func parseLocalCompletion(p *provider.Provider, ctx *provider.Context) (*types.CompletionResponse, bool) {
	completionText := ctx.Result.Text
	req := ctx.Request

	completionText = strings.TrimSuffix(completionText, "<|file_sep|>")
	completionText = strings.TrimSuffix(completionText, "</s>")
	completionText = strings.TrimRight(completionText, " \t\n\r")

	windowStart := ctx.WindowStart
	windowEnd := ctx.WindowEnd
	if windowStart < 0 {
		windowStart = 0
	}
	if windowEnd > len(req.Lines) {
		windowEnd = len(req.Lines)
	}
	if windowStart >= windowEnd || windowStart >= len(req.Lines) {
		return p.EmptyResponse(), true
	}

	oldLines := req.Lines[windowStart:windowEnd]
	oldText := strings.TrimRight(strings.Join(oldLines, "\n"), " \t\n\r")

	if completionText == oldText {
		return p.EmptyResponse(), true
	}

	newLines := strings.Split(completionText, "\n")

	endLineInc := ctx.EndLineInc
	if endLineInc == 0 {
		endLineInc = min(windowStart+len(newLines), windowEnd)
	}

	return p.BuildCompletion(ctx, windowStart+1, endLineInc, newLines)
}

func parseHostedCompletion(p *provider.Provider, ctx *provider.Context) (*types.CompletionResponse, bool) {
	// For hosted Sweep, the completion is already processed
	// We just need to convert it to the right format
	return parseLocalCompletion(p, ctx)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
