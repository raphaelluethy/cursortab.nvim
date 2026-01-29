package text

import (
	"testing"
)

func TestIncrementalDiffBuilder_BasicModification(t *testing.T) {
	oldLines := []string{"hello world", "foo bar", "baz qux"}
	builder := NewIncrementalDiffBuilder(oldLines)

	// Add lines that match/modify the old lines
	change1 := builder.AddLine("hello world") // exact match
	if change1 != nil {
		t.Error("expected no change for exact match, got", change1)
	}

	change2 := builder.AddLine("foo baz") // modification
	if change2 == nil {
		t.Fatal("expected change for modification")
	}
	if change2.Type != ChangeReplaceChars && change2.Type != ChangeModification {
		t.Errorf("expected modification type, got %v", change2.Type)
	}

	change3 := builder.AddLine("baz qux") // exact match
	if change3 != nil {
		t.Error("expected no change for exact match, got", change3)
	}

	// Verify final state
	if len(builder.Changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(builder.Changes))
	}
}

func TestIncrementalDiffBuilder_Addition(t *testing.T) {
	oldLines := []string{"line 1", "line 2"}
	builder := NewIncrementalDiffBuilder(oldLines)

	builder.AddLine("line 1") // match
	builder.AddLine("new line") // addition
	builder.AddLine("line 2") // match

	if len(builder.Changes) != 1 {
		t.Errorf("expected 1 change (addition), got %d", len(builder.Changes))
	}

	// Find the addition
	for _, change := range builder.Changes {
		if change.Type == ChangeAddition {
			if change.Content != "new line" {
				t.Errorf("expected 'new line' content, got %q", change.Content)
			}
			return
		}
	}
	t.Error("did not find addition change")
}

func TestIncrementalDiffBuilder_MultipleAdditions(t *testing.T) {
	oldLines := []string{"a", "b"}
	builder := NewIncrementalDiffBuilder(oldLines)

	builder.AddLine("a")    // match
	builder.AddLine("x")    // addition
	builder.AddLine("y")    // addition
	builder.AddLine("b")    // match

	if len(builder.Changes) != 2 {
		t.Errorf("expected 2 changes (additions), got %d", len(builder.Changes))
	}
}

func TestIncrementalStageBuilder_SingleStage(t *testing.T) {
	oldLines := []string{"line 1", "line 2", "line 3"}
	builder := NewIncrementalStageBuilder(
		oldLines,
		1,  // baseLineOffset
		3,  // proximityThreshold
		0,  // viewportTop (disabled)
		0,  // viewportBottom (disabled)
		1,  // cursorRow
		"test.go",
	)

	// Add modified lines that should all be in the same stage
	builder.AddLine("line 1 modified") // modification
	builder.AddLine("line 2 modified") // modification
	builder.AddLine("line 3")          // match

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	if len(result.Stages) != 1 {
		t.Errorf("expected 1 stage, got %d", len(result.Stages))
	}

	stage := result.Stages[0]
	if len(stage.Changes) != 2 {
		t.Errorf("expected 2 changes in stage, got %d", len(stage.Changes))
	}
}

func TestIncrementalStageBuilder_MultipleStages(t *testing.T) {
	oldLines := []string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
		"line 6",
		"line 7",
		"line 8",
		"line 9",
		"line 10",
	}
	builder := NewIncrementalStageBuilder(
		oldLines,
		1,  // baseLineOffset
		2,  // proximityThreshold (small to force multiple stages)
		0,  // viewportTop
		0,  // viewportBottom
		1,  // cursorRow
		"test.go",
	)

	// Add lines with gaps > proximityThreshold to create multiple stages
	builder.AddLine("line 1 modified") // modification at line 1
	builder.AddLine("line 2")          // match
	builder.AddLine("line 3")          // match
	builder.AddLine("line 4")          // match
	builder.AddLine("line 5")          // match
	builder.AddLine("line 6 modified") // modification at line 6 (gap > 2)
	builder.AddLine("line 7")          // match
	builder.AddLine("line 8")          // match
	builder.AddLine("line 9")          // match
	builder.AddLine("line 10")         // match

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	if len(result.Stages) != 2 {
		t.Errorf("expected 2 stages (due to gap > proximityThreshold), got %d", len(result.Stages))
	}
}

func TestIncrementalStageBuilder_StageFinalizationOnGap(t *testing.T) {
	oldLines := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	builder := NewIncrementalStageBuilder(
		oldLines,
		1,  // baseLineOffset
		2,  // proximityThreshold
		0, 0, // viewport disabled
		1,  // cursorRow
		"test.go",
	)

	// Line 1: modification, starts a stage
	finalized := builder.AddLine("a modified")
	if finalized != nil {
		t.Error("should not finalize on first change")
	}

	// Lines 2-3: matches (gap building up but not exceeding threshold)
	finalized = builder.AddLine("b") // gap = 1, threshold = 2
	if finalized != nil {
		t.Error("should not finalize when gap <= threshold")
	}
	finalized = builder.AddLine("c") // gap = 2, still not > threshold
	if finalized != nil {
		t.Error("should not finalize when gap == threshold")
	}

	// Line 4: match, but gap now exceeds threshold (gap = 3 > 2)
	// Stage should finalize even without a new change
	finalized = builder.AddLine("d")
	if finalized == nil {
		t.Error("should finalize stage when gap exceeds threshold (even without new change)")
	} else {
		if len(finalized.Changes) != 1 {
			t.Errorf("finalized stage should have 1 change, got %d", len(finalized.Changes))
		}
	}

	// Line 5: modification starts a new stage
	finalized = builder.AddLine("e modified")
	if finalized != nil {
		t.Error("should not finalize on first change of new stage")
	}
}

func TestIncrementalDiffBuilder_SimilarityMatching(t *testing.T) {
	oldLines := []string{
		"func hello() {",
		"    return world",
		"}",
	}
	builder := NewIncrementalDiffBuilder(oldLines)

	// Modified version with similar structure
	change1 := builder.AddLine("func hello() {") // exact match
	if change1 != nil {
		t.Error("expected exact match")
	}

	change2 := builder.AddLine("    return world + 1") // modification
	if change2 == nil {
		t.Fatal("expected modification")
	}
	if change2.OldLineNum != 2 {
		t.Errorf("expected to match old line 2, got %d", change2.OldLineNum)
	}

	change3 := builder.AddLine("}") // exact match
	if change3 != nil {
		t.Error("expected exact match for closing brace")
	}
}

func TestIncrementalStageBuilder_ViewportBoundary(t *testing.T) {
	// Use more distinct line content to avoid similarity matching issues
	oldLines := []string{
		"line one",
		"line two",
		"line three",
		"line four",
		"line five",
		"line six",
		"line seven",
		"line eight",
		"line nine",
		"line ten",
	}
	builder := NewIncrementalStageBuilder(
		oldLines,
		1,  // baseLineOffset
		10, // proximityThreshold (high to prevent gap-based splits)
		1,  // viewportTop
		5,  // viewportBottom (first 5 lines visible)
		3,  // cursorRow
		"test.go",
	)

	// Modifications in viewport (lines 1-5)
	builder.AddLine("line one modified") // in viewport, buffer line = 1
	builder.AddLine("line two")
	builder.AddLine("line three")
	builder.AddLine("line four")
	builder.AddLine("line five modified") // still in viewport, buffer line = 5

	// Verify current stage is in viewport before adding line 6
	if !builder.IsCurrentStageInViewport() {
		t.Error("current stage should be in viewport before line 6")
	}

	// Add remaining lines to complete the sequence
	builder.AddLine("line six modified") // outside viewport, buffer line = 6
	builder.AddLine("line seven")
	builder.AddLine("line eight")
	builder.AddLine("line nine")
	builder.AddLine("line ten")

	// Finalize and check we got multiple stages
	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	// Should have 2 stages: one for viewport changes (1, 5), one for outside (6)
	if len(result.Stages) < 2 {
		t.Errorf("expected at least 2 stages due to viewport boundary, got %d", len(result.Stages))
		for i, stage := range result.Stages {
			t.Logf("Stage %d: BufferStart=%d, BufferEnd=%d, changes=%d",
				i, stage.BufferStart, stage.BufferEnd, len(stage.Changes))
		}
	}
}

