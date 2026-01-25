package text

import (
	"cursortab/assert"
	"fmt"
	"strings"
	"testing"
)

// assertDiffResultEqual compares two DiffResult objects and reports any differences
func assertDiffResultEqual(t *testing.T, expected, actual *DiffResult) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}
	assert.NotNil(t, actual, "actual DiffResult")
	assert.NotNil(t, expected, "expected DiffResult")
	if expected == nil || actual == nil {
		return
	}

	for lineNum, expectedChange := range expected.Changes {
		actualChange, exists := actual.Changes[lineNum]
		assert.True(t, exists, fmt.Sprintf("change at line %d exists", lineNum))
		if !exists {
			continue
		}
		assertLineDiffEqual(t, expectedChange, actualChange)
	}

	for lineNum := range actual.Changes {
		_, exists := expected.Changes[lineNum]
		assert.True(t, exists, fmt.Sprintf("no unexpected change at line %d", lineNum))
	}

	assert.Equal(t, expected.IsOnlyLineDeletion, actual.IsOnlyLineDeletion, "IsOnlyLineDeletion")
	assert.Equal(t, expected.LastDeletion, actual.LastDeletion, "LastDeletion")
	assert.Equal(t, expected.LastAddition, actual.LastAddition, "LastAddition")
	assert.Equal(t, expected.LastLineModification, actual.LastLineModification, "LastLineModification")
	assert.Equal(t, expected.LastAppendChars, actual.LastAppendChars, "LastAppendChars")
	assert.Equal(t, expected.LastDeleteChars, actual.LastDeleteChars, "LastDeleteChars")
	assert.Equal(t, expected.LastReplaceChars, actual.LastReplaceChars, "LastReplaceChars")
	assert.Equal(t, expected.CursorLine, actual.CursorLine, "CursorLine")
	assert.Equal(t, expected.CursorCol, actual.CursorCol, "CursorCol")
}

// assertLineDiffEqual compares two LineDiff objects
func assertLineDiffEqual(t *testing.T, expected, actual LineDiff) {
	t.Helper()

	assert.Equal(t, expected.Type, actual.Type, "Type")
	assert.Equal(t, expected.LineNumber, actual.LineNumber, "LineNumber")
	assert.Equal(t, expected.Content, actual.Content, "Content")
	assert.Equal(t, expected.OldContent, actual.OldContent, "OldContent")
	assert.Equal(t, expected.ColStart, actual.ColStart, "ColStart")
	assert.Equal(t, expected.ColEnd, actual.ColEnd, "ColEnd")
}

