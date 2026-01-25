package text

import (
	"cursortab/assert"
	"fmt"
	"testing"
)

func TestClusterDistanceFromCursor(t *testing.T) {
	// Cluster with relative coordinates 1-6, baseLineOffset=10 means buffer lines 10-15
	cluster := &ChangeCluster{
		StartLine: 1,
		EndLine:   6,
		Changes: map[int]LineDiff{
			1: {Type: LineModification, LineNumber: 1, OldLineNum: 1, NewLineNum: 1},
			6: {Type: LineModification, LineNumber: 6, OldLineNum: 6, NewLineNum: 6},
		},
	}
	baseLineOffset := 10
	diff := &DiffResult{
		Changes:      cluster.Changes,
		OldLineCount: 6,
		NewLineCount: 6,
	}

	tests := []struct {
		cursorRow int // buffer coordinates
		expected  int
	}{
		{5, 5},  // cursor before cluster (buffer line 5, cluster starts at buffer 10)
		{10, 0}, // cursor at start (buffer line 10)
		{12, 0}, // cursor inside (buffer line 12)
		{15, 0}, // cursor at end (buffer line 15)
		{20, 5}, // cursor after cluster (buffer line 20, cluster ends at buffer 15)
	}

	for _, tt := range tests {
		result := clusterDistanceFromCursor(cluster, tt.cursorRow, baseLineOffset, diff)
		assert.Equal(t, tt.expected, result, fmt.Sprintf("distance for cursor at %d", tt.cursorRow))
	}
}

func TestClusterDistanceFromCursor_NoOffset(t *testing.T) {
	// When baseLineOffset=1, cluster coordinates match buffer coordinates
	cluster := &ChangeCluster{
		StartLine: 10,
		EndLine:   15,
		Changes: map[int]LineDiff{
			10: {Type: LineModification, LineNumber: 10, OldLineNum: 10, NewLineNum: 10},
			15: {Type: LineModification, LineNumber: 15, OldLineNum: 15, NewLineNum: 15},
		},
	}
	baseLineOffset := 1
	diff := &DiffResult{
		Changes:      cluster.Changes,
		OldLineCount: 15,
		NewLineCount: 15,
	}

	tests := []struct {
		cursorRow int
		expected  int
	}{
		{5, 5},  // cursor before cluster
		{10, 0}, // cursor at start
		{12, 0}, // cursor inside
		{15, 0}, // cursor at end
		{20, 5}, // cursor after cluster
	}

	for _, tt := range tests {
		result := clusterDistanceFromCursor(cluster, tt.cursorRow, baseLineOffset, diff)
		assert.Equal(t, tt.expected, result, fmt.Sprintf("distance for cursor at %d", tt.cursorRow))
	}
}

func TestJoinLines(t *testing.T) {
	lines := []string{"line1", "line2", "line3"}
	result := JoinLines(lines)
	expected := "line1\nline2\nline3"

	assert.Equal(t, expected, result, "JoinLines result")
}

func TestCreateStages_EmptyDiff(t *testing.T) {
	diff := &DiffResult{Changes: map[int]LineDiff{}}
	stages := CreateStages(diff, 10, 1, 50, 1, 3, "test.go", []string{})

	assert.Nil(t, stages, "stages for empty diff")
}

func TestCreateStages_SingleCluster(t *testing.T) {
	// All changes within proximity threshold - should return nil (no staging needed)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			10: {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
			11: {Type: LineModification, LineNumber: 11, Content: "new11", OldContent: "old11"},
			12: {Type: LineModification, LineNumber: 12, Content: "new12", OldContent: "old12"},
		},
	}
	newLines := make([]string, 20)
	for i := range newLines {
		newLines[i] = "line"
	}

	stages := CreateStages(diff, 10, 1, 50, 1, 3, "test.go", newLines)

	assert.Nil(t, stages, "stages for single cluster")
}

func TestCreateStages_TwoClusters(t *testing.T) {
	// Changes at lines 10-11 and 25-26 (gap > threshold of 3)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			10: {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
			11: {Type: LineModification, LineNumber: 11, Content: "new11", OldContent: "old11"},
			25: {Type: LineModification, LineNumber: 25, Content: "new25", OldContent: "old25"},
			26: {Type: LineModification, LineNumber: 26, Content: "new26", OldContent: "old26"},
		},
	}
	newLines := make([]string, 30)
	for i := range newLines {
		newLines[i] = "content"
	}

	// Cursor at line 15, baseLineOffset=1, so cluster 10-11 is closer
	result := CreateStages(diff, 15, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages")

	// First stage should be the closer cluster (10-11)
	assert.Equal(t, 10, result.Stages[0].Completion.StartLine, "first stage start line")
	assert.Equal(t, 11, result.Stages[0].Completion.EndLineInc, "first stage end line")

	// Second stage should be 25-26
	assert.Equal(t, 25, result.Stages[1].Completion.StartLine, "second stage start line")

	// First stage cursor target should point to next stage
	assert.Equal(t, 25, result.Stages[0].CursorTarget.LineNumber, "first stage cursor target line")
	assert.False(t, result.Stages[0].CursorTarget.ShouldRetrigger, "first stage ShouldRetrigger")

	// Last stage should have ShouldRetrigger=true
	assert.True(t, result.Stages[1].CursorTarget.ShouldRetrigger, "last stage ShouldRetrigger")
	assert.True(t, result.Stages[1].IsLastStage, "second stage IsLastStage")
}

func TestCreateStages_CursorDistanceSorting(t *testing.T) {
	// Three clusters: 5-6, 20-21, 35-36
	// Cursor at 22 - closest to cluster 20-21
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5:  {Type: LineModification, LineNumber: 5, Content: "new5", OldContent: "old5"},
			6:  {Type: LineModification, LineNumber: 6, Content: "new6", OldContent: "old6"},
			20: {Type: LineModification, LineNumber: 20, Content: "new20", OldContent: "old20"},
			21: {Type: LineModification, LineNumber: 21, Content: "new21", OldContent: "old21"},
			35: {Type: LineModification, LineNumber: 35, Content: "new35", OldContent: "old35"},
			36: {Type: LineModification, LineNumber: 36, Content: "new36", OldContent: "old36"},
		},
	}
	newLines := make([]string, 40)
	for i := range newLines {
		newLines[i] = "content"
	}

	result := CreateStages(diff, 22, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 3, result.Stages, "stages")

	// First stage should be closest to cursor (20-21)
	assert.Equal(t, 20, result.Stages[0].Completion.StartLine, "first stage should be closest cluster (20-21)")
}

func TestCreateStages_ViewportPartitioning(t *testing.T) {
	// Changes at lines 10 (in viewport) and 100 (out of viewport)
	// Viewport is 1-50
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			10:  {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
			100: {Type: LineModification, LineNumber: 100, Content: "new100", OldContent: "old100"},
		},
	}
	newLines := make([]string, 110)
	for i := range newLines {
		newLines[i] = "content"
	}

	// Cursor at 10, viewport 1-50
	result := CreateStages(diff, 10, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages")

	// In-view changes should come first (line 10)
	assert.Equal(t, 10, result.Stages[0].Completion.StartLine, "first stage should be in-viewport change")

	// Out-of-view change should be second (line 100)
	assert.Equal(t, 100, result.Stages[1].Completion.StartLine, "second stage should be out-of-viewport change")
}