func TestIncrementalDiffBuilder_EmptyOldLines(t *testing.T) {
	builder := NewIncrementalDiffBuilder([]string{})

	change := builder.AddLine("new content")
	if change == nil {
		t.Fatal("expected addition change")
	}
	if change.Type != ChangeAddition {
		t.Errorf("expected addition type, got %v", change.Type)
	}
}

// TestIncrementalStageBuilder_BaseLineOffset verifies that BufferStart/BufferEnd
// are correctly offset when the provider trims content (baseLineOffset > 1).
// This simulates when the model only sees a window of the file, not the full file.
func TestIncrementalStageBuilder_BaseLineOffset(t *testing.T) {
	// Simulate a trimmed window: model sees lines 20-25 of original file
	// oldLines here represents the TRIMMED content (what model sees)
	oldLines := []string{
		"  if (article.tags === null) {",  // buffer line 20
		"    article.tags = tag;",          // buffer line 21
		"  } else {",                       // buffer line 22
		"    article.tags = concat(tag);",  // buffer line 23
		"  }",                              // buffer line 24
	}

	baseLineOffset := 20 // Window starts at buffer line 20

	builder := NewIncrementalStageBuilder(
		oldLines,
		baseLineOffset,
		3,     // proximityThreshold
		15, 30, // viewport (lines 15-30 visible)
		22,    // cursorRow (in middle of window)
		"test.ts",
	)

	// Model outputs modified content
	builder.AddLine("  if (article.tags === null) {")      // match
	builder.AddLine("    article.tags = [tag];")           // modification
	builder.AddLine("  } else {")                          // match
	builder.AddLine("    article.tags.push(tag);")         // modification
	builder.AddLine("  }")                                 // match

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	if len(result.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(result.Stages))
	}

	stage := result.Stages[0]

	// BufferStart should be 21 (baseLineOffset + 1 for the second line where change is)
	// BufferEnd should be 24 (baseLineOffset + 3 for line 4 where last change is)
	// The key test: these should NOT be 1-5, they should be offset by baseLineOffset
	if stage.BufferStart < baseLineOffset {
		t.Errorf("BufferStart=%d should be >= baseLineOffset=%d", stage.BufferStart, baseLineOffset)
	}
	if stage.BufferEnd < baseLineOffset {
		t.Errorf("BufferEnd=%d should be >= baseLineOffset=%d", stage.BufferEnd, baseLineOffset)
	}

	// More specific check: changes are on lines 2 and 4 of input (1-indexed)
	// So BufferStart should be baseLineOffset + 1 = 21
	// And BufferEnd should be baseLineOffset + 3 = 23 (line 4 of 5)
	expectedStart := 21 // Line 2 of trimmed = buffer line 21
	expectedEnd := 23   // Line 4 of trimmed = buffer line 23

	if stage.BufferStart != expectedStart {
		t.Errorf("BufferStart=%d, expected %d (baseLineOffset + relative position)",
			stage.BufferStart, expectedStart)
	}
	if stage.BufferEnd != expectedEnd {
		t.Errorf("BufferEnd=%d, expected %d (baseLineOffset + relative position)",
			stage.BufferEnd, expectedEnd)
	}

	// Verify changes exist
	if len(stage.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(stage.Changes))
	}
}

// TestIncrementalStageBuilder_BaseLineOffsetWithGap tests that gap detection
// works correctly when baseLineOffset > 1 and stages finalize mid-stream.
func TestIncrementalStageBuilder_BaseLineOffsetWithGap(t *testing.T) {
	// Simulate window starting at line 50
	oldLines := []string{
		"line A", // buffer line 50
		"line B", // buffer line 51
		"line C", // buffer line 52
		"line D", // buffer line 53
		"line E", // buffer line 54
		"line F", // buffer line 55
		"line G", // buffer line 56
		"line H", // buffer line 57
	}

	baseLineOffset := 50

	builder := NewIncrementalStageBuilder(
		oldLines,
		baseLineOffset,
		2,     // proximityThreshold
		40, 60, // viewport
		52,    // cursorRow
		"test.go",
	)

	// Change on line 1 (buffer line 50)
	finalized := builder.AddLine("line A modified")
	if finalized != nil {
		t.Error("should not finalize on first change")
	}

	// Lines 2-4: no changes, building gap
	builder.AddLine("line B") // gap = 1
	builder.AddLine("line C") // gap = 2

	// Line 4: gap > threshold, should finalize
	finalized = builder.AddLine("line D") // gap = 3 > 2
	if finalized == nil {
		t.Fatal("expected stage to finalize on gap")
	}

	// Check the finalized stage has correct buffer positions
	if finalized.BufferStart != 50 {
		t.Errorf("finalized stage BufferStart=%d, expected 50", finalized.BufferStart)
	}
	// BufferEnd should also be 50 since there's only one change on line 1
	if finalized.BufferEnd != 50 {
		t.Errorf("finalized stage BufferEnd=%d, expected 50", finalized.BufferEnd)
	}
}

