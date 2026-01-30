package sweep

import (
	"context"
	"cursortab/assert"
	clientSweep "cursortab/client/sweep"
	"cursortab/types"
	"testing"
)

type fakeSweepClient struct {
	resp *clientSweep.AutocompleteResponse
	err  error

	lastReq *clientSweep.AutocompleteRequest
}

func (f *fakeSweepClient) DoAutocomplete(_ context.Context, req *clientSweep.AutocompleteRequest) (*clientSweep.AutocompleteResponse, error) {
	f.lastReq = req
	return f.resp, f.err
}

func TestBuildRecentChanges(t *testing.T) {
	req := &types.CompletionRequest{
		FileDiffHistories: []*types.FileDiffHistory{
			{
				FileName: "other.go",
				DiffHistory: []*types.DiffEntry{
					{Original: "old", Updated: "new"},
				},
			},
		},
	}

	changes := buildRecentChanges(req)
	assert.True(t, changes != "", "should build changes")
	assert.True(t, changes == "File: other.go:\n-old\n+new\n", "should match expected format")
}

func TestExtractRepoName(t *testing.T) {
	assert.Equal(t, "repo", extractRepoName("/home/me/repo/src/main.go"), "src parent")
	assert.Equal(t, "repo", extractRepoName("/home/me/repo/lib/x.go"), "lib parent")
	assert.Equal(t, "repo", extractRepoName("/home/me/repo/pkg/x.go"), "pkg parent")
	assert.Equal(t, "repo", extractRepoName("/home/me/repo/main.go"), "fallback parent")
}

func TestGetCompletion_ReplacesBytesAndBuildsLineRange(t *testing.T) {
	fc := &fakeSweepClient{resp: &clientSweep.AutocompleteResponse{Completion: "B2", StartIndex: 2, EndIndex: 3}}
	p := &hostedProvider{cfg: &types.ProviderConfig{}, client: fc}

	resp, err := p.GetCompletion(context.Background(), &types.CompletionRequest{
		FilePath:    "/home/me/repo/src/main.go",
		Lines:       []string{"a", "b", "c"},
		CursorRow:   2,
		CursorCol:   0,
		WorkspaceID: "",
	})
	assert.Nil(t, err, "no error")
	assert.Equal(t, 1, len(resp.Completions), "one completion")
	assert.Equal(t, 2, resp.Completions[0].StartLine, "start line")
	assert.Equal(t, 2, resp.Completions[0].EndLineInc, "end line inc")
	assert.Equal(t, "B2", resp.Completions[0].Lines[0], "replacement")

	assert.Equal(t, true, fc.lastReq.UseBytes, "must request byte offsets")
}

func TestGetCompletion_NoOpReturnsEmpty(t *testing.T) {
	// Replace "b" with "b" -> no change
	fc := &fakeSweepClient{resp: &clientSweep.AutocompleteResponse{Completion: "b", StartIndex: 2, EndIndex: 3}}
	p := &hostedProvider{cfg: &types.ProviderConfig{}, client: fc}

	resp, err := p.GetCompletion(context.Background(), &types.CompletionRequest{
		FilePath:  "main.go",
		Lines:     []string{"a", "b", "c"},
		CursorRow: 1,
		CursorCol: 0,
	})
	assert.Nil(t, err, "no error")
	assert.Equal(t, 0, len(resp.Completions), "no completions")
}