func TestCreateStages_ProximityGrouping(t *testing.T) {
	// Changes at lines 10, 12, 14 (all within threshold of 3)
	// Should form single cluster, so no staging
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			10: {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
			12: {Type: LineModification, LineNumber: 12, Content: "new12", OldContent: "old12"},
			14: {Type: LineModification, LineNumber: 14, Content: "new14", OldContent: "old14"},
		},
	}
	newLines := make([]string, 20)
	for i := range newLines {
		newLines[i] = "content"
	}

	stages := CreateStages(diff, 10, 1, 50, 1, 3, "test.go", newLines)

	// Gap between 10->12 is 2, 12->14 is 2 - all within threshold
	assert.Nil(t, stages, "stages for single cluster (all within threshold)")
}

func TestCreateStages_ProximityGrouping_SplitByGap(t *testing.T) {
	// Changes at lines 10, 12 (gap=2) and 20 (gap=8 from 12)
	// With threshold=3, should split into two clusters
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			10: {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
			12: {Type: LineModification, LineNumber: 12, Content: "new12", OldContent: "old12"},
			20: {Type: LineModification, LineNumber: 20, Content: "new20", OldContent: "old20"},
		},
	}
	newLines := make([]string, 25)
	for i := range newLines {
		newLines[i] = "content"
	}

	result := CreateStages(diff, 10, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages (gap > threshold)")

	// First cluster should be 10-12
	assert.Equal(t, 10, result.Stages[0].Completion.StartLine, "first stage start line")
	assert.Equal(t, 12, result.Stages[0].Completion.EndLineInc, "first stage end line")

	// Second cluster should be 20
	assert.Equal(t, 20, result.Stages[1].Completion.StartLine, "second stage start line")
}

func TestCreateStages_WithBaseLineOffset(t *testing.T) {
	// Diff coordinates are relative (1-indexed from start of extraction)
	// baseLineOffset=50 means diff line 1 = buffer line 50
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			1:  {Type: LineModification, LineNumber: 1, Content: "new1", OldContent: "old1"},
			10: {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
		},
	}
	newLines := make([]string, 15)
	for i := range newLines {
		newLines[i] = "content"
	}

	// baseLineOffset=50, so diff line 1 = buffer line 50, diff line 10 = buffer line 59
	result := CreateStages(diff, 55, 1, 100, 50, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages")

	// Verify buffer coordinates in completion
	assert.True(t, result.Stages[0].Completion.StartLine == 50 || result.Stages[0].Completion.StartLine == 59,
		fmt.Sprintf("stage should have buffer coordinates (50 or 59), got %d", result.Stages[0].Completion.StartLine))
}

func TestCreateStages_VisualGroupsComputed(t *testing.T) {
	// Verify that visual groups are computed for stages
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			1: {Type: LineModification, LineNumber: 1, Content: "new1", OldContent: "old1"},
			2: {Type: LineModification, LineNumber: 2, Content: "new2", OldContent: "old2"},
			// Gap
			10: {Type: LineModification, LineNumber: 10, Content: "new10", OldContent: "old10"},
		},
	}
	newLines := []string{"new1", "new2", "", "", "", "", "", "", "", "new10"}

	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages")

	// First stage (lines 1-2) should have visual groups
	assert.NotNil(t, result.Stages[0].VisualGroups, "first stage visual groups")
}

func TestGroupChangesByProximity(t *testing.T) {
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5:  {Type: LineModification, LineNumber: 5},
			6:  {Type: LineModification, LineNumber: 6},
			7:  {Type: LineModification, LineNumber: 7},
			20: {Type: LineModification, LineNumber: 20},
			21: {Type: LineModification, LineNumber: 21},
		},
	}

	lineNumbers := []int{5, 6, 7, 20, 21}
	clusters := groupChangesByProximity(diff, lineNumbers, 3)

	assert.Len(t, 2, clusters, "clusters")

	// First cluster: 5-7
	assert.Equal(t, 5, clusters[0].StartLine, "first cluster start line")
	assert.Equal(t, 7, clusters[0].EndLine, "first cluster end line")
	assert.Equal(t, 3, len(clusters[0].Changes), "first cluster change count")

	// Second cluster: 20-21
	assert.Equal(t, 20, clusters[1].StartLine, "second cluster start line")
	assert.Equal(t, 21, clusters[1].EndLine, "second cluster end line")
}

func TestGroupChangesByProximity_EmptyInput(t *testing.T) {
	diff := &DiffResult{Changes: map[int]LineDiff{}}
	clusters := groupChangesByProximity(diff, []int{}, 3)

	assert.Nil(t, clusters, "clusters for empty input")
}

func TestGroupChangesByProximity_WithGroups(t *testing.T) {
	// Test that group types (LineModificationGroup) use EndLine for cluster boundaries
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5: {
				Type:       LineModificationGroup,
				LineNumber: 5,
				StartLine:  5,
				EndLine:    10, // Group spans 5-10
			},
			15: {Type: LineModification, LineNumber: 15},
		},
	}

	lineNumbers := []int{5, 15}
	clusters := groupChangesByProximity(diff, lineNumbers, 3)

	assert.Len(t, 2, clusters, "clusters")

	// First cluster should end at 10 (group EndLine), not 5
	assert.Equal(t, 10, clusters[0].EndLine, "first cluster should end at group EndLine")
}

func TestComputeVisualGroups(t *testing.T) {
	changes := map[int]LineDiff{
		1: {Type: LineModification, LineNumber: 1, Content: "new1", OldContent: "old1"},
		2: {Type: LineModification, LineNumber: 2, Content: "new2", OldContent: "old2"},
		3: {Type: LineModification, LineNumber: 3, Content: "new3", OldContent: "old3"},
		// Gap
		7: {Type: LineAddition, LineNumber: 7, Content: "added7"},
		8: {Type: LineAddition, LineNumber: 8, Content: "added8"},
	}
	newLines := []string{"new1", "new2", "new3", "", "", "", "added7", "added8"}
	oldLines := []string{"old1", "old2", "old3", "", "", "", "", ""}

	groups := computeVisualGroups(changes, newLines, oldLines)

	assert.Len(t, 2, groups, "groups")

	// First group: modifications 1-3
	assert.Equal(t, "modification", groups[0].Type, "first group type")
	assert.Equal(t, 1, groups[0].StartLine, "first group start line")
	assert.Equal(t, 3, groups[0].EndLine, "first group end line")
	assert.Equal(t, 3, len(groups[0].Lines), "first group line count")
	assert.Equal(t, 3, len(groups[0].OldLines), "first group old line count")

	// Second group: additions 7-8
	assert.Equal(t, "addition", groups[1].Type, "second group type")
	assert.Equal(t, 7, groups[1].StartLine, "second group start line")
	assert.Equal(t, 8, groups[1].EndLine, "second group end line")
}

func TestComputeVisualGroups_NonConsecutive(t *testing.T) {
	// Non-consecutive changes should form separate groups
	changes := map[int]LineDiff{
		1: {Type: LineModification, LineNumber: 1, Content: "new1", OldContent: "old1"},
		3: {Type: LineModification, LineNumber: 3, Content: "new3", OldContent: "old3"},
		5: {Type: LineModification, LineNumber: 5, Content: "new5", OldContent: "old5"},
	}
	newLines := []string{"new1", "", "new3", "", "new5"}
	oldLines := []string{"old1", "", "old3", "", "old5"}

	groups := computeVisualGroups(changes, newLines, oldLines)

	// Each non-consecutive change should be its own group
	assert.Equal(t, 3, len(groups), "group count for non-consecutive changes")
}

