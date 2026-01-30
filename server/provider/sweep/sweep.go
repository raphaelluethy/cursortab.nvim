package sweep

import (
	"context"
	"fmt"
	"os"
	"strings"

	clientSweep "cursortab/client/sweep"
	"cursortab/engine"
	"cursortab/provider"
	"cursortab/types"
)

// hostedProvider implements engine.Provider using Sweep's hosted next-edit API.
type hostedProvider struct {
	cfg    *types.ProviderConfig
	client sweepClient
}

var _ engine.Provider = (*hostedProvider)(nil)

type sweepClient interface {
	DoAutocomplete(ctx context.Context, req *clientSweep.AutocompleteRequest) (*clientSweep.AutocompleteResponse, error)
}

func NewProvider(cfg *types.ProviderConfig) (engine.Provider, error) {
	// Check if API key is available before attempting to create client
	if cfg.APIKey == "" {
		envVar := cfg.APIKeyEnv
		if envVar == "" {
			envVar = clientSweep.DefaultAPIKeyEnv
		}
		if os.Getenv(envVar) == "" {
			return nil, fmt.Errorf("hosted Sweep requires API key: set %s environment variable or provide api_key in config", envVar)
		}
	}

	c, err := clientSweep.NewClient(cfg.ProviderURL, cfg.APIKey, cfg.APIKeyEnv)
	if err != nil {
		return nil, err
	}

	return &hostedProvider{cfg: cfg, client: c}, nil
}

func (p *hostedProvider) GetCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	// Send complete file contents (like Zed does)
	fileContents := strings.Join(req.Lines, "\n")

	// Get original file contents (before current edits)
	originalContents := fileContents
	if len(req.PreviousLines) > 0 {
		originalContents = strings.Join(req.PreviousLines, "\n")
	}

	// Calculate cursor position as byte offset in the FULL file
	cursorPosition := 0
	cursorLine := req.CursorRow - 1 // Convert to 0-indexed
	cursorCol := req.CursorCol      // Already 0-indexed

	for i, line := range req.Lines {
		if i < cursorLine {
			cursorPosition += len(line) + 1 // +1 for newline
			continue
		}
		if i == cursorLine {
			if cursorCol > len(line) {
				cursorCol = len(line)
			}
			cursorPosition += cursorCol
			break
		}
	}

	sweepReq := &clientSweep.AutocompleteRequest{
		DebugInfo:            "cursortab.nvim",
		RepoName:             extractRepoName(req.FilePath),
		Branch:               nil,
		FilePath:             req.FilePath,
		FileContents:         fileContents,
		RecentChanges:        buildRecentChanges(req),
		CursorPosition:       cursorPosition,
		OriginalFileContents: originalContents,
		FileChunks:           []clientSweep.FileChunk{},
		RetrievalChunks:      []clientSweep.FileChunk{},
		RecentUserActions:    []clientSweep.UserAction{},
		MultipleSuggestions:  false,
		PrivacyModeEnabled:   false,
		ChangesAboveCursor:   true,
		UseBytes:             true,
	}

	sweepResp, err := p.client.DoAutocomplete(ctx, sweepReq)
	if err != nil {
		return nil, err
	}

	completionText := sweepResp.Completion
	startIndex := sweepResp.StartIndex
	endIndex := sweepResp.EndIndex

	// If no completion, nothing to do
	if completionText == "" && startIndex == 0 && endIndex == 0 {
		return emptyResponse(), nil
	}

	// Apply byte replacement to get full updated content
	if startIndex > len(fileContents) {
		startIndex = len(fileContents)
	}
	if endIndex > len(fileContents) {
		endIndex = len(fileContents)
	}
	if startIndex > endIndex {
		startIndex = endIndex
	}
	updatedContent := fileContents[:startIndex] + completionText + fileContents[endIndex:]

	oldLines := req.Lines
	newLines := strings.Split(updatedContent, "\n")

	// Find first differing line
	firstDiff := 0
	for firstDiff < len(oldLines) && firstDiff < len(newLines) {
		if oldLines[firstDiff] != newLines[firstDiff] {
			break
		}
		firstDiff++
	}

	// Find last differing line (from the end)
	lastDiffOld := len(oldLines) - 1
	lastDiffNew := len(newLines) - 1
	for lastDiffOld > firstDiff && lastDiffNew > firstDiff {
		if oldLines[lastDiffOld] != newLines[lastDiffNew] {
			break
		}
		lastDiffOld--
		lastDiffNew--
	}

	// If no differences found, return empty
	if firstDiff >= len(oldLines) && firstDiff >= len(newLines) {
		return emptyResponse(), nil
	}

	changedNewLines := newLines[firstDiff : lastDiffNew+1]
	startLine := firstDiff + 1
	endLineInc := lastDiffOld + 1
	if endLineInc < startLine {
		endLineInc = startLine
	}

	// Match generic provider no-op detection behavior
	if endLineInc <= len(req.Lines) && provider.IsNoOpReplacement(changedNewLines, req.Lines[startLine-1:endLineInc]) {
		return emptyResponse(), nil
	}

	return &types.CompletionResponse{
		Completions: []*types.Completion{
			{
				StartLine:  startLine,
				EndLineInc: endLineInc,
				Lines:      changedNewLines,
			},
		},
		CursorTarget: nil,
	}, nil
}

func emptyResponse() *types.CompletionResponse {
	return &types.CompletionResponse{Completions: []*types.Completion{}, CursorTarget: nil}
}

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

func extractRepoName(filePath string) string {
	parts := strings.Split(filePath, "/")
	for i, part := range parts {
		if part == "src" || part == "lib" || part == "app" || part == "pkg" {
			if i > 0 {
				return parts[i-1]
			}
		}
	}
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return "unknown"
}