// TestIncrementalStageBuilder_ScatteredBufferLines tests the hypothesis that
// gap detection is based on NEW line numbers, not BUFFER line numbers.
// This causes changes that map to far-apart buffer lines to be grouped into one stage.
func TestIncrementalStageBuilder_ScatteredBufferLines(t *testing.T) {
	// Simulate a scenario where similarity matching maps new lines to scattered old lines.
	// Old lines represent different functions in a file.
	oldLines := []string{
		"function foo() {",           // line 1 (buffer 52)
		"  const x = 1;",             // line 2 (buffer 53)
		"  return x;",                // line 3 (buffer 54)
		"}",                          // line 4 (buffer 55)
		"",                           // line 5 (buffer 56)
		"function bar() {",           // line 6 (buffer 57)
		"  const y = 2;",             // line 7 (buffer 58)
		"  return y;",                // line 8 (buffer 59)
		"}",                          // line 9 (buffer 60)
		"",                           // line 10 (buffer 61)
		"function baz() {",           // line 11 (buffer 62)
		"  const z = 3;",             // line 12 (buffer 63)
		"  return z;",                // line 13 (buffer 64)
		"}",                          // line 14 (buffer 65)
	}

	baseLineOffset := 52

	builder := NewIncrementalStageBuilder(
		oldLines,
		baseLineOffset,
		3,      // proximityThreshold - gaps > 3 should split stages
		40, 80, // viewport
		58,     // cursorRow (in middle)
		"test.ts",
	)

	// Model outputs content where:
	// - Line 1 matches old line 1 (exact)
	// - Line 2 is a MODIFICATION of old line 2 -> buffer 53
	// - Line 3 matches old line 3 (exact)
	// - Line 4 matches old line 4 (exact)
	// - Line 5 matches old line 5 (exact)
	// - Line 6 matches old line 6 (exact)
	// - Line 7 is a MODIFICATION of old line 7 -> buffer 58 (buffer gap = 5!)
	// - Line 8 matches old line 8 (exact)
	// - etc.

	builder.AddLine("function foo() {")     // match line 1
	builder.AddLine("  const x = 100;")     // MODIFY line 2 -> buffer 53
	builder.AddLine("  return x;")          // match line 3
	builder.AddLine("}")                    // match line 4
	builder.AddLine("")                     // match line 5
	builder.AddLine("function bar() {")     // match line 6
	builder.AddLine("  const y = 200;")     // MODIFY line 7 -> buffer 58 (buffer gap = 5!)
	builder.AddLine("  return y;")          // match line 8
	builder.AddLine("}")                    // match line 9
	builder.AddLine("")                     // match line 10
	builder.AddLine("function baz() {")     // match line 11
	builder.AddLine("  const z = 300;")     // MODIFY line 12 -> buffer 63 (buffer gap = 5!)
	builder.AddLine("  return z;")          // match line 13
	builder.AddLine("}")                    // match line 14

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	// BUG HYPOTHESIS: Currently, all 3 changes get grouped into ONE stage
	// because the NEW line gaps are small (lines 2, 7, 12 -> gaps of 5 in new lines)
	// even though BUFFER line gaps are large (53, 58, 63 -> gaps of 5 in buffer lines).
	//
	// With proximityThreshold=3, buffer gaps of 5 SHOULD split into separate stages.
	// EXPECTED (correct): 3 stages (one per function)
	// ACTUAL (bug): 1 stage spanning buffer lines 53-63

	t.Logf("Number of stages: %d", len(result.Stages))
	for i, stage := range result.Stages {
		t.Logf("Stage %d: BufferStart=%d, BufferEnd=%d, changes=%d",
			i, stage.BufferStart, stage.BufferEnd, len(stage.Changes))
	}

	// This test SHOULD pass after the fix, but will FAIL with current code
	if len(result.Stages) < 3 {
		t.Errorf("HYPOTHESIS CONFIRMED: Expected 3 stages (buffer gaps > threshold), got %d. "+
			"Gap detection is using NEW line numbers instead of BUFFER line numbers.",
			len(result.Stages))
	}
}

// TestIncrementalStageBuilder_SimilarityMatchingScatter tests that similarity matching
// can cause changes to map to far-apart buffer lines when model output is wrong.
func TestIncrementalStageBuilder_SimilarityMatchingScatter(t *testing.T) {
	// Simulate model outputting content from wrong function.
	// Old lines have similar patterns in different locations.
	oldLines := []string{
		"  article.title = title;",     // line 1 (buffer 20) - in setTitle()
		"  article.author = author;",   // line 2 (buffer 21)
		"  return true;",               // line 3 (buffer 22)
		"}",                            // line 4 (buffer 23)
		"",                             // line 5 (buffer 24)
		"function updateTags() {",      // line 6 (buffer 25)
		"  article.tags = tags;",       // line 7 (buffer 26) - similar to line 1!
		"  article.count = count;",     // line 8 (buffer 27) - similar to line 2!
		"  return true;",               // line 9 (buffer 28)
		"}",                            // line 10 (buffer 29)
	}

	baseLineOffset := 20

	builder := NewIncrementalStageBuilder(
		oldLines,
		baseLineOffset,
		3,      // proximityThreshold
		15, 35, // viewport
		23,     // cursorRow
		"test.ts",
	)

	// Model outputs lines that SHOULD modify lines 1-3 (setTitle function)
	// but similarity matching might find matches at lines 7-9 (updateTags function)
	// if the content is similar enough.

	// Intentionally use content that's similar to BOTH locations
	builder.AddLine("  article.name = name;")       // Similar to line 1 AND line 7
	builder.AddLine("  article.value = value;")    // Similar to line 2 AND line 8
	builder.AddLine("  return false;")             // Similar to line 3 AND line 9

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	t.Logf("Number of stages: %d", len(result.Stages))
	for i, stage := range result.Stages {
		t.Logf("Stage %d: BufferStart=%d, BufferEnd=%d, changes=%d",
			i, stage.BufferStart, stage.BufferEnd, len(stage.Changes))
		for lineNum, change := range stage.Changes {
			t.Logf("  Change at relative line %d: OldLineNum=%d, type=%v",
				lineNum, change.OldLineNum, change.Type)
		}
	}

	// Check that changes are coherent (all in one location, not scattered)
	if len(result.Stages) == 1 {
		stage := result.Stages[0]
		bufferRange := stage.BufferEnd - stage.BufferStart
		// If all changes are coherent, range should be small (2-3 lines)
		// If scattered, range could be large (6+ lines spanning two functions)
		if bufferRange > 5 {
			t.Errorf("Changes appear scattered: BufferStart=%d, BufferEnd=%d, range=%d. "+
				"Expected coherent changes within ~3 lines.",
				stage.BufferStart, stage.BufferEnd, bufferRange)
		}
	}
}

// TestIncrementalStageBuilder_GapDetectionUsesNewLineNumbers directly tests
// that gap detection is based on new line numbers, not buffer line numbers.
func TestIncrementalStageBuilder_GapDetectionUsesNewLineNumbers(t *testing.T) {
	// Create old lines where we can control exactly which lines match
	// Use similar prefixes to ensure modifications are detected
	oldLines := []string{
		"  func alpha() {",   // line 1 (buffer 10)
		"    return 1",       // line 2 (buffer 11)
		"  }",                // line 3 (buffer 12)
		"",                   // line 4 (buffer 13)
		"  func beta() {",    // line 5 (buffer 14)
		"    return 2",       // line 6 (buffer 15)
	}

	baseLineOffset := 10

	builder := NewIncrementalStageBuilder(
		oldLines,
		baseLineOffset,
		2,     // proximityThreshold = 2 (gap > 2 should split)
		5, 20, // viewport
		12,    // cursorRow
		"test.go",
	)

	// Output lines where:
	// - New line 1: modification of old line 1 (buffer 10)
	// - New line 2: exact match of old line 2
	// - New line 3: exact match of old line 3
	// - New line 4: exact match of old line 4
	// - New line 5: modification of old line 6 (buffer 15!) - skipping old line 5
	//
	// New line gap between changes: 5 - 1 = 4 > threshold (should split)
	// Buffer line gap between changes: 15 - 10 = 5 > threshold (should split)

	builder.AddLine("  func alpha(x) {")  // Modify old line 1 -> buffer 10
	builder.AddLine("    return 1")       // Match old line 2
	builder.AddLine("  }")                // Match old line 3
	builder.AddLine("")                   // Match old line 4
	builder.AddLine("    return 200")     // Modify - similar to old line 6 -> buffer 15

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	t.Logf("Number of stages: %d", len(result.Stages))
	for i, stage := range result.Stages {
		t.Logf("Stage %d: BufferStart=%d, BufferEnd=%d, changes=%d",
			i, stage.BufferStart, stage.BufferEnd, len(stage.Changes))
		for lineNum, change := range stage.Changes {
			t.Logf("  Change %d: OldLineNum=%d, type=%v",
				lineNum, change.OldLineNum, change.Type)
		}
	}

	// Since BOTH new-line gap (4) and buffer-line gap (5) exceed threshold (2),
	// we expect 2 stages regardless of which gap metric is used.
	if len(result.Stages) != 2 {
		t.Errorf("Expected 2 stages (gap > threshold), got %d", len(result.Stages))
	}
}