// =============================================================================
// Tests for staging with unequal line counts
// =============================================================================

func TestCreateStages_WithInsertions(t *testing.T) {
	// Test staging when completion adds lines (net line increase)
	// Old: 3 lines, New: 5 lines (2 insertions at different locations)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2: {Type: LineAddition, LineNumber: 2, NewLineNum: 2, OldLineNum: -1, Content: "inserted1"},
			5: {Type: LineAddition, LineNumber: 5, NewLineNum: 5, OldLineNum: -1, Content: "inserted2"},
		},
		OldLineCount: 3,
		NewLineCount: 5,
		LineMapping: &LineMapping{
			NewToOld: []int{1, -1, 2, 3, -1}, // line 2 and 5 are insertions
			OldToNew: []int{1, 3, 4},         // old lines map to new positions
		},
	}
	newLines := []string{"line1", "inserted1", "line2", "line3", "inserted2"}

	// Gap between line 2 and line 5 is 3, with threshold 2 they should be separate
	result := CreateStages(diff, 1, 1, 50, 1, 2, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages for separated insertions")

	// Verify stages have valid completions
	for i, stage := range result.Stages {
		assert.NotNil(t, stage.Completion, fmt.Sprintf("stage %d completion", i))
		assert.True(t, len(stage.Completion.Lines) > 0, fmt.Sprintf("stage %d has lines", i))
	}
}

func TestCreateStages_WithDeletions(t *testing.T) {
	// Test staging when completion removes lines (net line decrease)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2:  {Type: LineDeletion, LineNumber: 2, OldLineNum: 2, NewLineNum: -1, Content: "deleted1"},
			10: {Type: LineDeletion, LineNumber: 10, OldLineNum: 10, NewLineNum: -1, Content: "deleted2"},
		},
		OldLineCount: 12,
		NewLineCount: 10,
		LineMapping: &LineMapping{
			NewToOld: []int{1, 3, 4, 5, 6, 7, 8, 9, 11, 12},
			OldToNew: []int{1, -1, 2, 3, 4, 5, 6, 7, 8, -1, 9, 10},
		},
	}
	newLines := make([]string, 10)
	for i := range newLines {
		newLines[i] = "content"
	}

	// Gap is large (10-2=8), should create 2 stages
	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages for separated deletions")
}

func TestCreateStages_MixedInsertionDeletion(t *testing.T) {
	// Test with both insertions and deletions in different regions
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2:  {Type: LineAddition, LineNumber: 2, NewLineNum: 2, OldLineNum: 1, Content: "inserted"},
			15: {Type: LineDeletion, LineNumber: 15, OldLineNum: 15, NewLineNum: 14, Content: "deleted"},
		},
		OldLineCount: 20,
		NewLineCount: 20, // net zero change
		LineMapping: &LineMapping{
			NewToOld: make([]int, 20),
			OldToNew: make([]int, 20),
		},
	}
	// Initialize mappings
	for i := range diff.LineMapping.NewToOld {
		diff.LineMapping.NewToOld[i] = i + 1
	}
	for i := range diff.LineMapping.OldToNew {
		diff.LineMapping.OldToNew[i] = i + 1
	}
	diff.LineMapping.NewToOld[1] = -1 // line 2 is insertion

	newLines := make([]string, 20)
	for i := range newLines {
		newLines[i] = "content"
	}

	// Large gap (15-2=13), should create 2 stages
	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages")
}

func TestGetBufferLineForChange_Insertion(t *testing.T) {
	// Test that insertions correctly calculate buffer position
	mapping := &LineMapping{
		NewToOld: []int{1, -1, -1, 2}, // lines 2,3 are insertions after line 1
		OldToNew: []int{1, 4},
	}

	// Pure insertion at new line 2, should anchor to old line 1
	change := LineDiff{
		Type:       LineAddition,
		NewLineNum: 2,
		OldLineNum: -1, // pure insertion
	}

	bufferLine := getBufferLineForChange(change, 2, 1, mapping)

	// Should find old line 1 as anchor (the mapped line before insertion)
	assert.Equal(t, 1, bufferLine, "buffer line for insertion (anchor)")
}

func TestGetBufferLineForChange_Modification(t *testing.T) {
	// Test that modifications use OldLineNum directly
	mapping := &LineMapping{
		NewToOld: []int{1, 2, 3},
		OldToNew: []int{1, 2, 3},
	}

	change := LineDiff{
		Type:       LineModification,
		NewLineNum: 2,
		OldLineNum: 2,
	}

	bufferLine := getBufferLineForChange(change, 2, 10, mapping)

	// baseLineOffset=10, oldLineNum=2: 2 + 10 - 1 = 11
	assert.Equal(t, 11, bufferLine, "buffer line for modification")
}

func TestGetClusterBufferRange_WithInsertions(t *testing.T) {
	// Cluster containing insertions should still compute valid buffer range
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2: {Type: LineAddition, LineNumber: 2, NewLineNum: 2, OldLineNum: -1},
			3: {Type: LineAddition, LineNumber: 3, NewLineNum: 3, OldLineNum: -1},
		},
		LineMapping: &LineMapping{
			NewToOld: []int{1, -1, -1, 2}, // insertions at lines 2,3
			OldToNew: []int{1, 4},
		},
	}

	cluster := &ChangeCluster{
		StartLine: 2,
		EndLine:   3,
		Changes:   diff.Changes,
	}

	start, end := getClusterBufferRange(cluster, 1, diff)

	// Both insertions anchor to line 1, so range should be 1-1
	assert.True(t, start <= end, fmt.Sprintf("invalid range: start=%d > end=%d", start, end))
}

func TestCreateCompletionFromCluster_UnequalLineCounts(t *testing.T) {
	// Test that completion correctly extracts content when line counts differ
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2: {Type: LineModification, LineNumber: 2, NewLineNum: 2, OldLineNum: 2, Content: "modified"},
			3: {Type: LineAddition, LineNumber: 3, NewLineNum: 3, OldLineNum: -1, Content: "inserted"},
		},
		OldLineCount: 3,
		NewLineCount: 4,
		LineMapping: &LineMapping{
			NewToOld: []int{1, 2, -1, 3},
			OldToNew: []int{1, 2, 4},
		},
	}

	cluster := &ChangeCluster{
		StartLine: 2,
		EndLine:   3,
		Changes:   diff.Changes,
	}

	newLines := []string{"line1", "modified", "inserted", "line3"}

	completion := createCompletionFromCluster(cluster, newLines, 1, diff)

	// Should have 2 lines (from new line 2 to new line 3)
	assert.Equal(t, 2, len(completion.Lines), "completion line count")

	// Verify content
	assert.Equal(t, "modified", completion.Lines[0], "first completion line")
	assert.Equal(t, "inserted", completion.Lines[1], "second completion line")
}

// =============================================================================
// Edge case tests for staging with extreme coordinate scenarios
// =============================================================================

func TestGetBufferLineForChange_DeletionAtLine1(t *testing.T) {
	// Edge case: deletion at line 1 with no preceding anchor
	mapping := &LineMapping{
		NewToOld: []int{2, 3}, // old line 1 deleted
		OldToNew: []int{-1, 1, 2},
	}

	change := LineDiff{
		Type:       LineDeletion,
		OldLineNum: 1,
		NewLineNum: -1, // no new line for deletions
	}

	bufferLine := getBufferLineForChange(change, 1, 1, mapping)

	// Should use OldLineNum directly: 1 + 1 - 1 = 1
	assert.Equal(t, 1, bufferLine, "buffer line for deletion at line 1")
}