func TestLineDeletion(t *testing.T) {
	text1 := "line 1\nline 2\nline 3\nline 4"
	text2 := "line 1\nline 3\nline 4"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineDeletion,
				LineNumber: 2,
				Content:    "line 2",
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
		},
		IsOnlyLineDeletion:   true,
		LastDeletion:         2,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           -1, // No cursor positioning for pure deletions
		CursorCol:            -1,
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineAddition(t *testing.T) {
	text1 := "line 1\nline 3\nline 4"
	text2 := "line 1\nline 2\nline 3\nline 4"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineAddition,
				LineNumber: 2,
				Content:    "line 2",
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         2,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           2, // Position at last addition
		CursorCol:            6, // End of "line 2"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineAppendChars(t *testing.T) {
	text1 := "Hello world"
	text2 := "Hello world!"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineAppendChars,
				LineNumber: 1,
				Content:    "Hello world!", // Full new line content
				OldContent: "Hello world",  // Full old line content
				ColStart:   11,
				ColEnd:     12,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           1,  // Position at last append chars
		CursorCol:            12, // End of "Hello world!"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineDeleteChars(t *testing.T) {
	text1 := "Hello world!"
	text2 := "Hello world"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineDeleteChars,
				LineNumber: 1,
				Content:    "Hello world",  // Full new line content
				OldContent: "Hello world!", // Full old line content
				ColStart:   11,
				ColEnd:     12,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      1,
		LastReplaceChars:     -1,
		CursorLine:           1,  // Position at last delete chars
		CursorCol:            11, // End of "Hello world"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineDeleteCharsMiddle(t *testing.T) {
	text1 := "Hello world John"
	text2 := "Hello John"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineDeleteChars,
				LineNumber: 1,
				Content:    "Hello John",       // Full new line content
				OldContent: "Hello world John", // Full old line content
				ColStart:   6,
				ColEnd:     12,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      1,
		LastReplaceChars:     -1,
		CursorLine:           1,  // Position at last delete chars
		CursorCol:            10, // End of "Hello John"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineReplaceChars(t *testing.T) {
	text1 := "Hello world"
	text2 := "Hello there"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineReplaceChars,
				LineNumber: 1,
				Content:    "Hello there", // Full new line content
				OldContent: "Hello world", // Full old line content
				ColStart:   6,
				ColEnd:     11,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     1,
		CursorLine:           1,  // Position at last replace chars
		CursorCol:            11, // End of "Hello there"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineReplaceCharsMiddle(t *testing.T) {
	text1 := "Hello world John"
	text2 := "Hello there John"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineReplaceChars,
				LineNumber: 1,
				Content:    "Hello there John", // Full new line content
				OldContent: "Hello world John", // Full old line content
				ColStart:   6,
				ColEnd:     11,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     1,
		CursorLine:           1,  // Position at last replace chars
		CursorCol:            16, // End of "Hello there John"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineModificationAndAddition(t *testing.T) {
	// Simple example with clear line changes
	text1 := `function hello() {
    console.log("old message");
    return true;
}`

	text2 := `function hello() {
    console.log("new message");
    return true;
    console.log("added line");
}`

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineReplaceChars, // "old" -> "new" replacement
				LineNumber: 2,
				Content:    `    console.log("new message");`, // Full new line content
				OldContent: `    console.log("old message");`, // Full old line content
				ColStart:   17,                                // Start of "new"
				ColEnd:     20,                                // End of "new"
			},
			4: {
				Type:       LineAddition,
				LineNumber: 4,
				Content:    `    console.log("added line");`,
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         4,
		LastLineModification: -1, // Only LineReplaceChars, no LineModification
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     2,
		CursorLine:           4,  // Position at last addition (since no line modification)
		CursorCol:            30, // End of "    console.log("added line");"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestMultipleDeletions(t *testing.T) {
	text1 := "line 1\nline 2\nline 3\nline 4\nline 5"
	text2 := "line 1\nline 3\nline 5"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineDeletion,
				LineNumber: 2,
				Content:    "line 2",
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
			4: {
				Type:       LineDeletion,
				LineNumber: 4,
				Content:    "line 4",
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
		},
		IsOnlyLineDeletion:   true,
		LastDeletion:         4, // Last deletion should be line 4
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           -1, // No cursor positioning for pure deletions
		CursorCol:            -1,
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestMultipleAdditions(t *testing.T) {
	text1 := "line 1\nline 3\nline 5"
	text2 := "line 1\nline 2\nline 3\nline 4\nline 5"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineAddition,
				LineNumber: 2,
				Content:    "line 2",
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
			4: {
				Type:       LineAddition,
				LineNumber: 4,
				Content:    "line 4",
				OldContent: "",
				ColStart:   0,
				ColEnd:     0,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         4, // Last addition should be line 4
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           4, // Position at last addition
		CursorCol:            6, // End of "line 4"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestMultipleCharacterChanges(t *testing.T) {
	text1 := "Hello world\nGoodbye world\nWelcome world"
	text2 := "Hello there\nGoodbye there\nWelcome there"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineReplaceChars,
				LineNumber: 1,
				Content:    "Hello there",
				OldContent: "Hello world",
				ColStart:   6,
				ColEnd:     11,
			},
			2: {
				Type:       LineReplaceChars,
				LineNumber: 2,
				Content:    "Goodbye there",
				OldContent: "Goodbye world",
				ColStart:   8,
				ColEnd:     13,
			},
			3: {
				Type:       LineReplaceChars,
				LineNumber: 3,
				Content:    "Welcome there",
				OldContent: "Welcome world",
				ColStart:   8,
				ColEnd:     13,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1, // No line modifications, only replace chars
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     3,  // Last replace should be line 3
		CursorLine:           3,  // Position at last replace chars
		CursorCol:            13, // End of "Welcome there"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestMixedCharacterOperations(t *testing.T) {
	text1 := "Hello world\nGoodbye world!\nWelcome world"
	text2 := "Hello there\nGoodbye world\nWelcome there!"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineReplaceChars,
				LineNumber: 1,
				Content:    "Hello there",
				OldContent: "Hello world",
				ColStart:   6,
				ColEnd:     11,
			},
			2: {
				Type:       LineDeleteChars,
				LineNumber: 2,
				Content:    "Goodbye world",
				OldContent: "Goodbye world!",
				ColStart:   13,
				ColEnd:     14,
			},
			3: {
				Type:       LineReplaceChars,
				LineNumber: 3,
				Content:    "Welcome there!",
				OldContent: "Welcome world",
				ColStart:   8,
				ColEnd:     14,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1, // No LineModification, only char-level operations
		LastAppendChars:      -1, // No append operations
		LastDeleteChars:      2,  // Last delete should be line 2
		LastReplaceChars:     3,  // Last replace should be line 3
		CursorLine:           3,  // Position at last replace chars
		CursorCol:            14, // End of "Welcome there!"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestLineModification(t *testing.T) {
	// Complex changes that result in multiple insertions and deletions
	// This should trigger the default case in categorizeLineChangeWithColumns
	text1 := "start middle end"
	text2 := "beginning middle finish extra"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineModification,
				LineNumber: 1,
				Content:    "beginning middle finish extra",
				OldContent: "start middle end",
				ColStart:   0,
				ColEnd:     0,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: 1, // LineModification should set this
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           1,  // Position at last line modification
		CursorCol:            29, // End of "beginning middle finish extra"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestNoChanges(t *testing.T) {
	text1 := "line 1\nline 2\nline 3"
	text2 := "line 1\nline 2\nline 3"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes:              map[int]LineDiff{},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           -1, // No cursor positioning when no changes
		CursorCol:            -1,
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestConsecutiveModifications(t *testing.T) {
	text1 := `function test() {
    start middle end
    start middle end
    start middle end
}`

	text2 := `function test() {
    beginning middle finish extra
    beginning middle finish extra
    beginning middle finish extra
}`

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineModification,
				LineNumber: 2,
				Content:    "    beginning middle finish extra",
				OldContent: "    start middle end",
			},
			3: {
				Type:       LineModification,
				LineNumber: 3,
				Content:    "    beginning middle finish extra",
				OldContent: "    start middle end",
			},
			4: {
				Type:       LineModification,
				LineNumber: 4,
				Content:    "    beginning middle finish extra",
				OldContent: "    start middle end",
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: 4, // Last modification line
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           4,  // Last modification
		CursorCol:            33, // End of "    beginning middle finish extra"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestConsecutiveAdditions(t *testing.T) {
	text1 := `function test() {
    return true;
}`

	text2 := `function test() {
    let x = 1;
    let y = 2;
    let z = 3;
    return true;
}`

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			2: {
				Type:       LineAddition,
				LineNumber: 2,
				Content:    "    let x = 1;",
			},
			3: {
				Type:       LineAddition,
				LineNumber: 3,
				Content:    "    let y = 2;",
			},
			4: {
				Type:       LineAddition,
				LineNumber: 4,
				Content:    "    let z = 3;",
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         4, // Last addition line
		LastLineModification: -1,
		LastAppendChars:      -1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           4,  // Last addition
		CursorCol:            14, // End of "    let z = 3;"
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestMixedChangesNoGrouping(t *testing.T) {
	text1 := `function test() {
    let x = 1;
    console.log("test");
    let y = 2;
}`

	text2 := `function test() {
    let x = 10;
    console.log("test");
    let y = 20;
}`

	actual := analyzeDiff(text1, text2)

	// Should NOT create groups because they're not consecutive (line 3 unchanged)
	// Lines 2 and 4 are modifications but not consecutive
	assert.True(t, len(actual.Changes) > 0, "changes detected")

	// Verify that no group types are present
	for _, change := range actual.Changes {
		assert.False(t, change.Type == LineModificationGroup || change.Type == LineAdditionGroup,
			"no grouping for non-consecutive changes")
	}
}

func TestLineChangeClassification(t *testing.T) {
	// Test the hypothesis: LineReplaceChars should only be used when there's
	// exactly 1 addition and 1 deletion at the same place

	tests := []struct {
		name     string
		oldLine  string
		newLine  string
		expected DiffType
	}{
		{
			name:     "Simple word replacement - should be replace_chars",
			oldLine:  "Hello world",
			newLine:  "Hello there",
			expected: LineReplaceChars,
		},
		{
			name:     "Multiple changes - should be modification",
			oldLine:  "start middle end",
			newLine:  "beginning middle finish extra",
			expected: LineModification,
		},
		{
			name:     "Single word change - should be replace_chars",
			oldLine:  "let x = 1;",
			newLine:  "let x = 10;",
			expected: LineReplaceChars,
		},
		{
			name:     "Complex restructuring - should be modification",
			oldLine:  "function hello() { return true; }",
			newLine:  "async function hello() { const result = await process(); return result; }",
			expected: LineModification,
		},
		{
			name:     "Append at end - should be append_chars",
			oldLine:  "Hello world",
			newLine:  "Hello world!",
			expected: LineAppendChars,
		},
		{
			name:     "App to server replacement - should be replace_chars",
			oldLine:  `app.route("/health", health);`,
			newLine:  `server.route("/health", health);`,
			expected: LineReplaceChars,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test single line change classification
			diffType, _, _ := categorizeLineChangeWithColumns(test.oldLine, test.newLine)
			assert.Equal(t, test.expected, diffType, "change classification")
		})
	}
}

// Edge case tests for cursor at top/end of file

func TestEmptyOldText(t *testing.T) {
	// Edge case: starting from empty file
	text1 := ""
	text2 := "line 1\nline 2\nline 3"

	actual := analyzeDiff(text1, text2)

	// Should detect additions for all lines
	assert.True(t, len(actual.Changes) > 0, "changes when adding content to empty file")

	// Cursor should be positioned
	assert.True(t, actual.CursorLine != -1, "cursor positioning when adding to empty file")
}

func TestEmptyNewText(t *testing.T) {
	// Edge case: deleting everything
	text1 := "line 1\nline 2\nline 3"
	text2 := ""

	actual := analyzeDiff(text1, text2)

	// Should detect deletions for all lines
	assert.True(t, len(actual.Changes) > 0, "changes when deleting all content")

	// Should be only deletions
	assert.True(t, actual.IsOnlyLineDeletion, "IsOnlyLineDeletion")

	// No cursor positioning for deletions
	assert.Equal(t, -1, actual.CursorLine, "no cursor positioning for deletions")
}

func TestSingleLineFile(t *testing.T) {
	// Edge case: single line file modification
	text1 := "hello"
	text2 := "hello world"

	actual := analyzeDiff(text1, text2)

	expected := &DiffResult{
		Changes: map[int]LineDiff{
			1: {
				Type:       LineAppendChars,
				LineNumber: 1,
				Content:    "hello world",
				OldContent: "hello",
				ColStart:   5,
				ColEnd:     11,
			},
		},
		IsOnlyLineDeletion:   false,
		LastDeletion:         -1,
		LastAddition:         -1,
		LastLineModification: -1,
		LastAppendChars:      1,
		LastDeleteChars:      -1,
		LastReplaceChars:     -1,
		CursorLine:           1,
		CursorCol:            11,
	}

	assertDiffResultEqual(t, expected, actual)
}

func TestAdditionAtFirstLine(t *testing.T) {
	// Edge case: adding content before line 1
	text1 := "line 2\nline 3"
	text2 := "line 1\nline 2\nline 3"

	actual := analyzeDiff(text1, text2)

	// Should have an addition at line 1
	change, exists := actual.Changes[1]
	assert.True(t, exists, "addition at line 1 exists")
	assert.Equal(t, LineAddition, change.Type, "addition type")

	// Cursor should be positioned at the addition
	assert.Equal(t, 1, actual.CursorLine, "cursor at line 1")
}

func TestMultipleAdditionsAtBeginning(t *testing.T) {
	// Edge case: adding multiple lines at the very beginning
	text1 := "original line"
	text2 := "new line 1\nnew line 2\nnew line 3\noriginal line"

	actual := analyzeDiff(text1, text2)

	// Should have additions (either grouped or individual)
	assert.True(t, len(actual.Changes) > 0, "changes for additions at beginning")

	// Cursor should be positioned
	assert.True(t, actual.CursorLine != -1, "cursor positioning for additions")
}

func TestModificationAtFirstLine(t *testing.T) {
	// Edge case: modifying line 1
	text1 := "old content\nline 2"
	text2 := "new content here\nline 2"

	actual := analyzeDiff(text1, text2)

	// Should have a modification at line 1
	_, exists := actual.Changes[1]
	assert.True(t, exists, "change at line 1")

	// Cursor should be positioned at line 1
	assert.Equal(t, 1, actual.CursorLine, "cursor at line 1")
}

func TestAdditionAtEndOfFile(t *testing.T) {
	// Edge case: adding lines at the end
	// Note: The diff algorithm may detect line 2 as modified due to trailing newline differences
	// when the old text doesn't have a trailing newline but the new text does
	text1 := "line 1\nline 2\n"
	text2 := "line 1\nline 2\nline 3\nline 4\n"

	actual := analyzeDiff(text1, text2)

	// Verify we have additions
	hasAddition := false
	for _, change := range actual.Changes {
		if change.Type == LineAddition || change.Type == LineAdditionGroup {
			hasAddition = true
			break
		}
	}
	assert.True(t, hasAddition, "at least one addition")

	// Cursor should be positioned at some change
	assert.True(t, actual.CursorLine != -1, "cursor positioning for additions")
}

func TestDeletionAtFirstLine(t *testing.T) {
	// Edge case: deleting line 1
	text1 := "line 1\nline 2\nline 3"
	text2 := "line 2\nline 3"

	actual := analyzeDiff(text1, text2)

	// Should have a deletion at line 1
	change, exists := actual.Changes[1]
	assert.True(t, exists, "deletion at line 1 exists")
	assert.Equal(t, LineDeletion, change.Type, "deletion type")

	// Should be only deletion
	assert.True(t, actual.IsOnlyLineDeletion, "IsOnlyLineDeletion")
}

func TestDeletionAtLastLine(t *testing.T) {
	// Edge case: deleting the last line
	text1 := "line 1\nline 2\nline 3"
	text2 := "line 1\nline 2"

	actual := analyzeDiff(text1, text2)

	// Should have a deletion at line 3
	change, exists := actual.Changes[3]
	assert.True(t, exists, "deletion at line 3 exists")
	assert.Equal(t, LineDeletion, change.Type, "deletion type")

	// LastDeletion should be 3
	assert.Equal(t, 3, actual.LastDeletion, "LastDeletion")
}

func TestCursorPositionBeyondBuffer(t *testing.T) {
	// Edge case: new text longer than old, cursor should be within bounds
	text1 := "a"
	text2 := "a\nb\nc\nd\ne"

	actual := analyzeDiff(text1, text2)

	newLines := strings.Split(text2, "\n")

	// Cursor line should be within the new text bounds
	assert.True(t, actual.CursorLine <= len(newLines), "cursor line within bounds")

	// Cursor col should be within the line bounds
	if actual.CursorLine > 0 && actual.CursorLine <= len(newLines) {
		lineContent := newLines[actual.CursorLine-1]
		assert.True(t, actual.CursorCol <= len(lineContent), "cursor col within line bounds")
	}
}

func TestIdenticalLineMarkedAsModification(t *testing.T) {
	// Bug from log: line 11 is marked as "modification" even though content == oldContent
	// This happens when adding a new line at the end of a file
	oldText := `def bubble_sort(arr):
    n = len(arr)
    for i in range(n):
        for j in range(0, n - i - 1):
            if arr[j] > arr[j + 1]:
                arr[j], arr[j + 1] = arr[j + 1], arr[j]
    return arr


if __name__ == "__main__":
    arr = [64, 34, 25, 12, 22, 11, 90]`

	newText := `def bubble_sort(arr):
    n = len(arr)
    for i in range(n):
        for j in range(0, n - i - 1):
            if arr[j] > arr[j + 1]:
                arr[j], arr[j + 1] = arr[j + 1], arr[j]
    return arr


if __name__ == "__main__":
    arr = [64, 34, 25, 12, 22, 11, 90]
    print(bubble_sort(arr))`

	actual := analyzeDiff(oldText, newText)

	// Check that line 11 is NOT in changes (it's identical in both)
	if change, exists := actual.Changes[11]; exists {
		assert.False(t, change.Content == change.OldContent,
			"line 11 should not be marked as change when content == oldContent")
	}

	// Line 12 should be an addition
	change, exists := actual.Changes[12]
	assert.True(t, exists, "line 12 exists")
	assert.Equal(t, LineAddition, change.Type, "line 12 is addition")
}

func TestIfCompletionBug(t *testing.T) {
	// Reproduce bug: typing "if " and getting completion to "if __name__ == "__main__":"
	// The preview was showing "if " as deleted instead of showing the completion
	oldText := `def bubble_sort(arr):
    n = len(arr)
    for i in range(n):
        for j in range(0, n - i - 1):
            if arr[j] > arr[j + 1]:
                arr[j], arr[j + 1] = arr[j + 1], arr[j]
    return arr

if `

	newText := `def bubble_sort(arr):
    n = len(arr)
    for i in range(n):
        for j in range(0, n - i - 1):
            if arr[j] > arr[j + 1]:
                arr[j], arr[j + 1] = arr[j + 1], arr[j]
    return arr

if __name__ == "__main__":
    arr = [64, 34, 25, 12, 22, 11, 90]
    sorted_arr = bubble_sort(arr)
    print(sorted_arr)`

	actual := analyzeDiff(oldText, newText)

	// Line 9 ("if " -> "if __name__ == "__main__":") should be append_chars, not deletion
	change9, exists := actual.Changes[9]
	assert.True(t, exists, "change at line 9 exists")
	assert.False(t, change9.Type == LineDeletion, "line 9 not categorized as deletion")
	assert.Equal(t, LineAppendChars, change9.Type, "line 9 is append_chars")
	assert.Equal(t, "if ", change9.OldContent, "oldContent")
}

func TestSingleLineToMultipleLinesWithSpacesReproduceBug(t *testing.T) {
	// This reproduces the bug where typing 'def test' in a one-line buffer
	// and getting a completion with multiple new lines only shows the first two changes
	oldText := "def test"
	newText := `def test():
    print("test")

test()



`

	actual := analyzeDiff(oldText, newText)

	// This test verifies what changes are detected by the diff algorithm
	// The user reports only seeing modification of "def test" line and addition of print line
	// but not seeing the rest (empty lines and test() call)

	// The algorithm should detect at least 2 changes: one append_chars and one addition_group
	// The addition_group should contain all the new lines including empty ones
	assert.True(t, len(actual.Changes) >= 2, "at least 2 changes detected")
}

// =============================================================================
// Tests for unequal line count scenarios (insertions/deletions)
// =============================================================================

func TestLineMapping_EqualLineCounts(t *testing.T) {
	// When line counts are equal, mapping should be 1:1 for unchanged lines
	text1 := "line 1\nline 2\nline 3"
	text2 := "line 1\nmodified\nline 3"

	actual := analyzeDiff(text1, text2)

	assert.NotNil(t, actual.LineMapping, "LineMapping")
	assert.Equal(t, 3, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 3, actual.NewLineCount, "NewLineCount")

	// Line 1 and 3 are unchanged, should map 1:1
	assert.Equal(t, 1, actual.LineMapping.NewToOld[0], "new line 1 maps to old line 1")
	assert.Equal(t, 3, actual.LineMapping.NewToOld[2], "new line 3 maps to old line 3")
	// Line 2 is modified, should still map 1:1
	assert.Equal(t, 2, actual.LineMapping.NewToOld[1], "new line 2 maps to old line 2")
}

func TestLineMapping_PureInsertion(t *testing.T) {
	// Adding lines increases new line count
	text1 := "line 1\nline 3"
	text2 := "line 1\nline 2\nline 3"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 2, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 3, actual.NewLineCount, "NewLineCount")

	// Verify line mapping
	assert.NotNil(t, actual.LineMapping, "LineMapping")

	// New line 1 -> old line 1
	assert.Equal(t, 1, actual.LineMapping.NewToOld[0], "new line 1 maps to old line 1")
	// New line 2 is inserted, should have no old mapping
	assert.Equal(t, -1, actual.LineMapping.NewToOld[1], "new line 2 (inserted) maps to -1")
	// New line 3 -> old line 2
	assert.Equal(t, 2, actual.LineMapping.NewToOld[2], "new line 3 maps to old line 2")

	// Verify the change has correct coordinates
	change, exists := actual.Changes[2]
	assert.True(t, exists, "change at line 2 exists")
	assert.Equal(t, LineAddition, change.Type, "change type")
	assert.Equal(t, 2, change.NewLineNum, "NewLineNum")
}

func TestLineMapping_PureDeletion(t *testing.T) {
	// Removing lines decreases new line count
	text1 := "line 1\nline 2\nline 3"
	text2 := "line 1\nline 3"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 3, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 2, actual.NewLineCount, "NewLineCount")

	// Verify the deleted line has correct coordinates
	change, exists := actual.Changes[2]
	assert.True(t, exists, "change at line 2 (deletion) exists")
	assert.Equal(t, LineDeletion, change.Type, "change type")
	assert.Equal(t, 2, change.OldLineNum, "OldLineNum")

	// Old line 2 should have no new mapping
	assert.Equal(t, -1, actual.LineMapping.OldToNew[1], "old line 2 (deleted) maps to -1")
}

func TestLineMapping_MultipleInsertions(t *testing.T) {
	// Adding multiple lines at once
	text1 := "start\nend"
	text2 := "start\nnew 1\nnew 2\nnew 3\nend"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 2, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 5, actual.NewLineCount, "NewLineCount")

	// Verify insertions are detected
	additionCount := 0
	for _, change := range actual.Changes {
		if change.Type == LineAddition {
			additionCount++
			// Inserted lines should have NewLineNum set but OldLineNum as anchor
			assert.True(t, change.NewLineNum > 0, "positive NewLineNum for insertion")
		}
	}
	assert.Equal(t, 3, additionCount, "addition count")
}

func TestLineMapping_MultipleDeletions(t *testing.T) {
	// Removing multiple lines at once
	text1 := "start\ndel 1\ndel 2\ndel 3\nend"
	text2 := "start\nend"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 5, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 2, actual.NewLineCount, "NewLineCount")

	// Verify deletions are detected
	deletionCount := 0
	for _, change := range actual.Changes {
		if change.Type == LineDeletion {
			deletionCount++
			assert.True(t, change.OldLineNum > 0, "positive OldLineNum for deletion")
		}
	}
	assert.Equal(t, 3, deletionCount, "deletion count")
}