// TestIncrementalStageBuilder_BufferGapExceedsThresholdButNewLineGapDoesNot
// tests the specific case where buffer gaps are large but new-line gaps are small.
// This is the bug scenario.
func TestIncrementalStageBuilder_BufferGapExceedsThresholdButNewLineGapDoesNot(t *testing.T) {
	// Old lines with distinct content. We'll output SIMILAR lines to ensure
	// they match as modifications (not additions).
	oldLines := []string{
		"  article.title = title;",     // line 1 (buffer 50) - will modify
		"  const x = 1;",               // line 2 (buffer 51) - will skip
		"  article.author = author;",   // line 3 (buffer 52) - will modify
		"  const y = 2;",               // line 4 (buffer 53) - will skip
		"  article.tags = tags;",       // line 5 (buffer 54) - will modify
	}

	baseLineOffset := 50

	builder := NewIncrementalStageBuilder(
		oldLines,
		baseLineOffset,
		1,      // proximityThreshold = 1 (very strict - gap > 1 should split)
		45, 60, // viewport
		52,     // cursorRow
		"test.go",
	)

	// Model outputs CONSECUTIVE new lines (no gaps in new-line numbers)
	// These are similar enough to match the old lines at positions 1, 3, 5:
	// - New line 1 -> old line 1 (buffer 50) - similar "article.title"
	// - New line 2 -> old line 3 (buffer 52) - similar "article.author" (buffer gap = 2!)
	// - New line 3 -> old line 5 (buffer 54) - similar "article.tags" (buffer gap = 2!)
	//
	// New line gaps: 1, 1 (NOT > threshold)
	// Buffer line gaps: 2, 2 (> threshold!)
	//
	// BUG: With current code, NO stage splitting because new-line gaps are small.
	// FIX: Should split based on buffer-line gaps.

	finalized1 := builder.AddLine("  article.title = newTitle;")   // Modify -> buffer 50
	finalized2 := builder.AddLine("  article.author = newAuthor;") // Modify -> buffer 52
	finalized3 := builder.AddLine("  article.tags = newTags;")     // Modify -> buffer 54

	// Track what was finalized during streaming
	streamFinalizedCount := 0
	if finalized1 != nil {
		streamFinalizedCount++
		t.Log("Stage finalized after line 1")
	}
	if finalized2 != nil {
		streamFinalizedCount++
		t.Log("Stage finalized after line 2")
	}
	if finalized3 != nil {
		streamFinalizedCount++
		t.Log("Stage finalized after line 3")
	}

	result := builder.Finalize()
	if result == nil {
		t.Fatal("expected staging result")
	}

	t.Logf("Stages finalized during streaming: %d", streamFinalizedCount)
	t.Logf("Total stages after Finalize: %d", len(result.Stages))
	for i, stage := range result.Stages {
		t.Logf("Stage %d: BufferStart=%d, BufferEnd=%d, changes=%d",
			i, stage.BufferStart, stage.BufferEnd, len(stage.Changes))
		for lineNum, change := range stage.Changes {
			t.Logf("  Change %d: OldLineNum=%d, type=%v, content=%q",
				lineNum, change.OldLineNum, change.Type, change.Content)
		}
	}

	// Verify the changes mapped to the expected old lines
	if len(result.Stages) >= 1 {
		// Check that we have changes at buffer lines 50, 52, 54
		allChanges := make(map[int]bool)
		for _, stage := range result.Stages {
			for _, change := range stage.Changes {
				bufLine := change.OldLineNum + baseLineOffset - 1
				allChanges[bufLine] = true
				t.Logf("  -> Mapped to buffer line: %d", bufLine)
			}
		}
	}

	// With buffer-gap-based splitting (the fix), we'd expect 3 stages.
	// With new-line-gap-based splitting (the bug), we get 1 stage.
	if len(result.Stages) == 1 {
		stage := result.Stages[0]
		t.Errorf("BUG CONFIRMED: All changes grouped into 1 stage (BufferStart=%d, BufferEnd=%d). "+
			"Gap detection is using NEW line numbers (consecutive: 1,2,3) "+
			"instead of BUFFER line numbers (scattered: 50,52,54 with gaps of 2).",
			stage.BufferStart, stage.BufferEnd)
	} else if len(result.Stages) < 3 {
		t.Errorf("Expected 3 stages (buffer gaps > threshold=1), got %d", len(result.Stages))
	}
}