func TestGetBufferLineForChange_InsertionWithNoAnchor(t *testing.T) {
	// Edge case: insertion at line 1 with no preceding mapped line
	mapping := &LineMapping{
		NewToOld: []int{-1, 1}, // new line 1 is insertion, new line 2 maps to old line 1
		OldToNew: []int{2},
	}

	change := LineDiff{
		Type:       LineAddition,
		NewLineNum: 1,
		OldLineNum: -1, // pure insertion
	}

	bufferLine := getBufferLineForChange(change, 1, 1, mapping)

	// No anchor found, should fallback to mapKey: 1 + 1 - 1 = 1
	assert.Equal(t, 1, bufferLine, "buffer line for insertion at line 1 with no anchor")
}

func TestCreateStages_CumulativeOffsetScenario(t *testing.T) {
	// Scenario: first stage increases line count, affecting second stage coordinates
	// This tests the scenario where cumulative offset might exceed remaining line count
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2:  {Type: LineAddition, LineNumber: 2, NewLineNum: 2, OldLineNum: 1, Content: "insert1"},
			3:  {Type: LineAddition, LineNumber: 3, NewLineNum: 3, OldLineNum: 1, Content: "insert2"},
			20: {Type: LineModification, LineNumber: 20, NewLineNum: 22, OldLineNum: 20, Content: "modified"},
		},
		OldLineCount: 25,
		NewLineCount: 27, // +2 from insertions
		LineMapping: &LineMapping{
			NewToOld: make([]int, 27),
			OldToNew: make([]int, 25),
		},
	}
	// Initialize mappings
	for i := range diff.LineMapping.OldToNew {
		diff.LineMapping.OldToNew[i] = i + 3 // offset by 2 after line 1
	}
	diff.LineMapping.OldToNew[0] = 1
	for i := range diff.LineMapping.NewToOld {
		switch i {
		case 0:
			diff.LineMapping.NewToOld[i] = 1
		case 1, 2:
			diff.LineMapping.NewToOld[i] = -1 // insertions
		default:
			diff.LineMapping.NewToOld[i] = i - 1
		}
	}

	newLines := make([]string, 27)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}

	// Large gap (20-3=17), should create 2 stages
	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.Len(t, 2, result.Stages, "stages for insertions + distant modification")

	// Verify both stages have valid completions
	for i, stage := range result.Stages {
		assert.NotNil(t, stage.Completion, fmt.Sprintf("stage %d completion", i))
		assert.True(t, stage.Completion.StartLine > 0, fmt.Sprintf("stage %d has valid StartLine", i))
	}
}

func TestCreateStages_AllDeletions(t *testing.T) {
	// Edge case: completion that only contains deletions
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2: {Type: LineDeletion, LineNumber: 2, OldLineNum: 2, NewLineNum: -1, Content: "deleted1"},
			3: {Type: LineDeletion, LineNumber: 3, OldLineNum: 3, NewLineNum: -1, Content: "deleted2"},
		},
		OldLineCount: 5,
		NewLineCount: 3,
		LineMapping: &LineMapping{
			NewToOld: []int{1, 4, 5},
			OldToNew: []int{1, -1, -1, 2, 3},
		},
	}
	newLines := []string{"line1", "line4", "line5"}

	// Deletions are at same location, should form single cluster
	result := CreateStages(diff, 1, 1, 50, 1, 3, "test.go", newLines)

	// Single cluster = nil (no staging needed)
	assert.Nil(t, result, "single cluster of deletions")
}

func TestGetClusterBufferRange_AllInsertions(t *testing.T) {
	// Cluster containing only insertions (all OldLineNum = -1)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			2: {Type: LineAddition, LineNumber: 2, NewLineNum: 2, OldLineNum: -1},
			3: {Type: LineAddition, LineNumber: 3, NewLineNum: 3, OldLineNum: -1},
			4: {Type: LineAddition, LineNumber: 4, NewLineNum: 4, OldLineNum: -1},
		},
		LineMapping: &LineMapping{
			NewToOld: []int{1, -1, -1, -1, 2}, // lines 2,3,4 are insertions
			OldToNew: []int{1, 5},
		},
	}

	cluster := &ChangeCluster{
		StartLine: 2,
		EndLine:   4,
		Changes:   diff.Changes,
	}

	start, end := getClusterBufferRange(cluster, 1, diff)

	// All insertions anchor to line 1 (the mapped line before them)
	assert.True(t, start <= end, fmt.Sprintf("valid range: start=%d end=%d", start, end))
	assert.Equal(t, 1, start, "insertions anchor to line 1")
}

// =============================================================================
// Tests for visual group bounds validation
// =============================================================================

func TestStageVisualGroups_ShouldNotExceedStageContent(t *testing.T) {
	// Stage's visual groups should only reference lines within the stage's content.
	// When changes span multiple clusters, each stage's visual groups must be
	// bounded by that stage's line count.
	changes := make(map[int]LineDiff)

	// Add changes at low line numbers (will be in first cluster)
	for i := 1; i <= 17; i++ {
		changes[i] = LineDiff{
			Type:       LineAddition,
			LineNumber: i,
			NewLineNum: i,
			OldLineNum: -1,
			Content:    fmt.Sprintf("line%d", i),
		}
	}

	// Add changes at high line numbers (should be in separate cluster)
	for i := 41; i <= 54; i++ {
		changes[i] = LineDiff{
			Type:       LineAddition,
			LineNumber: i,
			NewLineNum: i,
			OldLineNum: -1,
			Content:    fmt.Sprintf("line%d", i),
		}
	}

	diff := &DiffResult{
		Changes:      changes,
		OldLineCount: 3,
		NewLineCount: 54,
	}

	// Create newLines for full completion
	newLines := make([]string, 54)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("content%d", i+1)
	}

	// Create stages with proximity threshold of 3
	// Gap between line 17 and 41 is 24, so they should be in separate clusters
	result := CreateStages(diff, 1, 1, 100, 1, 3, "test.go", newLines)

	// Should have at least 2 stages (may have more due to viewport partitioning)
	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 2, fmt.Sprintf("should have at least 2 stages, got %d", len(result.Stages)))

	// CRITICAL: For ALL stages, VisualGroups should NOT reference lines
	// beyond the stage's content line count
	for i, stage := range result.Stages {
		stageLineCount := len(stage.Completion.Lines)

		for _, vg := range stage.VisualGroups {
			assert.True(t, vg.StartLine <= stageLineCount,
				fmt.Sprintf("Stage %d: VisualGroup StartLine (%d) exceeds stage content line count (%d)",
					i, vg.StartLine, stageLineCount))
			assert.True(t, vg.EndLine <= stageLineCount,
				fmt.Sprintf("Stage %d: VisualGroup EndLine (%d) exceeds stage content line count (%d)",
					i, vg.EndLine, stageLineCount))
		}
	}
}

