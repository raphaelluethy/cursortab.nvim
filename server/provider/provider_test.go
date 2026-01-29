package provider

import (
	"testing"
)

// TestContext_TrimmedContextInterface verifies that Context implements
// the methods needed for TrimmedContext interface in engine.
// This ensures the engine can extract trim info from provider context.
func TestContext_TrimmedContextInterface(t *testing.T) {
	ctx := &Context{
		WindowStart:  20,
		TrimmedLines: []string{"line 1", "line 2", "line 3"},
	}

	// These methods should exist and return correct values
	if ctx.GetWindowStart() != 20 {
		t.Errorf("GetWindowStart() = %d, expected 20", ctx.GetWindowStart())
	}

	lines := ctx.GetTrimmedLines()
	if len(lines) != 3 {
		t.Errorf("GetTrimmedLines() length = %d, expected 3", len(lines))
	}
	if lines[0] != "line 1" {
		t.Errorf("GetTrimmedLines()[0] = %q, expected %q", lines[0], "line 1")
	}
}

// TestContext_EmptyTrimmedLines verifies behavior when no trimming occurred.
func TestContext_EmptyTrimmedLines(t *testing.T) {
	ctx := &Context{
		WindowStart:  0,
		TrimmedLines: nil,
	}

	if ctx.GetWindowStart() != 0 {
		t.Errorf("GetWindowStart() = %d, expected 0", ctx.GetWindowStart())
	}

	lines := ctx.GetTrimmedLines()
	if lines != nil {
		t.Errorf("GetTrimmedLines() should be nil when not trimmed")
	}
}