func TestLineMapping_MixedInsertionDeletion(t *testing.T) {
	// Mix of insertions and deletions resulting in net line change
	text1 := "line 1\nold line 2\nline 3"
	text2 := "line 1\nnew line 2a\nnew line 2b\nline 3"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 3, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 4, actual.NewLineCount, "NewLineCount")

	// Should have changes for the modified/added lines
	assert.True(t, len(actual.Changes) > 0, "changes detected")
}

func TestLineMapping_NetLineIncrease(t *testing.T) {
	// Scenario where completion adds more lines than original
	text1 := `func hello() {
}`
	text2 := `func hello() {
    fmt.Println("Hello")
    fmt.Println("World")
}`

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 2, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 4, actual.NewLineCount, "NewLineCount")

	// Line 1 unchanged, lines 2-3 inserted, line 4 matches old line 2
	// Verify unchanged lines map correctly
	assert.Equal(t, 1, actual.LineMapping.NewToOld[0], "new line 1 maps to old line 1")
}

func TestLineMapping_NetLineDecrease(t *testing.T) {
	// Scenario where completion removes lines
	text1 := `func hello() {
    fmt.Println("Hello")
    fmt.Println("World")
    fmt.Println("!")
}`
	text2 := `func hello() {
    fmt.Println("Hello World!")
}`

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 5, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 3, actual.NewLineCount, "NewLineCount")

	// Verify we detect the changes
	assert.True(t, len(actual.Changes) > 0, "changes detected")
}