func TestComputeVisualGroups_ShouldNotReferenceOutOfBoundsLines(t *testing.T) {
	// Test that computeVisualGroups doesn't create groups with lines beyond newLines length

	// Cluster has changes at lines 1-5 and 41-45
	// But newLines only has 10 entries
	changes := map[int]LineDiff{
		1:  {Type: LineAddition, LineNumber: 1, NewLineNum: 1, Content: "line1"},
		2:  {Type: LineAddition, LineNumber: 2, NewLineNum: 2, Content: "line2"},
		3:  {Type: LineAddition, LineNumber: 3, NewLineNum: 3, Content: "line3"},
		41: {Type: LineAddition, LineNumber: 41, NewLineNum: 41, Content: "line41"},
		42: {Type: LineAddition, LineNumber: 42, NewLineNum: 42, Content: "line42"},
	}

	newLines := make([]string, 10) // Only 10 lines
	for i := range newLines {
		newLines[i] = fmt.Sprintf("content%d", i+1)
	}
	oldLines := make([]string, 10)

	groups := computeVisualGroups(changes, newLines, oldLines)

	// Visual groups should only reference lines within newLines bounds
	for _, vg := range groups {
		assert.True(t, vg.StartLine <= len(newLines),
			fmt.Sprintf("VisualGroup StartLine (%d) exceeds newLines length (%d)",
				vg.StartLine, len(newLines)))
		assert.True(t, vg.EndLine <= len(newLines),
			fmt.Sprintf("VisualGroup EndLine (%d) exceeds newLines length (%d)",
				vg.EndLine, len(newLines)))

		// Group's Lines should not be empty if StartLine is within bounds
		if vg.StartLine <= len(newLines) {
			assert.True(t, len(vg.Lines) > 0,
				fmt.Sprintf("VisualGroup at line %d has empty Lines", vg.StartLine))
		}
	}
}

func TestBuildStagesFromClusters_VisualGroupsUseClusterNewLines(t *testing.T) {
	// Test that buildStagesFromClusters computes visual groups using
	// only the lines relevant to each cluster, not the full newLines

	// Create a cluster with changes at lines 1-3
	cluster := &ChangeCluster{
		StartLine: 1,
		EndLine:   3,
		Changes: map[int]LineDiff{
			1: {Type: LineAddition, LineNumber: 1, NewLineNum: 1, OldLineNum: -1, Content: "new1"},
			2: {Type: LineAddition, LineNumber: 2, NewLineNum: 2, OldLineNum: -1, Content: "new2"},
			3: {Type: LineAddition, LineNumber: 3, NewLineNum: 3, OldLineNum: -1, Content: "new3"},
		},
	}

	// Full newLines has 50 entries, but the cluster only uses lines 1-3
	fullNewLines := make([]string, 50)
	for i := range fullNewLines {
		fullNewLines[i] = fmt.Sprintf("fullcontent%d", i+1)
	}

	diff := &DiffResult{
		Changes:      cluster.Changes,
		OldLineCount: 3,
		NewLineCount: 50,
	}

	stages := buildStagesFromClusters([]*ChangeCluster{cluster}, fullNewLines, "test.go", 1, diff)

	assert.Len(t, 1, stages, "should have 1 stage")

	stage := stages[0]

	// The stage's completion should only have 3 lines
	assert.Equal(t, 3, len(stage.Completion.Lines), "stage completion should have 3 lines")

	// The visual groups should only reference lines 1-3
	for _, vg := range stage.VisualGroups {
		assert.True(t, vg.EndLine <= 3,
			fmt.Sprintf("VisualGroup EndLine (%d) should be <= 3 (stage content size)", vg.EndLine))
	}
}

// =============================================================================
// Tests for additions at end of file: buffer range should extend to end of original
// =============================================================================

func TestGetClusterBufferRange_AdditionsAtEndOfFile(t *testing.T) {
	// When a cluster contains additions that extend beyond the original buffer,
	// the buffer range should extend to the end of the original buffer.
	// This ensures the stage correctly replaces the affected region.

	// Scenario: 10-line file, modification at line 8, additions at lines 9-12
	// Buffer range should be 8-10 (not 8-8)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			8:  {Type: LineModification, LineNumber: 8, NewLineNum: 8, OldLineNum: 8, Content: "modified", OldContent: "original"},
			9:  {Type: LineAddition, LineNumber: 9, NewLineNum: 9, OldLineNum: 8, Content: "added1"},
			10: {Type: LineAddition, LineNumber: 10, NewLineNum: 10, OldLineNum: 8, Content: "added2"},
			11: {Type: LineAddition, LineNumber: 11, NewLineNum: 11, OldLineNum: 8, Content: "added3"},
			12: {Type: LineAddition, LineNumber: 12, NewLineNum: 12, OldLineNum: 8, Content: "added4"},
		},
		OldLineCount: 10,
		NewLineCount: 14,
	}

	cluster := &ChangeCluster{
		StartLine: 8,
		EndLine:   12,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	assert.Equal(t, 8, startLine, "buffer start line")
	assert.Equal(t, 10, endLine, "buffer end line should extend to end of original buffer")
}

func TestGetClusterBufferRange_AdditionsWithinBuffer(t *testing.T) {
	// When additions don't extend beyond the original buffer,
	// the buffer range should not be artificially extended.

	// Scenario: 20-line file, additions at lines 5-7 (well within buffer)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5: {Type: LineAddition, LineNumber: 5, NewLineNum: 5, OldLineNum: 4, Content: "added1"},
			6: {Type: LineAddition, LineNumber: 6, NewLineNum: 6, OldLineNum: 4, Content: "added2"},
			7: {Type: LineAddition, LineNumber: 7, NewLineNum: 7, OldLineNum: 4, Content: "added3"},
		},
		OldLineCount: 20,
		NewLineCount: 23,
		LineMapping: &LineMapping{
			NewToOld: []int{1, 2, 3, 4, -1, -1, -1, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			OldToNew: []int{1, 2, 3, 4, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		},
	}

	cluster := &ChangeCluster{
		StartLine: 5,
		EndLine:   7,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// Additions anchor to line 4, range should be 4-4 (not extended to 20)
	assert.Equal(t, 4, startLine, "buffer start line")
	assert.Equal(t, 4, endLine, "buffer end line should not extend beyond anchor for mid-file additions")
}

func TestCreateStages_AdditionsAtEndOfFile(t *testing.T) {
	// End-to-end test: stage should have correct buffer range when
	// additions extend beyond the original buffer.

	// Scenario: 15-line file with changes at lines 12-18 (modification + 5 additions)
	diff := &DiffResult{
		Changes: map[int]LineDiff{
			12: {Type: LineModification, LineNumber: 12, NewLineNum: 12, OldLineNum: 12, Content: "modified", OldContent: "original"},
			13: {Type: LineAddition, LineNumber: 13, NewLineNum: 13, OldLineNum: 12, Content: "added1"},
			14: {Type: LineAddition, LineNumber: 14, NewLineNum: 14, OldLineNum: 12, Content: "added2"},
			15: {Type: LineAddition, LineNumber: 15, NewLineNum: 15, OldLineNum: 12, Content: "added3"},
			16: {Type: LineAddition, LineNumber: 16, NewLineNum: 16, OldLineNum: 12, Content: "added4"},
			17: {Type: LineAddition, LineNumber: 17, NewLineNum: 17, OldLineNum: 12, Content: "added5"},
			18: {Type: LineAddition, LineNumber: 18, NewLineNum: 18, OldLineNum: 12, Content: "added6"},
		},
		OldLineCount: 15,
		NewLineCount: 21,
	}

	newLines := make([]string, 21)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}

	// Cursor at line 1 (far from changes), viewport covers all
	result := CreateStages(diff, 1, 1, 30, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "result")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	// Find stage with changes at line 12+
	var stage *struct{ StartLine, EndLineInc, Lines int }
	for _, s := range result.Stages {
		if s.Completion.StartLine >= 12 {
			stage = &struct{ StartLine, EndLineInc, Lines int }{
				s.Completion.StartLine, s.Completion.EndLineInc, len(s.Completion.Lines),
			}
			break
		}
	}

	assert.NotNil(t, stage, "should have stage at line 12+")
	if stage != nil {
		assert.Equal(t, 12, stage.StartLine, "stage StartLine")
		assert.Equal(t, 15, stage.EndLineInc, "stage EndLineInc should extend to end of original buffer")
		assert.Equal(t, 7, stage.Lines, "stage should have 7 lines (new lines 12-18)")
	}
}