// TestIncrementalDiffBuilder_FirstLineMatchingBug reproduces the bug from logs where
// the first model line matches to a far-away old line (outside the search window).
// This should be impossible given the search window of [0, expectedPos+10].
func TestIncrementalDiffBuilder_FirstLineMatchingBug(t *testing.T) {
	// Simulate the scenario from the logs:
	// - 98 old lines (full file)
	// - Model outputs something, and first line matches to old line 41
	//
	// Old line 0: "import { Hono } from \"hono\";"
	// Old line 40: "export { WorkflowRuntimeEntrypoint... }"
	// Old line 41: empty or comment
	//
	// If model line 1 is "import apiKeyRoutes...", it should match within [0, 10),
	// NOT to line 41.

	oldLines := make([]string, 98)
	// Fill with realistic content
	oldLines[0] = "import { Hono } from \"hono\";"
	oldLines[1] = ""
	oldLines[2] = "import auth from \"./auth\";"
	oldLines[3] = "import { ApiContext } from \"./context\";"
	for i := 4; i < 40; i++ {
		oldLines[i] = "import something from \"./something\";"
	}
	oldLines[40] = "// Export comment"
	oldLines[41] = "export { WorkflowRuntimeEntrypoint as Runtime } from \"./runtime\";"
	oldLines[42] = ""
	oldLines[43] = "// Initialize app"
	oldLines[44] = "const application = new Hono<ApiContext>();"
	for i := 45; i < 98; i++ {
		oldLines[i] = "app.route(\"/path\", handler);"
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// Model outputs "import apiKeyRoutes..." as first line
	// This should match within [0, 10), NOT to line 41
	modelLine1 := "import apiKeyRoutes from \"./routes/api-keys\";"
	change1 := builder.AddLine(modelLine1)

	// Check where it matched
	if len(builder.LineMapping.NewToOld) == 0 {
		t.Fatal("expected line mapping to be populated")
	}

	matchedOldLine := builder.LineMapping.NewToOld[0] // 1-indexed old line number

	t.Logf("Model line 1: %q", modelLine1)
	t.Logf("Matched to old line: %d", matchedOldLine)
	if matchedOldLine > 0 && matchedOldLine <= len(oldLines) {
		t.Logf("Old line content: %q", oldLines[matchedOldLine-1])
	}

	// The search window for first line is [0, 10) (expectedPos=0, so max(0,-2) to min(98,10))
	// Any match outside this range indicates a bug
	if matchedOldLine > 10 {
		t.Errorf("BUG: First model line matched to old line %d, but search window is [0, 10). "+
			"The matching algorithm should not be able to find matches outside the search window.",
			matchedOldLine)
	}

	// If no match was found (matchedOldLine == 0), check what type of change was recorded
	if change1 != nil {
		t.Logf("Change recorded: Type=%v, OldLineNum=%d", change1.Type, change1.OldLineNum)
		if change1.OldLineNum > 10 {
			t.Errorf("BUG: Change OldLineNum=%d is outside search window [0, 10)", change1.OldLineNum)
		}
	}
}

// TestIncrementalDiffBuilder_SearchWindowRespected verifies the search window bounds
func TestIncrementalDiffBuilder_SearchWindowRespected(t *testing.T) {
	// Create old lines where exact match exists ONLY outside the search window
	oldLines := make([]string, 50)
	for i := 0; i < 50; i++ {
		oldLines[i] = "generic line"
	}
	// Put a unique line at position 30 (outside initial search window [0, 10))
	oldLines[30] = "unique content at line 31"

	builder := NewIncrementalDiffBuilder(oldLines)

	// Try to match the unique line as FIRST model line
	// It should NOT match because it's outside [0, 10)
	change := builder.AddLine("unique content at line 31")

	matchedOldLine := builder.LineMapping.NewToOld[0]
	t.Logf("Matched to old line: %d", matchedOldLine)
	if change != nil {
		t.Logf("Change type: %v, OldLineNum: %d", change.Type, change.OldLineNum)
	}

	// Should either:
	// 1. Match to something in [0, 10) via similarity
	// 2. Or be recorded as addition (matchedOldLine == 0)
	// Should NOT match to line 31
	if matchedOldLine == 31 {
		t.Error("BUG: Matched to line 31 which is outside search window [0, 10)")
	}
}

// TestIncrementalDiffBuilder_OldLineIdxProgression verifies oldLineIdx advances correctly
func TestIncrementalDiffBuilder_OldLineIdxProgression(t *testing.T) {
	// Use very distinct lines to avoid similarity matching
	oldLines := []string{
		"function alpha() {", // 1
		"function beta() {",  // 2
		"function gamma() {", // 3
		"function delta() {", // 4
		"function epsilon() {", // 5
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// Match line 1
	builder.AddLine("function alpha() {")
	if builder.oldLineIdx != 1 {
		t.Errorf("After matching line 1, oldLineIdx should be 1, got %d", builder.oldLineIdx)
	}

	// Match line 2
	builder.AddLine("function beta() {")
	if builder.oldLineIdx != 2 {
		t.Errorf("After matching line 2, oldLineIdx should be 2, got %d", builder.oldLineIdx)
	}

	// Add something completely different (should be addition)
	change := builder.AddLine("ZZZZZ COMPLETELY DIFFERENT ZZZZZ")
	t.Logf("After adding different content: oldLineIdx=%d, change=%v", builder.oldLineIdx, change)
	if change != nil {
		t.Logf("Change type: %v, OldLineNum: %d", change.Type, change.OldLineNum)
	}
	// With 0.3 similarity threshold, this might still match something
	// The test documents actual behavior rather than expected behavior

	// Match line 3 (should still work because search window extends forward)
	builder.AddLine("function gamma() {")
	t.Logf("After matching gamma: oldLineIdx=%d", builder.oldLineIdx)
}

// TestIncrementalDiffBuilder_ModelOutputsGarbage simulates when model outputs
// completely different content from expected, causing chaotic matching
func TestIncrementalDiffBuilder_ModelOutputsGarbage(t *testing.T) {
	// Real scenario: model supposed to output file with `app` -> `application` renames
	// But instead outputs scrambled/duplicated imports

	oldLines := []string{
		"import { Hono } from \"hono\";",
		"",
		"import auth from \"./auth\";",
		"import apiKeyRoutes from \"./routes/api-keys\";",
		"import billingRoutes from \"./routes/billing\";",
		"import dashboardRoutes from \"./routes/dashboard\";",
		"import databaseRoutes from \"./routes/databases\";",
		"// Rest of imports...",
		"",
		"const app = new Hono();",
		"",
		"app.use(middleware);",
		"app.route(\"/health\", health);",
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// Model outputs imports in WRONG order (simulating garbage output)
	garbageOutput := []string{
		"import apiKeyRoutes from \"./routes/api-keys\";", // Should be line 4, appearing first
		"import apiKeyRoutes from \"./routes/api-keys\";", // DUPLICATE!
		"import billingRoutes from \"./routes/billing\";",
		"import dashboardRoutes from \"./routes/dashboard\";",
	}

	for i, line := range garbageOutput {
		change := builder.AddLine(line)
		matchedOld := builder.LineMapping.NewToOld[i]

		t.Logf("Model line %d: %q -> matched old line %d", i+1, line, matchedOld)
		if change != nil {
			t.Logf("  Change: Type=%v, OldLineNum=%d", change.Type, change.OldLineNum)
		}
	}

	// The first line "import apiKeyRoutes..." should match to old line 4 (within search window [0, 10))
	firstMatch := builder.LineMapping.NewToOld[0]
	if firstMatch != 4 {
		t.Logf("First model line matched to old line %d (expected 4)", firstMatch)
	}

	// After first match, oldLineIdx should be 4
	// So for duplicate line 2, search window is [2, 14)
	// The duplicate should either:
	// - Match via similarity to some other line
	// - Or be addition (old line 4 is already used)

	// Check that usedOldLines prevents duplicate matching
	secondMatch := builder.LineMapping.NewToOld[1]
	if secondMatch == firstMatch && firstMatch > 0 {
		t.Errorf("BUG: Duplicate model line matched to same old line %d. "+
			"usedOldLines should prevent this.", firstMatch)
	}
}

// TestIncrementalDiffBuilder_VerifySearchWindowBounds directly tests the search bounds
func TestIncrementalDiffBuilder_VerifySearchWindowBounds(t *testing.T) {
	// Test that search window is correctly [max(0, expectedPos-2), min(len, expectedPos+10))

	cases := []struct {
		name          string
		oldLineCount  int
		expectPosHint int // Not directly accessible, but we can infer from oldLineIdx
		wantStart     int
		wantEnd       int
	}{
		{"first line", 100, 0, 0, 10},
		{"after matching line 5", 100, 5, 3, 15},
		{"near end", 20, 18, 16, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldLines := make([]string, tc.oldLineCount)
			for i := 0; i < tc.oldLineCount; i++ {
				oldLines[i] = "generic"
			}
			// Put unique content at specific positions
			oldLines[tc.wantStart] = "unique_start"
			if tc.wantEnd-1 < tc.oldLineCount {
				oldLines[tc.wantEnd-1] = "unique_end"
			}

			builder := NewIncrementalDiffBuilder(oldLines)

			// Manually set oldLineIdx to simulate state
			builder.oldLineIdx = tc.expectPosHint

			// Try to match content that ONLY exists at wantStart
			builder.AddLine("unique_start")

			// Should find it
			matched := builder.LineMapping.NewToOld[0]
			expectedLine := tc.wantStart + 1 // 1-indexed

			t.Logf("Expected match at old line %d, got %d", expectedLine, matched)

			// Try to match content that ONLY exists at wantEnd-1 (boundary of search)
			if tc.wantEnd-1 > tc.wantStart && tc.wantEnd-1 < tc.oldLineCount {
				builder2 := NewIncrementalDiffBuilder(oldLines)
				builder2.oldLineIdx = tc.expectPosHint
				builder2.AddLine("unique_end")
				matched2 := builder2.LineMapping.NewToOld[0]
				t.Logf("Boundary match: expected %d, got %d", tc.wantEnd, matched2)
			}
		})
	}
}

// TestIncrementalDiffBuilder_ReproduceLogScenario reproduces the exact scenario from logs
// where the first change appears at buffer line 42
func TestIncrementalDiffBuilder_ReproduceLogScenario(t *testing.T) {
	// From the log:
	// - oldLines has 98 lines (the "current" file)
	// - First change at buffer_line=42
	// - This means lines 1-41 matched exactly (no changes)
	// - Line 42 is the first change

	// Create oldLines similar to the real file structure
	oldLines := make([]string, 98)
	for i := 0; i < 39; i++ {
		oldLines[i] = "import something from \"./something\";" // Lines 1-39: imports
	}
	oldLines[39] = "// Comment line 40"
	oldLines[40] = "export { Runtime } from \"./runtime\";" // Line 41: export
	oldLines[41] = ""                                        // Line 42: empty
	oldLines[42] = "// Initialize app"                       // Line 43
	oldLines[43] = "const application = new Hono();"         // Line 44: the changed line
	for i := 44; i < 98; i++ {
		oldLines[i] = "app.route(\"/path\", handler);"
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// Model outputs EXACT SAME content for lines 1-41
	for i := 0; i < 41; i++ {
		change := builder.AddLine(oldLines[i])
		if change != nil {
			t.Errorf("Line %d should be exact match, got change type %v", i+1, change.Type)
		}
	}

	t.Logf("After 41 lines: oldLineIdx=%d", builder.oldLineIdx)

	// Now model outputs something DIFFERENT for line 42
	// This should be the first change
	change42 := builder.AddLine("import apiKeyRoutes from \"./routes/api-keys\";")
	if change42 == nil {
		t.Fatal("Expected change at line 42")
	}

	t.Logf("Line 42 change: Type=%v, OldLineNum=%d", change42.Type, change42.OldLineNum)
	t.Logf("Matched old line content: %q", oldLines[change42.OldLineNum-1])

	// The change should map to an old line in the search window
	// After processing 41 lines, oldLineIdx=41, search window is [39, 51)
	// So OldLineNum should be in [40, 52) (1-indexed)
	if change42.OldLineNum < 40 || change42.OldLineNum > 51 {
		t.Errorf("OldLineNum=%d is outside expected search window [40, 51)", change42.OldLineNum)
	}

	// Buffer line calculation: OldLineNum + baseLineOffset - 1
	// With baseLineOffset=1: buffer_line = OldLineNum
	baseLineOffset := 1
	bufferLine := change42.OldLineNum + baseLineOffset - 1
	t.Logf("Buffer line for first change: %d", bufferLine)

	// This test documents that if model outputs different content at line 42,
	// the change's buffer position depends on similarity matching within the search window
}

// TestIncrementalDiffBuilder_MatchingWhenModelSkipsLines tests what happens when
// model output is out of order or skips lines
func TestIncrementalDiffBuilder_MatchingWhenModelSkipsLines(t *testing.T) {
	// Old lines: a predictable sequence
	oldLines := []string{
		"line one",   // 1
		"line two",   // 2
		"line three", // 3
		"line four",  // 4
		"line five",  // 5
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// Model outputs in WRONG order: line four first, then line one
	change1 := builder.AddLine("line four") // Should match old line 4 (in window [0, 10))
	if change1 != nil {
		t.Logf("Change1: Type=%v, OldLineNum=%d", change1.Type, change1.OldLineNum)
	}

	matched1 := builder.LineMapping.NewToOld[0]
	t.Logf("Model line 1 'line four' matched to old line %d", matched1)

	if matched1 != 4 {
		t.Errorf("'line four' should match old line 4, got %d", matched1)
	}

	// Now model outputs "line one" - but old line 4 is already used and oldLineIdx=4
	// Search window is [2, 14), and "line one" is at old line 1 (outside!)
	change2 := builder.AddLine("line one")
	if change2 != nil {
		t.Logf("Change2: Type=%v, OldLineNum=%d", change2.Type, change2.OldLineNum)
	}

	matched2 := builder.LineMapping.NewToOld[1]
	t.Logf("Model line 2 'line one' matched to old line %d", matched2)

	// "line one" at old line 1 is OUTSIDE search window [2, 14) after first match
	// So it either:
	// 1. Matches via similarity to something else in [2, 14)
	// 2. Is treated as addition (matched2 == 0)
	if matched2 == 1 {
		t.Error("BUG: 'line one' matched to old line 1 but that's outside search window [2, 14)")
	}
}

// TestIncrementalStageBuilder_WhenModelOutputStartsMidFile simulates when model
// starts outputting from middle of file instead of beginning
func TestIncrementalStageBuilder_WhenModelOutputStartsMidFile(t *testing.T) {
	// Old lines represent the full file
	oldLines := []string{
		"import { Hono } from \"hono\";",
		"import auth from \"./auth\";",
		"import { Context } from \"./context\";",
		"",
		"// Initialize app",
		"const app = new Hono();",
		"",
		"app.use(middleware);",
		"app.route(\"/health\", health);",
		"app.route(\"/auth\", auth);",
	}

	builder := NewIncrementalStageBuilder(
		oldLines,
		1,      // baseLineOffset
		3,      // proximityThreshold
		0, 0,   // viewport disabled
		5,      // cursorRow
		"test.ts",
	)

	// Model INCORRECTLY starts outputting from line 5 (skipping lines 1-4)
	// This simulates a model that hallucinates and starts mid-file
	modelOutput := []string{
		"// Initialize app",           // Should be line 5, but model outputs as line 1
		"const application = new Hono();", // Modified line 6
		"",
		"application.use(middleware);", // Modified line 8
		"application.route(\"/health\", health);", // Modified line 9
	}

	for i, line := range modelOutput {
		finalized := builder.AddLine(line)
		if finalized != nil {
			t.Logf("Stage finalized after model line %d", i+1)
		}
	}

	result := builder.Finalize()
	if result == nil {
		t.Fatal("Expected staging result")
	}

	t.Logf("Stages: %d", len(result.Stages))
	for i, stage := range result.Stages {
		t.Logf("Stage %d: BufferStart=%d, BufferEnd=%d, changes=%d",
			i, stage.BufferStart, stage.BufferEnd, len(stage.Changes))

		// Check that buffer positions make sense
		// Even though model output starts at line 1, changes should map to
		// buffer positions based on where they match in old lines
		if stage.BufferStart > len(oldLines)+1 {
			t.Errorf("Stage %d BufferStart=%d exceeds file length", i, stage.BufferStart)
		}
	}
}

// TestLineSimilarity_RealLogExample checks similarity for the exact lines from the bug log
func TestLineSimilarity_RealLogExample(t *testing.T) {
	// From the log:
	// old_lines: "export { WorkflowRuntimeEntrypoint as Runtime } from \"./runtime/workflow-runtime-entrypoint\";"
	// lines: "import apiKeyRoutes from \"./routes/api-keys\";"

	oldLine := `export { WorkflowRuntimeEntrypoint as Runtime } from "./runtime/workflow-runtime-entrypoint";`
	newLine := `import apiKeyRoutes from "./routes/api-keys";`

	similarity := LineSimilarity(newLine, oldLine)
	t.Logf("Similarity between:")
	t.Logf("  New: %q (%d chars)", newLine, len(newLine))
	t.Logf("  Old: %q (%d chars)", oldLine, len(oldLine))
	t.Logf("  Similarity: %.3f (threshold: 0.3)", similarity)

	// With threshold 0.3, check if this would match
	if similarity > 0.3 {
		t.Logf("WOULD MATCH: similarity %.3f > 0.3 threshold", similarity)
	} else {
		t.Logf("WOULD NOT MATCH: similarity %.3f <= 0.3 threshold", similarity)
	}

	// Check similarity with other lines that might be in the search window
	otherLines := []string{
		"",                                            // empty line
		"// Export comment",                           // comment
		"// Initialize Hono app with types",           // another comment
		"const application = new Hono<ApiContext>();", // the actual change target
		`import { Hono } from "hono";`,                // first import
		`import auth from "./auth";`,                  // another import
	}

	t.Log("\nSimilarity with other potential matches in search window:")
	for i, line := range otherLines {
		sim := LineSimilarity(newLine, line)
		matchStr := ""
		if sim > 0.3 {
			matchStr = " <- WOULD MATCH"
		}
		t.Logf("  [%d] %.3f: %q%s", i, sim, line, matchStr)
	}
}

// TestIncrementalDiffBuilder_ExactLogScenario reproduces the exact scenario from logs
// with real file content structure
func TestIncrementalDiffBuilder_ExactLogScenario(t *testing.T) {
	// Reconstruct the exact old lines from the log (current/ section)
	// The file has 98 lines, structured as:
	// - Lines 1-40: imports
	// - Line 41: export statement
	// - Line 42: empty
	// - Line 43: comment
	// - Line 44: const application = new Hono...
	// - Lines 45+: app.use() and app.route() calls

	oldLines := make([]string, 98)
	oldLines[0] = `import { Hono } from "hono";`
	oldLines[1] = ""
	oldLines[2] = `import auth from "./auth";`
	oldLines[3] = `import { ApiContext } from "./context";`
	// Fill imports (lines 5-40)
	for i := 4; i < 40; i++ {
		oldLines[i] = `import something from "./something";`
	}
	oldLines[40] = `export { WorkflowRuntimeEntrypoint as Runtime } from "./runtime/workflow-runtime-entrypoint";`
	oldLines[41] = ""
	oldLines[42] = `// Initialize Hono app with types`
	oldLines[43] = `const application = new Hono<ApiContext>();`
	oldLines[44] = ""
	oldLines[45] = `// Global middleware`
	oldLines[46] = `app.use("*", corsMiddleware);`
	for i := 47; i < 98; i++ {
		oldLines[i] = `app.route("/path", handler);`
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// What would happen if model outputs "import apiKeyRoutes..." as FIRST line?
	// Expected: should match via similarity to one of the imports in [0, 10)
	modelFirstLine := `import apiKeyRoutes from "./routes/api-keys";`
	change := builder.AddLine(modelFirstLine)

	matched := builder.LineMapping.NewToOld[0]
	t.Logf("Model line 1: %q", modelFirstLine)
	t.Logf("Matched to old line: %d", matched)
	if matched > 0 && matched <= len(oldLines) {
		t.Logf("Old line content: %q", oldLines[matched-1])
	}
	if change != nil {
		t.Logf("Change type: %v", change.Type)
	}

	// Check: can it match to line 41 (the export)?
	// Only if search window includes line 41, which would require oldLineIdx >= 31
	// For FIRST line, oldLineIdx=0, search window=[0, 10), so line 41 is UNREACHABLE

	if matched == 41 {
		t.Error("BUG: First model line matched to old line 41, but search window is [0, 10)")
	}

	// The key insight: if first model line matches within [0, 10),
	// how can the log show buffer_line=42?
	// Answer: the LOG must be showing data from a DIFFERENT point in processing
	// OR: the oldLines used in the actual run were DIFFERENT from what we expect

	// Let's simulate what would happen if lines 1-40 output exactly matches old lines 1-40
	builder2 := NewIncrementalDiffBuilder(oldLines)
	for i := 0; i < 40; i++ {
		builder2.AddLine(oldLines[i])
	}
	t.Logf("After 40 exact matches, oldLineIdx=%d", builder2.oldLineIdx)

	// Now output "import apiKeyRoutes..." as line 41
	// Search window is [38, 50), which INCLUDES old line 41 (the export)!
	change41 := builder2.AddLine(modelFirstLine)
	matched41 := builder2.LineMapping.NewToOld[40]
	t.Logf("Model line 41: %q", modelFirstLine)
	t.Logf("Matched to old line: %d", matched41)
	if matched41 > 0 && matched41 <= len(oldLines) {
		t.Logf("Old line content: %q", oldLines[matched41-1])
		similarity := LineSimilarity(modelFirstLine, oldLines[matched41-1])
		t.Logf("Similarity: %.3f", similarity)
	}
	if change41 != nil {
		t.Logf("Change type: %v, OldLineNum=%d", change41.Type, change41.OldLineNum)
	}
}

// TestIncrementalDiffBuilder_SimulateBadModelOutput simulates when model outputs
// completely wrong/scrambled content after initial matches
func TestIncrementalDiffBuilder_SimulateBadModelOutput(t *testing.T) {
	// Create realistic old lines
	oldLines := make([]string, 98)
	oldLines[0] = `import { Hono } from "hono";`
	oldLines[1] = ""
	oldLines[2] = `import auth from "./auth";`
	oldLines[3] = `import { ApiContext } from "./context";`
	for i := 4; i < 40; i++ {
		oldLines[i] = `import route` + string(rune('A'+i)) + ` from "./routes";`
	}
	oldLines[40] = `export { Runtime } from "./runtime";`
	oldLines[41] = ""
	oldLines[42] = `// Initialize`
	oldLines[43] = `const app = new Hono();`
	for i := 44; i < 98; i++ {
		oldLines[i] = `app.route("/path", handler);`
	}

	builder := NewIncrementalDiffBuilder(oldLines)

	// Model outputs correctly for first 40 lines
	for i := 0; i < 40; i++ {
		change := builder.AddLine(oldLines[i])
		if change != nil {
			t.Errorf("Line %d should match exactly, got change", i+1)
		}
	}

	t.Logf("After 40 matches: oldLineIdx=%d", builder.oldLineIdx)

	// Now model starts outputting GARBAGE - repeated imports in wrong order
	garbageOutput := []string{
		`import apiKeyRoutes from "./routes/api-keys";`,
		`import apiKeyRoutes from "./routes/api-keys";`, // duplicate!
		`import billingRoutes from "./routes/billing";`,
		`import billingRoutes from "./routes/billing";`, // duplicate!
		`import dashboardRoutes from "./routes/dashboard";`,
	}

	t.Log("\nProcessing garbage output after 40 good lines:")
	for i, line := range garbageOutput {
		change := builder.AddLine(line)
		matched := builder.LineMapping.NewToOld[40+i]
		t.Logf("Garbage line %d: matched=%d, change=%v", i+1, matched, change != nil)
		if change != nil {
			t.Logf("  Type=%v, OldLineNum=%d", change.Type, change.OldLineNum)
		}
	}

	// Check how changes are distributed
	t.Logf("\nTotal changes recorded: %d", len(builder.Changes))

	// Key insight: if model outputs garbage after line 40, the changes will be:
	// - Modifications (if similarity > 0.3 to something in search window)
	// - Additions (if no match found)
	// The scrambled/duplicated output pattern in the log suggests model failure
}

// TestIncrementalStageBuilder_FullLogScenarioSimulation fully simulates the log scenario
func TestIncrementalStageBuilder_FullLogScenarioSimulation(t *testing.T) {
	// Build oldLines matching the "current/" file from the log
	oldLines := make([]string, 98)
	oldLines[0] = `import { Hono } from "hono";`
	oldLines[1] = ""
	oldLines[2] = `import auth from "./auth";`
	for i := 3; i < 40; i++ {
		oldLines[i] = `import route` + string(rune('A'+i-3)) + ` from "./routes";`
	}
	oldLines[40] = `export { WorkflowRuntimeEntrypoint as Runtime } from "./runtime/workflow-runtime-entrypoint";`
	oldLines[41] = ""
	oldLines[42] = `// Initialize Hono app with types`
	oldLines[43] = `const application = new Hono<ApiContext>();`
	oldLines[44] = ""
	oldLines[45] = `// Global middleware`
	oldLines[46] = `app.use("*", corsMiddleware);`
	for i := 47; i < 98; i++ {
		oldLines[i] = `app.route("/path` + string(rune('0'+i%10)) + `", handler);`
	}

	builder := NewIncrementalStageBuilder(
		oldLines,
		1, // baseLineOffset - window starts at buffer line 1
		3, // proximityThreshold
		0, 0, // viewport disabled
		44, // cursorRow
		"test.ts",
	)

	// Model outputs 40 exact matches, then garbage
	for i := 0; i < 40; i++ {
		builder.AddLine(oldLines[i])
	}

	// Then model outputs scrambled imports (simulating bad model behavior)
	scrambledOutput := []string{
		`import apiKeyRoutes from "./routes/api-keys";`,
		`import apiKeyRoutes from "./routes/api-keys";`,
		`import billingRoutes from "./routes/billing";`,
		`import billingRoutes from "./routes/billing";`,
		`import dashboardRoutes from "./routes/dashboard";`,
		// ... more scrambled content
	}

	for _, line := range scrambledOutput {
		builder.AddLine(line)
	}

	result := builder.Finalize()
	if result == nil {
		t.Fatal("Expected staging result")
	}

	t.Logf("\nStaging result:")
	t.Logf("  Total stages: %d", len(result.Stages))
	t.Logf("  FirstNeedsNavigation: %v", result.FirstNeedsNavigation)

	for i, stage := range result.Stages {
		t.Logf("\nStage %d:", i)
		t.Logf("  BufferStart: %d", stage.BufferStart)
		t.Logf("  BufferEnd: %d", stage.BufferEnd)
		t.Logf("  Changes: %d", len(stage.Changes))
		t.Logf("  Groups: %d", len(stage.Groups))

		// The key observation: with bad model output, we get:
		// - First change at buffer line ~41 (where export is)
		// - Lots of additions/modifications scattered around
	}

	// If first stage BufferStart is around 41, it matches the log (buffer_line: 42)
	if len(result.Stages) > 0 && result.Stages[0].BufferStart >= 40 && result.Stages[0].BufferStart <= 43 {
		t.Log("\nThis matches the log scenario!")
		t.Log("Root cause: similarity threshold (0.3) is too low,")
		t.Log("causing 'import...' to match 'export...' with similarity 0.312")
	}
}

// TestSimilarityThreshold_TooLow demonstrates the bug where similarity threshold
// is too low, causing incorrect matches between unrelated lines
func TestSimilarityThreshold_TooLow(t *testing.T) {
	// The problematic pair from the logs
	importLine := `import apiKeyRoutes from "./routes/api-keys";`
	exportLine := `export { WorkflowRuntimeEntrypoint as Runtime } from "./runtime/workflow-runtime-entrypoint";`

	similarity := LineSimilarity(importLine, exportLine)
	t.Logf("Similarity: %.3f", similarity)

	// Current threshold is 0.3
	currentThreshold := 0.3
	if similarity > currentThreshold {
		t.Logf("BUG: These unrelated lines WOULD MATCH with current threshold %.1f", currentThreshold)
		t.Logf("  import: %q", importLine)
		t.Logf("  export: %q", exportLine)
	}

	// Recommended fix: increase threshold to 0.4 or 0.5
	for _, threshold := range []float64{0.35, 0.4, 0.5} {
		if similarity <= threshold {
			t.Logf("FIX: With threshold >= %.2f, these lines would NOT match", threshold)
			break
		}
	}

	// Check what lines SHOULD match with higher thresholds
	goodMatches := []struct {
		line1, line2 string
	}{
		{`import { Hono } from "hono";`, `import { Hono } from "hono";`},  // exact: 1.0
		{`app.use("*", middleware);`, `application.use("*", middleware);`}, // rename: high
		{`const app = new Hono();`, `const application = new Hono();`},    // rename: high
	}

	t.Log("\nSimilarities that SHOULD match:")
	for _, pair := range goodMatches {
		sim := LineSimilarity(pair.line1, pair.line2)
		t.Logf("  %.3f: %q vs %q", sim, pair.line1, pair.line2)
	}

	// The key insight: legitimate modifications (like `app` -> `application` renames)
	// have much higher similarity than accidental matches
	appLine := `const app = new Hono();`
	applicationLine := `const application = new Hono();`
	renameSimilarity := LineSimilarity(appLine, applicationLine)

	t.Logf("\nLegitimate rename similarity: %.3f", renameSimilarity)
	t.Logf("Accidental match similarity: %.3f", similarity)

	// Recommended threshold should be between these values
	recommendedThreshold := (renameSimilarity + similarity) / 2
	t.Logf("Recommended threshold: ~%.2f (midpoint)", recommendedThreshold)

	// Document the problem (not failing the test, just logging)
	if similarity > currentThreshold {
		t.Logf("ISSUE: Current similarity threshold (0.3) causes incorrect matches. " +
			"Consider increasing to 0.4 or higher.")
	}
}

func TestIncrementalStageBuilder_ConsistencyWithComputeDiff(t *testing.T) {
	oldLines := []string{
		"func main() {",
		"    fmt.Println(\"hello\")",
		"    return",
		"}",
	}
	newLines := []string{
		"func main() {",
		"    fmt.Println(\"hello world\")",
		"    return nil",
		"}",
	}

	// Use batch ComputeDiff
	oldText := JoinLines(oldLines)
	newText := JoinLines(newLines)
	batchResult := ComputeDiff(oldText, newText)

	// Use incremental builder
	builder := NewIncrementalDiffBuilder(oldLines)
	for _, line := range newLines {
		builder.AddLine(line)
	}

	// Compare change counts
	if len(builder.Changes) != len(batchResult.Changes) {
		t.Errorf("incremental builder got %d changes, batch got %d",
			len(builder.Changes), len(batchResult.Changes))
	}

	// Both should identify modifications on lines 2 and 3
	for lineNum, change := range batchResult.Changes {
		incChange, ok := builder.Changes[lineNum]
		if !ok {
			t.Errorf("batch has change at line %d, incremental doesn't", lineNum)
			continue
		}

		// Types might differ slightly (e.g., ReplaceChars vs Modification)
		// but both should identify it as a modification-like change
		batchIsMod := change.Type != ChangeAddition && change.Type != ChangeDeletion
		incIsMod := incChange.Type != ChangeAddition && incChange.Type != ChangeDeletion
		if batchIsMod != incIsMod {
			t.Errorf("line %d: batch type=%v, incremental type=%v",
				lineNum, change.Type, incChange.Type)
		}
	}
}