func TestLineDiffCoordinates_Modification(t *testing.T) {
	// Verify that modifications have both OldLineNum and NewLineNum set
	text1 := "Hello world"
	text2 := "Hello there"

	actual := analyzeDiff(text1, text2)

	change, exists := actual.Changes[1]
	assert.True(t, exists, "change at line 1 exists")
	assert.Equal(t, 1, change.OldLineNum, "OldLineNum")
	assert.Equal(t, 1, change.NewLineNum, "NewLineNum")
}

func TestLineDiffCoordinates_Addition(t *testing.T) {
	// Verify that additions have NewLineNum set and OldLineNum as anchor
	text1 := "line 1\nline 3"
	text2 := "line 1\nline 2\nline 3"

	actual := analyzeDiff(text1, text2)

	change, exists := actual.Changes[2]
	assert.True(t, exists, "change at line 2 exists")
	assert.Equal(t, LineAddition, change.Type, "change type")
	assert.Equal(t, 2, change.NewLineNum, "NewLineNum")
	// OldLineNum should be -1 or an anchor point (line before insertion)
	// For pure insertions, this is the anchor
}

func TestLineDiffCoordinates_Deletion(t *testing.T) {
	// Verify that deletions have OldLineNum set
	text1 := "line 1\nline 2\nline 3"
	text2 := "line 1\nline 3"

	actual := analyzeDiff(text1, text2)

	change, exists := actual.Changes[2]
	assert.True(t, exists, "change at line 2 exists")
	assert.Equal(t, LineDeletion, change.Type, "change type")
	assert.Equal(t, 2, change.OldLineNum, "OldLineNum")
}