func TestGetClusterBufferRange_AdditionsAnchoredBeforeModifications(t *testing.T) {
	// When modifications exist at line N and additions are anchored to line N-1,
	// the buffer range should start at the first modification, not the anchor.

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Modifications at lines 43-44 (have OldLineNum)
			43: {Type: LineModification, LineNumber: 43, NewLineNum: 43, OldLineNum: 43, Content: "mod1", OldContent: "old1"},
			44: {Type: LineModification, LineNumber: 44, NewLineNum: 44, OldLineNum: 44, Content: "mod2", OldContent: "old2"},
			// Additions at lines 45-50, anchored to line 42 (line before modification region)
			45: {Type: LineAddition, LineNumber: 45, NewLineNum: 45, OldLineNum: 42, Content: "added1"},
			46: {Type: LineAddition, LineNumber: 46, NewLineNum: 46, OldLineNum: 42, Content: "added2"},
			47: {Type: LineAddition, LineNumber: 47, NewLineNum: 47, OldLineNum: 42, Content: "added3"},
			48: {Type: LineAddition, LineNumber: 48, NewLineNum: 48, OldLineNum: 42, Content: "added4"},
			49: {Type: LineAddition, LineNumber: 49, NewLineNum: 49, OldLineNum: 42, Content: "added5"},
			50: {Type: LineAddition, LineNumber: 50, NewLineNum: 50, OldLineNum: 42, Content: "added6"},
		},
		OldLineCount: 44,
		NewLineCount: 50,
	}

	cluster := &ChangeCluster{
		StartLine: 43,
		EndLine:   50,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// StartLine should be 43 (first modification), NOT 42 (anchor of additions)
	assert.Equal(t, 43, startLine,
		"buffer start should be 43 (first modification), not 42 (addition anchor)")

	// EndLine should be 44 (end of original buffer)
	assert.Equal(t, 44, endLine, "buffer end should be 44")
}

func TestGetClusterBufferRange_OnlyAdditionsWithAnchor(t *testing.T) {
	// When a cluster has ONLY additions (no modifications), the anchor determines
	// the buffer range. Additions insert "after" the anchor line.

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Only additions, all anchored to line 10
			11: {Type: LineAddition, LineNumber: 11, NewLineNum: 11, OldLineNum: 10, Content: "added1"},
			12: {Type: LineAddition, LineNumber: 12, NewLineNum: 12, OldLineNum: 10, Content: "added2"},
			13: {Type: LineAddition, LineNumber: 13, NewLineNum: 13, OldLineNum: 10, Content: "added3"},
		},
		OldLineCount: 15,
		NewLineCount: 18,
	}

	cluster := &ChangeCluster{
		StartLine: 11,
		EndLine:   13,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// For pure additions within original buffer bounds, anchor determines position
	// All additions anchored to line 10, so range is 10-10
	assert.Equal(t, 10, startLine, "buffer start should be anchor line 10")
	assert.Equal(t, 10, endLine, "buffer end should be anchor line 10 for mid-file insertions")
}

func TestGetClusterBufferRange_OnlyAdditionsBeyondBuffer(t *testing.T) {
	// When additions extend beyond the original buffer, extend endLine
	// to cover the full affected range.

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Additions beyond original buffer (OldLineCount=10, additions at new lines 11-15)
			11: {Type: LineAddition, LineNumber: 11, NewLineNum: 11, OldLineNum: 10, Content: "added1"},
			12: {Type: LineAddition, LineNumber: 12, NewLineNum: 12, OldLineNum: 10, Content: "added2"},
			13: {Type: LineAddition, LineNumber: 13, NewLineNum: 13, OldLineNum: 10, Content: "added3"},
			14: {Type: LineAddition, LineNumber: 14, NewLineNum: 14, OldLineNum: 10, Content: "added4"},
			15: {Type: LineAddition, LineNumber: 15, NewLineNum: 15, OldLineNum: 10, Content: "added5"},
		},
		OldLineCount: 10,
		NewLineCount: 15,
	}

	cluster := &ChangeCluster{
		StartLine: 11,
		EndLine:   15,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// Anchor is line 10, end extends to end of original buffer (10)
	assert.Equal(t, 10, startLine, "buffer start should be anchor line 10")
	assert.Equal(t, 10, endLine, "buffer end should be 10 (end of original buffer)")
}

func TestGetClusterBufferRange_AdditionGroupEndLineIsNewCoordinate(t *testing.T) {
	// LineAdditionGroup's EndLine is in NEW coordinates, not buffer coordinates.
	// The buffer range should not extend beyond the original buffer size.

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Modification at line 43 (exists in both old and new)
			43: {Type: LineModification, LineNumber: 43, NewLineNum: 43, OldLineNum: 43, Content: "mod", OldContent: "old"},
			// Addition at line 44
			44: {Type: LineAddition, LineNumber: 44, NewLineNum: 44, OldLineNum: 42, Content: "add1"},
			// Modification at line 45
			45: {Type: LineModification, LineNumber: 45, NewLineNum: 45, OldLineNum: 45, Content: "mod2", OldContent: "old2"},
			// Addition group spanning lines 49-56 in NEW content
			// EndLine=56 is in NEW coordinates, NOT buffer coordinates
			49: {
				Type:       LineAdditionGroup,
				LineNumber: 49,
				NewLineNum: 49,
				OldLineNum: 44, // Anchored to old line 44
				StartLine:  49,
				EndLine:    56, // This is NEW coordinate, buffer only has 45 lines!
			},
		},
		OldLineCount: 45,
		NewLineCount: 56,
	}

	cluster := &ChangeCluster{
		StartLine: 43,
		EndLine:   56,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// Buffer range should be based on OLD/buffer coordinates only
	// Modifications are at old lines 43, 45
	// Addition anchors are at old lines 42, 44
	// So buffer range should be approximately 42-45, NOT extending to 56
	assert.Equal(t, 43, startLine, "buffer start should be 43 (first modification)")
	assert.True(t, endLine <= 45,
		fmt.Sprintf("buffer end should be <= 45 (buffer size), got %d", endLine))
}

