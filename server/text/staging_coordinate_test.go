package text

import (
	"cursortab/assert"
	"fmt"
	"testing"
)

// TestStageCoordinates_ModificationHasCorrectMapping verifies that modifications
// in a stage have correct coordinate mapping for rendering.
func TestStageCoordinates_ModificationHasCorrectMapping(t *testing.T) {
	// Create a diff with a modification
	diff := &DiffResult{
		Changes: map[int]LineChange{
			2: {Type: ChangeModification, NewLineNum: 2, OldLineNum: 2, Content: "new line 2", OldContent: "old line 2"},
		},
		OldLineCount: 5,
		NewLineCount: 5,
	}

	newLines := []string{"line1", "new line 2", "line3", "line4", "line5"}
	oldLines := []string{"line1", "old line 2", "line3", "line4", "line5"}

	result := CreateStages(diff, 2, 1, 50, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	stage := result.Stages[0]
	assert.True(t, len(stage.Changes) > 0, "stage should have changes")

	// Verify the change has correct NewLineNum
	for _, change := range stage.Changes {
		if change.Type == ChangeModification {
			assert.True(t, change.NewLineNum > 0, "modification should have NewLineNum > 0")
		}
	}
}

// TestStageCoordinates_AdditionMapping verifies that additions have correct coordinate mapping.
func TestStageCoordinates_AdditionMapping(t *testing.T) {
	diff := &DiffResult{
		Changes: map[int]LineChange{
			2: {Type: ChangeAddition, NewLineNum: 2, OldLineNum: 1, Content: "added line"},
		},
		OldLineCount: 3,
		NewLineCount: 4,
		LineMapping: &LineMapping{
			NewToOld: []int{1, -1, 2, 3},
			OldToNew: []int{1, 3, 4},
		},
	}

	newLines := []string{"line1", "added line", "line2", "line3"}
	oldLines := []string{"line1", "line2", "line3"}

	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")
}

// TestSingleLineToMultipleLinesAllIncluded verifies that when one old line becomes
// multiple new lines, all the new lines are included in the stage.
func TestSingleLineToMultipleLinesAllIncluded(t *testing.T) {
	// Scenario: one whitespace line becomes three content lines
	oldText := `        {

        }`
	newText := `        {
            "timestamp": "2022-01-04T01:00:00Z",
            "value": 260,
            "name": "John"
        }`

	diffResult := ComputeDiff(oldText, newText)

	// We should have changes for new lines 2, 3, 4 (the three content lines)
	assert.True(t, len(diffResult.Changes) >= 2,
		fmt.Sprintf("Expected at least 2 changes, got %d", len(diffResult.Changes)))

	// Verify that additions from a delete+insert block stay together for staging
	var additionOldLineNums []int
	for _, change := range diffResult.Changes {
		if change.Type == ChangeAddition {
			additionOldLineNums = append(additionOldLineNums, change.OldLineNum)
		}
	}

	// All additions should have the same anchor (OldLineNum)
	for i := 1; i < len(additionOldLineNums); i++ {
		assert.Equal(t, additionOldLineNums[0], additionOldLineNums[i],
			"All additions in a delete+insert block should have the same OldLineNum anchor")
	}
}

// TestStageIncludesAllLinesFromDeleteInsertBlock verifies that when creating stages
// from a diff where one line becomes multiple lines, all the new lines are included.
func TestStageIncludesAllLinesFromDeleteInsertBlock(t *testing.T) {
	// Scenario: one whitespace line becomes three content lines
	oldText := "            " // just whitespace
	newText := `            "timestamp": "2022-01-04T01:00:00Z",
            "value": 260,
            "name": "John"`

	diffResult := ComputeDiff(oldText, newText)

	// Create stages from this diff
	newLines := splitLines(newText)
	oldLines := splitLines(oldText)
	stagingResult := CreateStages(
		diffResult,
		1,    // cursorRow
		0, 0, // no viewport (all visible)
		1, // baseLineOffset
		3, // proximityThreshold
		"test.json",
		newLines,
		oldLines,
	)

	// All changes should be in one stage since they're from the same delete+insert block
	if stagingResult != nil && len(stagingResult.Stages) > 0 {
		stage := stagingResult.Stages[0]

		// The stage should have all 3 lines
		assert.Equal(t, 3, len(stage.Lines),
			fmt.Sprintf("Stage should have 3 lines, got %d", len(stage.Lines)))

		// The groups should cover the changed lines
		totalLinesInGroups := 0
		for _, g := range stage.Groups {
			totalLinesInGroups += g.EndLine - g.StartLine + 1
		}
		assert.True(t, totalLinesInGroups >= 2,
			fmt.Sprintf("Groups should cover at least 2 lines, got %d", totalLinesInGroups))
	}
}

// TestMixedChangesCoordinates verifies that a mix of modifications and additions
// have correct coordinate mappings.
func TestMixedChangesCoordinates(t *testing.T) {
	// Simulate: 5 old lines replaced by 8 new lines
	diff := &DiffResult{
		Changes: map[int]LineChange{
			4: {Type: ChangeModification, NewLineNum: 4, OldLineNum: 4, Content: "MODIFIED line 4", OldContent: "line 4"},
			5: {Type: ChangeAddition, NewLineNum: 5, OldLineNum: 4, Content: "ADDED line 5"},
			6: {Type: ChangeAddition, NewLineNum: 6, OldLineNum: 4, Content: "ADDED line 6"},
			7: {Type: ChangeAddition, NewLineNum: 7, OldLineNum: 4, Content: "ADDED line 7"},
			8: {Type: ChangeAddition, NewLineNum: 8, OldLineNum: 4, Content: "ADDED line 8"},
		},
		OldLineCount: 5,
		NewLineCount: 8,
	}

	newLines := []string{
		"unchanged1",
		"unchanged2",
		"unchanged3",
		"MODIFIED line 4",
		"ADDED line 5",
		"ADDED line 6",
		"ADDED line 7",
		"ADDED line 8",
	}
	oldLines := []string{
		"unchanged1",
		"unchanged2",
		"unchanged3",
		"line 4",
		"line 5",
	}

	result := CreateStages(diff, 4, 1, 50, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	// Find the stage with changes
	stage := result.Stages[0]

	// Should have changes
	assert.True(t, len(stage.Changes) > 0, "stage should have changes")

	// Check that modification has OldLineNum
	for _, change := range stage.Changes {
		if change.Type == ChangeModification {
			assert.True(t, change.OldLineNum > 0, "modification should have OldLineNum > 0")
		}
	}
}

// TestGroupsDoNotOverlapWithModifications verifies that when staging creates groups,
// they don't overlap.
func TestGroupsDoNotOverlapWithModifications(t *testing.T) {
	diff := &DiffResult{
		Changes: map[int]LineChange{
			// Modification at new line 1
			1: {Type: ChangeModification, NewLineNum: 1, OldLineNum: 3, Content: "new1", OldContent: "old3"},
			// Character-level change at new line 2
			2: {Type: ChangeDeleteChars, NewLineNum: 2, OldLineNum: 1, Content: "new2", OldContent: "old1", ColStart: 0, ColEnd: 4},
			// Addition at new line 3
			3: {Type: ChangeAddition, NewLineNum: 3, OldLineNum: -1, Content: "added3"},
			// Addition at new line 4
			4: {Type: ChangeAddition, NewLineNum: 4, OldLineNum: -1, Content: "added4"},
		},
		OldLineCount: 5,
		NewLineCount: 8,
	}

	newLines := []string{"new1", "new2", "added3", "added4", "added5", "added6", "added7", "added8"}
	oldLines := []string{"", "", "old3", "", "old1"}

	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result")

	// Verify groups don't reference lines beyond stage content bounds
	for _, stage := range result.Stages {
		for _, g := range stage.Groups {
			assert.True(t, g.StartLine >= 1 && g.StartLine <= len(stage.Lines),
				fmt.Sprintf("Group StartLine %d should be within [1, %d]", g.StartLine, len(stage.Lines)))
			assert.True(t, g.EndLine >= 1 && g.EndLine <= len(stage.Lines),
				fmt.Sprintf("Group EndLine %d should be within [1, %d]", g.EndLine, len(stage.Lines)))
		}
	}
}

// TestBufferLineCalculation simulates how buffer positions are calculated and verifies
// that stage coordinates are correct.
func TestBufferLineCalculation(t *testing.T) {
	// Simulate a stage that covers buffer lines 28-32 (baseLineOffset=28)
	// with 8 new lines replacing 5 old lines
	diff := &DiffResult{
		Changes: map[int]LineChange{
			4: {Type: ChangeModification, NewLineNum: 4, OldLineNum: 4, Content: "MODIFIED line 4", OldContent: "old 4"},
			5: {Type: ChangeAddition, NewLineNum: 5, OldLineNum: 4, Content: "ADDED 5"},
			6: {Type: ChangeAddition, NewLineNum: 6, OldLineNum: 4, Content: "ADDED 6"},
			7: {Type: ChangeAddition, NewLineNum: 7, OldLineNum: 4, Content: "ADDED 7"},
			8: {Type: ChangeAddition, NewLineNum: 8, OldLineNum: 4, Content: "ADDED 8"},
		},
		OldLineCount: 5,
		NewLineCount: 8,
	}

	newLines := []string{
		"line1", "line2", "line3",
		"MODIFIED line 4",
		"ADDED 5", "ADDED 6", "ADDED 7", "ADDED 8",
	}
	oldLines := []string{"line1", "line2", "line3", "old 4", "line5"}

	baseLineOffset := 28
	result := CreateStages(diff, 30, 1, 100, baseLineOffset, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	// Check that BufferStart is correctly offset
	stage := result.Stages[0]
	assert.True(t, stage.BufferStart >= baseLineOffset,
		fmt.Sprintf("BufferStart (%d) should be >= baseLineOffset (%d)", stage.BufferStart, baseLineOffset))
}

// TestPureAdditionsAfterExistingContent verifies that when adding lines after
// the end of existing content, BufferStart points to the first new line, not the anchor.
func TestPureAdditionsAfterExistingContent(t *testing.T) {
	// File has 2 lines, completion adds 8 more lines
	// Lines 1-2 unchanged, lines 3-10 are pure additions
	oldLines := []string{"const x = 1;", ""}
	newLines := []string{
		"const x = 1;",
		"",
		"func helper1() {}",
		"func helper2() {}",
		"",
		"func helper3() {}",
		"func helper4() {}",
		"",
		"func helper5() {}",
		"func helper6() {}",
	}

	oldText := JoinLines(oldLines)
	newText := JoinLines(newLines)
	diff := ComputeDiff(oldText, newText)

	// Lines 3-10 should be additions
	assert.True(t, len(diff.Changes) >= 8, fmt.Sprintf("Expected at least 8 changes, got %d", len(diff.Changes)))

	// All changes should be additions
	for k, change := range diff.Changes {
		assert.Equal(t, ChangeAddition, change.Type,
			fmt.Sprintf("Change at key %d should be addition", k))
	}

	result := CreateStages(diff, 2, 0, 0, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result should not be nil")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	stage := result.Stages[0]

	// BufferStart should be 3 (first new line), not 2 (anchor line)
	assert.Equal(t, 3, stage.BufferStart,
		fmt.Sprintf("BufferStart should be 3 (insertion point), got %d", stage.BufferStart))

	// BufferEnd should be >= BufferStart
	assert.True(t, stage.BufferEnd >= stage.BufferStart,
		fmt.Sprintf("BufferEnd (%d) should be >= BufferStart (%d)", stage.BufferEnd, stage.BufferStart))
}

// TestMixedDeletionAndAdditions verifies correct staging when old content has a
// leading line that's deleted while new lines are added at the end.
func TestMixedDeletionAndAdditions(t *testing.T) {
	// Old has leading empty line, new does not (trimmed)
	oldLines := []string{"", "// Comment", "const x = 1;", ""}
	newLines := []string{"// Comment", "const x = 1;", "", "// New section", "const y = 2;", ""}

	oldText := JoinLines(oldLines)
	newText := JoinLines(newLines)
	diff := ComputeDiff(oldText, newText)

	assert.True(t, len(diff.Changes) >= 1, fmt.Sprintf("Expected at least 1 change, got %d", len(diff.Changes)))

	result := CreateStages(diff, 1, 1, 100, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result should not be nil")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	stage := result.Stages[0]

	// The stage should include ALL changed lines, not just 1
	assert.True(t, len(stage.Lines) >= 3,
		fmt.Sprintf("Stage should have at least 3 lines for meaningful changes, got %d", len(stage.Lines)))

	// BufferStart should be 1 (where the deletion is, with baseLineOffset=1)
	assert.Equal(t, 1, stage.BufferStart,
		fmt.Sprintf("BufferStart should be 1, got %d", stage.BufferStart))
}

// TestShortBufferDiffComputation tests what happens when the buffer has fewer
// lines than expected by the completion range.
func TestShortBufferDiffComputation(t *testing.T) {
	// Old: 1 line (buffer only had this much)
	oldLines := []string{"// Comment"}

	// New: 6 lines from completion
	newLines := []string{
		"// Comment",
		"const x = 1;",
		"",
		"// Section",
		"const y = 2;",
		"",
	}

	oldText := JoinLines(oldLines)
	newText := JoinLines(newLines)
	diff := ComputeDiff(oldText, newText)

	// The diff should detect additions (lines 2-6)
	assert.True(t, len(diff.Changes) >= 5,
		fmt.Sprintf("Expected at least 5 changes (additions), got %d", len(diff.Changes)))

	// Create stages
	baseLineOffset := 43
	result := CreateStages(
		diff,
		43,     // cursorRow
		1, 100, // viewport
		baseLineOffset,
		3, // proximityThreshold
		"test.ts",
		newLines,
		oldLines,
	)

	assert.NotNil(t, result, "result should not be nil")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	stage := result.Stages[0]

	// The stage should have all 5 additions
	assert.True(t, len(stage.Lines) >= 5,
		fmt.Sprintf("Stage should have at least 5 lines (additions), got %d", len(stage.Lines)))

	// BufferStart should be 44 (after the unchanged line 43, for pure additions)
	assert.Equal(t, 44, stage.BufferStart,
		fmt.Sprintf("BufferStart should be 44 (insertion point after anchor 43), got %d", stage.BufferStart))
}

// TestEmptyOldContent tests what happens when old content is empty
// (buffer has fewer lines than StartLine). All new lines become additions.
func TestEmptyOldContent(t *testing.T) {
	// Old: empty (buffer didn't have lines in this range)
	oldLines := []string{}

	// New: 6 lines from completion
	newLines := []string{
		"// Initialize Hono app with types",
		"const application = new Hono<ApiContext>();",
		"",
		"// Global middleware",
		"application.use(\"*\", corsMiddleware);",
		"",
	}

	oldText := JoinLines(oldLines)
	newText := JoinLines(newLines)
	diff := ComputeDiff(oldText, newText)

	// All 6 new lines should be additions
	assert.Equal(t, 6, len(diff.Changes), "All 6 lines should be additions")

	// Create stages
	baseLineOffset := 43
	result := CreateStages(
		diff,
		43,     // cursorRow
		1, 100, // viewport
		baseLineOffset,
		3, // proximityThreshold
		"test.ts",
		newLines,
		oldLines,
	)

	if result == nil {
		// Staging may return nil for empty old content
		return
	}

	// All additions should be in a single stage
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	// Total lines should be 6
	totalLines := 0
	for _, stage := range result.Stages {
		totalLines += len(stage.Lines)
	}
	assert.Equal(t, 6, totalLines, "Total lines should be 6")
}

// TestLeadingEmptyLineDeletion tests staging when old content has a leading
// empty line that gets deleted while additions occur.
func TestLeadingEmptyLineDeletion(t *testing.T) {
	// Original has leading blank line that gets trimmed in completion
	oldLines := []string{
		"", // leading blank line
		"// Comment",
		"const x = 1;",
		"",
	}

	// Completion with leading newline removed, plus additions
	newLines := []string{
		"// Comment",
		"const x = 1;",
		"",
		"// New section",
		"const y = 2;",
		"",
	}

	oldText := JoinLines(oldLines)
	newText := JoinLines(newLines)
	diff := ComputeDiff(oldText, newText)

	// Create stages
	result := CreateStages(
		diff,
		5,      // cursorRow
		1, 100, // viewport
		1, // baseLineOffset
		3, // proximityThreshold
		"test.go",
		newLines,
		oldLines,
	)

	if result == nil {
		assert.NotNil(t, result, "staging result should not be nil")
		return
	}
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	// The total lines across all stages should be more than 1
	totalLines := 0
	for _, stage := range result.Stages {
		totalLines += len(stage.Lines)
	}
	assert.True(t, totalLines >= 3,
		fmt.Sprintf("Total lines across stages should be at least 3, got %d", totalLines))
}

// TestStageGroupBounds verifies that stage groups don't exceed stage content bounds.
func TestStageGroupBounds(t *testing.T) {
	// Create changes at different line numbers with a gap
	diff := &DiffResult{
		Changes: map[int]LineChange{
			1: {Type: ChangeAddition, NewLineNum: 1, OldLineNum: -1, Content: "line1"},
			2: {Type: ChangeAddition, NewLineNum: 2, OldLineNum: -1, Content: "line2"},
			3: {Type: ChangeAddition, NewLineNum: 3, OldLineNum: -1, Content: "line3"},
			// Gap
			20: {Type: ChangeAddition, NewLineNum: 20, OldLineNum: -1, Content: "line20"},
			21: {Type: ChangeAddition, NewLineNum: 21, OldLineNum: -1, Content: "line21"},
		},
		OldLineCount: 3,
		NewLineCount: 21,
	}

	newLines := make([]string, 21)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}
	oldLines := []string{"old1", "old2", "old3"}

	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines, oldLines)

	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 2, "should have at least 2 stages (gap between 3 and 20)")

	// Each stage's groups should only reference lines within that stage's content
	for i, stage := range result.Stages {
		stageLineCount := len(stage.Lines)
		for _, g := range stage.Groups {
			assert.True(t, g.StartLine <= stageLineCount,
				fmt.Sprintf("Stage %d: Group StartLine (%d) exceeds stage line count (%d)",
					i, g.StartLine, stageLineCount))
			assert.True(t, g.EndLine <= stageLineCount,
				fmt.Sprintf("Stage %d: Group EndLine (%d) exceeds stage line count (%d)",
					i, g.EndLine, stageLineCount))
		}
	}
}