// =============================================================================
// Edge case tests for coordinate handling
// =============================================================================

func TestDeletionAtLine1(t *testing.T) {
	// Edge case: deletion at the very first line (no preceding line for anchor)
	text1 := "first line\nsecond line\nthird line"
	text2 := "second line\nthird line"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 3, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 2, actual.NewLineCount, "NewLineCount")

	// Should have a deletion at old line 1
	change, exists := actual.Changes[1]
	assert.True(t, exists, "deletion at line 1 exists")
	assert.Equal(t, LineDeletion, change.Type, "change type")
	assert.Equal(t, 1, change.OldLineNum, "OldLineNum")
	// NewLineNum should be -1 or anchor (no line before to anchor to)
	assert.True(t, change.NewLineNum <= 0, "no valid new line anchor for first line deletion")

	// Old line 1 should map to nothing
	assert.Equal(t, -1, actual.LineMapping.OldToNew[0], "old line 1 deleted")
	// Old line 2 should map to new line 1
	assert.Equal(t, 1, actual.LineMapping.OldToNew[1], "old line 2 maps to new line 1")
}

func TestMultipleConsecutiveInsertionsThenDeletions(t *testing.T) {
	// Edge case: multiple consecutive insertions followed by deletions
	text1 := "line A\nline B\nline C\nline D\nline E"
	text2 := "line A\nnew 1\nnew 2\nline C\nline E"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 5, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 5, actual.NewLineCount, "NewLineCount")

	// Should detect changes - B deleted, new lines added, D deleted
	assert.True(t, len(actual.Changes) > 0, "changes detected")

	// Verify mapping is consistent
	assert.NotNil(t, actual.LineMapping, "LineMapping exists")
	assert.Equal(t, 5, len(actual.LineMapping.NewToOld), "NewToOld length")
	assert.Equal(t, 5, len(actual.LineMapping.OldToNew), "OldToNew length")
}