func TestGetClusterBufferRange_AdditionGroupWithZeroOldLineNum(t *testing.T) {
	// When addition_group has oldLineNum=0 and newLineNum=0, the fallback logic
	// should not extend the buffer range beyond the original buffer size.

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Addition with valid oldLineNum (anchor)
			44: {Type: LineAddition, LineNumber: 44, NewLineNum: 44, OldLineNum: 43, Content: "add1"},
			// Modification at line 45
			45: {Type: LineModification, LineNumber: 45, NewLineNum: 45, OldLineNum: 45, Content: "mod", OldContent: "old"},
			// Addition group with NO oldLineNum (common from diff analysis)
			46: {
				Type:       LineAdditionGroup,
				LineNumber: 46,
				NewLineNum: 0,  // No newLineNum
				OldLineNum: 0,  // No oldLineNum - will fallback to mapKey!
				StartLine:  46,
				EndLine:    47,
			},
			// replace_chars at line 48
			48: {Type: LineReplaceChars, LineNumber: 48, NewLineNum: 48, OldLineNum: 44, Content: "rep"},
			// Another addition group with NO oldLineNum
			49: {
				Type:       LineAdditionGroup,
				LineNumber: 49,
				NewLineNum: 0,  // No newLineNum
				OldLineNum: 0,  // No oldLineNum - will fallback to mapKey=49!
				StartLine:  49,
				EndLine:    56,
			},
		},
		OldLineCount: 45,
		NewLineCount: 56,
	}

	cluster := &ChangeCluster{
		StartLine: 44,
		EndLine:   56,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// Buffer range should NOT extend beyond the actual buffer (45 lines)
	// Non-additions are at old lines 45 (modification) and 44 (replace_chars)
	// So buffer end should be 45 at most
	assert.True(t, startLine >= 43 && startLine <= 45,
		fmt.Sprintf("buffer start should be 43-45, got %d", startLine))
	assert.True(t, endLine <= 45,
		fmt.Sprintf("buffer end should be <= 45 (buffer size), got %d", endLine))
}

// =============================================================================
// Edge case tests from code review
// =============================================================================

func TestCreateStages_EmptyNewLines(t *testing.T) {
	// Edge case: newLines slice is empty but diff has changes
	// Should handle gracefully without panic

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			1: {Type: LineModification, LineNumber: 1, NewLineNum: 1, OldLineNum: 1, Content: "mod", OldContent: "old"},
			5: {Type: LineModification, LineNumber: 5, NewLineNum: 5, OldLineNum: 5, Content: "mod2", OldContent: "old2"},
		},
		OldLineCount: 10,
		NewLineCount: 10,
	}

	// Empty newLines slice
	newLines := []string{}

	// Should not panic, should return result with empty Lines in stages
	result := CreateStages(diff, 3, 1, 20, 1, 2, "test.go", newLines)

	assert.NotNil(t, result, "result should not be nil")
	assert.True(t, len(result.Stages) >= 1, "should have stages")

	// Stages should have empty Lines arrays
	for i, stage := range result.Stages {
		assert.NotNil(t, stage.Completion, fmt.Sprintf("stage %d completion should not be nil", i))
		// Lines will be empty since newLines is empty
		assert.Equal(t, 0, len(stage.Completion.Lines), fmt.Sprintf("stage %d should have empty lines", i))
	}
}

func TestClusterDistanceFromCursor_CursorAtZero(t *testing.T) {
	// Edge case: cursor at line 0 (invalid but should handle gracefully)

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5: {Type: LineModification, LineNumber: 5, NewLineNum: 5, OldLineNum: 5, Content: "mod", OldContent: "old"},
		},
		OldLineCount: 10,
		NewLineCount: 10,
	}

	cluster := &ChangeCluster{
		StartLine: 5,
		EndLine:   5,
		Changes:   diff.Changes,
	}

	// Cursor at line 0
	distance := clusterDistanceFromCursor(cluster, 0, 1, diff)

	// Buffer line for change is 5, cursor is 0
	// Distance should be 5 - 0 = 5
	assert.Equal(t, 5, distance, "distance from cursor 0 to line 5")
}

func TestGetClusterBufferRange_AllAdditionsNoValidAnchor(t *testing.T) {
	// Edge case: all additions with OldLineNum=0 and no LineMapping
	// Should fall back to cluster.StartLine for minOldLine

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// All additions with no valid anchor (OldLineNum=0, no mapping)
			5: {Type: LineAddition, LineNumber: 5, NewLineNum: 5, OldLineNum: 0, Content: "added1"},
			6: {Type: LineAddition, LineNumber: 6, NewLineNum: 6, OldLineNum: 0, Content: "added2"},
			7: {Type: LineAddition, LineNumber: 7, NewLineNum: 7, OldLineNum: 0, Content: "added3"},
		},
		OldLineCount: 10,
		NewLineCount: 13,
		LineMapping:  nil, // No mapping available
	}

	cluster := &ChangeCluster{
		StartLine: 5,
		EndLine:   7,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 1, diff)

	// Should fall back to cluster.StartLine + baseLineOffset - 1 = 5 + 1 - 1 = 5
	assert.Equal(t, 5, startLine, "should fallback to cluster.StartLine")
	// For pure additions, maxOldLine = minOldLine
	assert.Equal(t, 5, endLine, "should equal startLine for pure additions")
}

func TestGetClusterBufferRange_BaseLineOffsetZero(t *testing.T) {
	// Edge case: baseLineOffset = 0
	// This is technically invalid (should be 1-indexed), but test behavior

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5: {Type: LineModification, LineNumber: 5, NewLineNum: 5, OldLineNum: 5, Content: "mod", OldContent: "old"},
		},
		OldLineCount: 10,
		NewLineCount: 10,
	}

	cluster := &ChangeCluster{
		StartLine: 5,
		EndLine:   5,
		Changes:   diff.Changes,
	}

	startLine, endLine := getClusterBufferRange(cluster, 0, diff)

	// With baseLineOffset=0: bufferLine = 5 + 0 - 1 = 4
	assert.Equal(t, 4, startLine, "buffer start with offset 0")
	assert.Equal(t, 4, endLine, "buffer end with offset 0")
}

func TestCreateStages_PartiallyVisibleSingleCluster_FarFromCursor(t *testing.T) {
	// Edge case: single cluster that is partially visible but far from cursor
	// clusterIsInViewport returns false (not entirely visible)
	// clusterNeedsNavigation returns true (far from cursor)
	// Should create a stage with FirstNeedsNavigation=true

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Cluster spans lines 45-55, viewport is 1-50, so partially visible
			45: {Type: LineModification, LineNumber: 45, NewLineNum: 45, OldLineNum: 45, Content: "mod1", OldContent: "old1"},
			50: {Type: LineModification, LineNumber: 50, NewLineNum: 50, OldLineNum: 50, Content: "mod2", OldContent: "old2"},
			55: {Type: LineModification, LineNumber: 55, NewLineNum: 55, OldLineNum: 55, Content: "mod3", OldContent: "old3"},
		},
		OldLineCount: 60,
		NewLineCount: 60,
	}

	newLines := make([]string, 60)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}

	// Cursor at line 5 (far from cluster at 45-55), viewport 1-50 (cluster partially visible)
	result := CreateStages(diff, 5, 1, 50, 1, 3, "test.go", newLines)

	// Single cluster that's partially visible but far from cursor
	// Should return a stage with FirstNeedsNavigation=true
	assert.NotNil(t, result, "should create staging result")
	assert.True(t, len(result.Stages) >= 1, "should have at least 1 stage")

	// Distance from cursor (5) to cluster start (45) = 40 > threshold (3)
	assert.True(t, result.FirstNeedsNavigation,
		"FirstNeedsNavigation should be true when cluster is far from cursor")
}

