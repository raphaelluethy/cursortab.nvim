package sweep

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	// Check if API key is available before attempting to create client
	if config.APIKey == "" {
		envVar := config.APIKeyEnv
		if envVar == "" {
			envVar = sweep.DefaultAPIKeyEnv
		}
		if os.Getenv(envVar) == "" {
			return nil, fmt.Errorf("hosted Sweep requires API key: set %s environment variable or provide api_key in config", envVar)
		}
	}

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
	// Hosted Sweep doesn't support streaming, so we call the Sweep API directly
	sweepReq := convertToSweepRequest(req)

	sweepResp, err := h.client.DoAutocomplete(ctx, sweepReq)
	if err != nil {
		return nil, err
	}

	return &openai.StreamResult{
		Text:         sweepResp.Completion,
		FinishReason: getFinishReason(sweepResp.FinishReason),
		StartIndex:   sweepResp.StartIndex,
		EndIndex:     sweepResp.EndIndex,
	}, nil
}

// sweepRequestContext holds the context needed to build a Sweep request
// This is passed through the OpenAI request via a custom field
type sweepRequestContext struct {
	FilePath             string
	FileContents         string
	OriginalFileContents string
	CursorPosition       int
	RecentChanges        string
	RepoName             string
}

// convertToSweepRequest converts an OpenAI completion request to Sweep format
func convertToSweepRequest(req *openai.CompletionRequest) *sweep.AutocompleteRequest {
	// The Prompt field now contains JSON-encoded sweep request context
	var ctx sweepRequestContext
	if err := json.Unmarshal([]byte(req.Prompt), &ctx); err != nil {
		// Fallback for backwards compatibility
		return &sweep.AutocompleteRequest{
			FilePath:             "",
			FileContents:         req.Prompt,
			OriginalFileContents: req.Prompt,
			CursorPosition:       len(req.Prompt),
			DebugInfo:            "cursortab.nvim",
			RepoName:             "unknown",
			ChangesAboveCursor:   true,
			UseBytes:             true,
			FileChunks:           []sweep.FileChunk{},
			RetrievalChunks:      []sweep.FileChunk{},
			RecentUserActions:    []sweep.UserAction{},
		}
	}

	return &sweep.AutocompleteRequest{
		DebugInfo:            "cursortab.nvim",
		RepoName:             ctx.RepoName,
		FilePath:             ctx.FilePath,
		FileContents:         ctx.FileContents,
		OriginalFileContents: ctx.OriginalFileContents,
		CursorPosition:       ctx.CursorPosition,
		RecentChanges:        ctx.RecentChanges,
		ChangesAboveCursor:   true,
		MultipleSuggestions:  false,
		PrivacyModeEnabled:   false,
		UseBytes:             true,
		FileChunks:           []sweep.FileChunk{},
		RetrievalChunks:      []sweep.FileChunk{},
		RecentUserActions:    []sweep.UserAction{},
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
	req := ctx.Request
	fileContents := strings.Join(ctx.TrimmedLines, "\n")

	// Get original file contents
	originalContents := fileContents
	if len(req.PreviousLines) > 0 {
		originalContents = strings.Join(req.PreviousLines, "\n")
	}

	// Calculate cursor position as byte offset
	cursorPosition := 0
	cursorLine := req.CursorRow - 1 // Convert to 0-indexed (CursorRow is 1-indexed)
	cursorCol := req.CursorCol      // CursorCol is already 0-indexed

	for i, line := range ctx.TrimmedLines {
		if i < cursorLine {
			cursorPosition += len(line) + 1 // +1 for newline
		} else if i == cursorLine {
			if cursorCol > len(line) {
				cursorCol = len(line)
			}
			cursorPosition += cursorCol
			break
		}
	}

	// Build recent changes from diff history
	recentChanges := buildRecentChanges(req)

	// Get repo name from file path
	repoName := extractRepoName(req.FilePath)

	// Encode context as JSON for the sweep client
	sweepCtx := sweepRequestContext{
		FilePath:             req.FilePath,
		FileContents:         fileContents,
		OriginalFileContents: originalContents,
		CursorPosition:       cursorPosition,
		RecentChanges:        recentChanges,
		RepoName:             repoName,
	}

	contextJSON, _ := json.Marshal(sweepCtx)

	return &openai.CompletionRequest{
		Model:       "sweep",
		Prompt:      string(contextJSON),
		Temperature: p.Config.ProviderTemperature,
		MaxTokens:   p.Config.ProviderMaxTokens,
		TopK:        p.Config.ProviderTopK,
		Stop:        []string{},
		N:           1,
		Echo:        false,
	}
}

// buildRecentChanges builds the recent changes string from diff history
func buildRecentChanges(req *types.CompletionRequest) string {
	if len(req.FileDiffHistories) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, fileHistory := range req.FileDiffHistories {
		for _, diffEntry := range fileHistory.DiffHistory {
			if diffEntry.Original == "" && diffEntry.Updated == "" {
				continue
			}
			fmt.Fprintf(&builder, "File: %s:\n", fileHistory.FileName)
			if diffEntry.Original != "" {
				fmt.Fprintf(&builder, "-%s\n", diffEntry.Original)
			}
			if diffEntry.Updated != "" {
				fmt.Fprintf(&builder, "+%s\n", diffEntry.Updated)
			}
		}
	}
	return builder.String()
}

// extractRepoName extracts the repository name from a file path
func extractRepoName(filePath string) string {
	parts := strings.Split(filePath, "/")
	for i, part := range parts {
		if part == "src" || part == "lib" || part == "app" || part == "pkg" {
			if i > 0 {
				return parts[i-1]
			}
		}
	}
	// Fallback: use parent directory name
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return "unknown"
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
	// The Sweep response contains completion text and start/end indices
	// The indices are byte offsets in the file
	completionText := ctx.Result.Text
	if completionText == "" {
		return p.EmptyResponse(), true
	}

	req := ctx.Request
	fileContents := strings.Join(req.Lines, "\n")

	// Get start and end indices from the result metadata
	startIndex := ctx.Result.StartIndex
	endIndex := ctx.Result.EndIndex

	// If indices are not set, fall back to local parsing
	if startIndex == 0 && endIndex == 0 {
		return parseLocalCompletion(p, ctx)
	}

	// Extract the old text being replaced
	if startIndex > len(fileContents) {
		startIndex = len(fileContents)
	}
	if endIndex > len(fileContents) {
		endIndex = len(fileContents)
	}
	if startIndex > endIndex {
		startIndex = endIndex
	}

	oldText := fileContents[startIndex:endIndex]

	// If the completion is the same as the old text, no change needed
	if completionText == oldText {
		return p.EmptyResponse(), true
	}

	// Convert byte offsets to line numbers
	startLine := 1
	endLine := 1
	byteCount := 0
	for i, line := range req.Lines {
		lineLen := len(line) + 1 // +1 for newline
		if byteCount+lineLen > startIndex && startLine == 1 {
			startLine = i + 1
		}
		if byteCount+lineLen >= endIndex {
			endLine = i + 1
			break
		}
		byteCount += lineLen
	}

	// Build the new lines
	newLines := strings.Split(completionText, "\n")

	return p.BuildCompletion(ctx, startLine, endLine, newLines)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