func TestInsertionAtLine1(t *testing.T) {
	// Edge case: insertion at the very first line
	text1 := "existing line"
	text2 := "new first line\nexisting line"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 1, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 2, actual.NewLineCount, "NewLineCount")

	// New line 1 is an insertion (no old line maps to it)
	assert.Equal(t, -1, actual.LineMapping.NewToOld[0], "new line 1 is insertion")
	// New line 2 maps to old line 1
	assert.Equal(t, 1, actual.LineMapping.NewToOld[1], "new line 2 maps to old line 1")
}

func TestLargeLineCountDrift(t *testing.T) {
	// Edge case: large difference in line counts
	text1 := "line 1\nline 2"
	text2 := "line 1\nnew a\nnew b\nnew c\nnew d\nnew e\nline 2"

	actual := analyzeDiff(text1, text2)

	assert.Equal(t, 2, actual.OldLineCount, "OldLineCount")
	assert.Equal(t, 7, actual.NewLineCount, "NewLineCount")

	// All inserted lines should have NewLineNum set
	insertionCount := 0
	for _, change := range actual.Changes {
		if change.Type == LineAddition {
			insertionCount++
			assert.True(t, change.NewLineNum > 0, "insertion has valid NewLineNum")
		}
	}
	assert.Equal(t, 5, insertionCount, "5 insertions detected")

	// Verify line 1 and line 2 still map correctly
	assert.Equal(t, 1, actual.LineMapping.NewToOld[0], "new line 1 maps to old line 1")
	assert.Equal(t, 2, actual.LineMapping.NewToOld[6], "new line 7 maps to old line 2")
}