func TestCreateStages_PartiallyVisibleSingleCluster_CloseToCursor(t *testing.T) {
	// Edge case: single cluster that is partially visible but close to cursor
	// This is tricky: clusterIsInViewport=false, distance<=threshold
	// Current behavior: returns nil (no staging needed)

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			// Cluster spans lines 48-55, viewport is 1-50
			// Cursor at 47 is within threshold of 48
			48: {Type: LineModification, LineNumber: 48, NewLineNum: 48, OldLineNum: 48, Content: "mod1", OldContent: "old1"},
			55: {Type: LineModification, LineNumber: 55, NewLineNum: 55, OldLineNum: 55, Content: "mod2", OldContent: "old2"},
		},
		OldLineCount: 60,
		NewLineCount: 60,
	}

	newLines := make([]string, 60)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}

	// Cursor at line 47, viewport 1-50
	// Cluster is 48-55: partially visible (extends beyond viewport)
	// Distance from cursor to cluster = 48-47 = 1 <= threshold (3)
	result := CreateStages(diff, 47, 1, 50, 1, 3, "test.go", newLines)

	// In current logic: clusterIsInViewport checks if ENTIRELY within viewport
	// 48-55 is NOT entirely within 1-50, so inViewport=false
	// But distance=1 <= threshold, so condition `inViewport && distance <= proximityThreshold` is false
	// Therefore it falls through and creates a stage
	assert.NotNil(t, result, "should create staging result for partially visible cluster")
}

func TestCreateStages_SingleClusterEntirelyOutsideViewport(t *testing.T) {
	// Edge case: single cluster entirely outside viewport
	// Should return stage with FirstNeedsNavigation=true

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			100: {Type: LineModification, LineNumber: 100, NewLineNum: 100, OldLineNum: 100, Content: "mod", OldContent: "old"},
		},
		OldLineCount: 150,
		NewLineCount: 150,
	}

	newLines := make([]string, 150)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}

	// Cursor at line 10, viewport 1-50, cluster at 100 (entirely outside)
	result := CreateStages(diff, 10, 1, 50, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "should create staging result")
	assert.Equal(t, 1, len(result.Stages), "should have 1 stage")
	assert.True(t, result.FirstNeedsNavigation,
		"FirstNeedsNavigation should be true when cluster is outside viewport")
}

func TestClusterNeedsNavigation_PartiallyVisible(t *testing.T) {
	// Test clusterNeedsNavigation for partially visible cluster

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			45: {Type: LineModification, LineNumber: 45, NewLineNum: 45, OldLineNum: 45, Content: "mod1", OldContent: "old1"},
			55: {Type: LineModification, LineNumber: 55, NewLineNum: 55, OldLineNum: 55, Content: "mod2", OldContent: "old2"},
		},
		OldLineCount: 60,
		NewLineCount: 60,
	}

	cluster := &ChangeCluster{
		StartLine: 45,
		EndLine:   55,
		Changes:   diff.Changes,
	}

	// Viewport 1-50: cluster 45-55 is partially visible
	// Cursor at 47: distance to 45-55 is 0 (cursor within range)
	needsNav := clusterNeedsNavigation(cluster, 47, 1, 50, 1, diff, 3)

	// Not entirely outside viewport, cursor within cluster
	// So needsNav should be false
	assert.False(t, needsNav, "should not need navigation when cursor is within cluster")

	// Cursor at 10: distance to 45-55 is 35
	needsNav = clusterNeedsNavigation(cluster, 10, 1, 50, 1, diff, 3)

	// Not entirely outside viewport, but far from cursor
	assert.True(t, needsNav, "should need navigation when far from cursor")
}

func TestCreateStages_NoViewportInfo(t *testing.T) {
	// Edge case: viewportTop=0 and viewportBottom=0 (no viewport info)
	// All changes should be treated as visible

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			10:  {Type: LineModification, LineNumber: 10, NewLineNum: 10, OldLineNum: 10, Content: "mod1", OldContent: "old1"},
			100: {Type: LineModification, LineNumber: 100, NewLineNum: 100, OldLineNum: 100, Content: "mod2", OldContent: "old2"},
		},
		OldLineCount: 150,
		NewLineCount: 150,
	}

	newLines := make([]string, 150)
	for i := range newLines {
		newLines[i] = fmt.Sprintf("line%d", i+1)
	}

	// No viewport info (0, 0), cursor at 50
	result := CreateStages(diff, 50, 0, 0, 1, 3, "test.go", newLines)

	assert.NotNil(t, result, "should create staging result")
	assert.Equal(t, 2, len(result.Stages), "should have 2 stages (large gap between 10 and 100)")

	// Both should be treated as in-view, sorted by distance to cursor
	// Cursor at 50: distance to 10 is 40, distance to 100 is 50
	// So line 10 should be first
	assert.Equal(t, 10, result.Stages[0].Completion.StartLine, "closer stage first")
}

func TestGetClusterNewLineRange_NoNewLineNum(t *testing.T) {
	// Edge case: changes have NewLineNum=0, should fallback to cluster coordinates

	cluster := &ChangeCluster{
		StartLine: 5,
		EndLine:   10,
		Changes: map[int]LineDiff{
			5: {Type: LineModification, LineNumber: 5, NewLineNum: 0, OldLineNum: 5, Content: "mod"},
			7: {Type: LineModification, LineNumber: 7, NewLineNum: 0, OldLineNum: 7, Content: "mod"},
		},
	}

	startLine, endLine := getClusterNewLineRange(cluster)

	// Should fallback to cluster.StartLine and cluster.EndLine
	assert.Equal(t, 5, startLine, "should fallback to cluster.StartLine")
	assert.Equal(t, 10, endLine, "should fallback to cluster.EndLine")
}

func TestComputeVisualGroups_AllChangesOutOfBounds(t *testing.T) {
	// Edge case: all changes reference lines beyond newLines bounds

	changes := map[int]LineDiff{
		100: {Type: LineModification, LineNumber: 100, Content: "mod1"},
		101: {Type: LineModification, LineNumber: 101, Content: "mod2"},
	}

	newLines := []string{"line1", "line2", "line3"} // Only 3 lines
	oldLines := []string{"old1", "old2", "old3"}

	groups := computeVisualGroups(changes, newLines, oldLines)

	// All changes are out of bounds (100, 101 > 3), should return nil
	assert.Nil(t, groups, "should return nil when all changes are out of bounds")
}

func TestBuildStagesFromClusters_SingleDeletion(t *testing.T) {
	// Edge case: cluster with only deletions (no new content)

	diff := &DiffResult{
		Changes: map[int]LineDiff{
			5: {Type: LineDeletion, LineNumber: 5, NewLineNum: 0, OldLineNum: 5, OldContent: "deleted"},
		},
		OldLineCount: 10,
		NewLineCount: 9,
	}

	cluster := &ChangeCluster{
		StartLine: 5,
		EndLine:   5,
		Changes:   diff.Changes,
	}

	newLines := []string{"1", "2", "3", "4", "6", "7", "8", "9", "10"} // Line 5 deleted

	stages := buildStagesFromClusters([]*ChangeCluster{cluster}, newLines, "test.go", 1, diff)

	assert.Equal(t, 1, len(stages), "should have 1 stage")
	// For deletions, completion.Lines might be empty or contain surrounding context
	// The important thing is it doesn't panic
	assert.NotNil(t, stages[0].Completion, "completion should not be nil")
}
