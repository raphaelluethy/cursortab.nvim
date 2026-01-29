package engine

import (
	"context"
	"cursortab/assert"
	"cursortab/buffer"
	"cursortab/text"
	"cursortab/types"
	"sync"
	"testing"
	"time"
)

// --- Mock implementations ---

// mockBuffer implements the Buffer interface for testing
type mockBuffer struct {
	mu             sync.Mutex
	lines          []string
	row            int
	col            int
	path           string
	version        int
	viewportTop    int
	viewportBottom int
	previousLines  []string
	originalLines  []string
	diffHistories  []*types.DiffEntry
	linterErrors   *types.LinterErrors

	// Track method calls
	syncCalls              int
	clearUICalls           int
	commitPendingCalls     int
	showCursorTargetLine   int
	prepareCompletionCalls int
	lastPreparedCompletion struct {
		startLine  int
		endLineInc int
		lines      []string
	}
}

func newMockBuffer() *mockBuffer {
	return &mockBuffer{
		lines:          []string{"line 1", "line 2", "line 3"},
		row:            1,
		col:            0,
		path:           "test.go",
		version:        1,
		viewportTop:    1,
		viewportBottom: 50,
	}
}

func (b *mockBuffer) Sync(workspacePath string) (*buffer.SyncResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.syncCalls++
	return &buffer.SyncResult{BufferChanged: false}, nil
}

func (b *mockBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lines
}

func (b *mockBuffer) Row() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.row
}

func (b *mockBuffer) Col() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.col
}

func (b *mockBuffer) Path() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.path
}

func (b *mockBuffer) Version() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.version
}

func (b *mockBuffer) ViewportBounds() (top, bottom int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.viewportTop, b.viewportBottom
}

func (b *mockBuffer) PreviousLines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.previousLines
}

func (b *mockBuffer) OriginalLines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.originalLines
}

func (b *mockBuffer) DiffHistories() []*types.DiffEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.diffHistories
}

func (b *mockBuffer) SetFileContext(prev, orig []string, diffs []*types.DiffEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.previousLines = prev
	b.originalLines = orig
	b.diffHistories = diffs
}

func (b *mockBuffer) HasChanges(startLine, endLineInc int, lines []string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Simple implementation: check if any line differs
	for i := startLine; i <= endLineInc && i-1 < len(b.lines); i++ {
		relIdx := i - startLine
		if relIdx < len(lines) {
			if i-1 < len(b.lines) && lines[relIdx] != b.lines[i-1] {
				return true
			}
		}
	}
	return len(lines) != (endLineInc - startLine + 1)
}

func (b *mockBuffer) PrepareCompletion(startLine, endLineInc int, lines []string, groups []*text.Group) buffer.Batch {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.prepareCompletionCalls++
	b.lastPreparedCompletion.startLine = startLine
	b.lastPreparedCompletion.endLineInc = endLineInc
	b.lastPreparedCompletion.lines = lines
	return &mockBatch{}
}

func (b *mockBuffer) CommitPending() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.commitPendingCalls++
}

func (b *mockBuffer) CommitUserEdits() bool {
	return false
}

func (b *mockBuffer) ShowCursorTarget(line int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.showCursorTargetLine = line
	return nil
}

func (b *mockBuffer) ClearUI() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clearUICalls++
	return nil
}

func (b *mockBuffer) MoveCursor(line int, center, mark bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.row = line
	return nil
}

func (b *mockBuffer) LinterErrors() *types.LinterErrors {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.linterErrors
}

func (b *mockBuffer) RegisterEventHandler(handler func(event string)) error {
	return nil
}

// mockBatch implements buffer.Batch
type mockBatch struct {
	executed bool
}

func (b *mockBatch) Execute() error {
	b.executed = true
	return nil
}

// mockProvider implements the Provider interface for testing
type mockProvider struct {
	mu              sync.Mutex
	completionResp  *types.CompletionResponse
	completionErr   error
	completionCalls int
	lastRequest     *types.CompletionRequest
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		completionResp: &types.CompletionResponse{
			Completions: []*types.Completion{{
				StartLine:  1,
				EndLineInc: 1,
				Lines:      []string{"completed line 1"},
			}},
		},
	}
}

func (p *mockProvider) GetCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completionCalls++
	p.lastRequest = req
	if p.completionErr != nil {
		return nil, p.completionErr
	}
	return p.completionResp, nil
}

// mockClock implements Clock for testing
type mockClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*mockTimer
}

func newMockClock() *mockClock {
	return &mockClock{
		now: time.Now(),
	}
}

func (c *mockClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &mockTimer{
		fireTime: c.now.Add(d),
		f:        f,
		stopped:  false,
	}
	c.timers = append(c.timers, t)
	return t
}

func (c *mockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mockClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	// Copy timers to avoid holding lock during callback
	var toFire []*mockTimer
	for _, t := range c.timers {
		if !t.stopped && !t.now.After(c.now) {
			toFire = append(toFire, t)
		}
	}
	c.mu.Unlock()

	for _, t := range toFire {
		t.fire()
	}
}

type mockTimer struct {
	fireTime time.Time
	now      time.Time
	f        func()
	stopped  bool
	mu       sync.Mutex
}

func (t *mockTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	wasActive := !t.stopped
	t.stopped = true
	return wasActive
}

func (t *mockTimer) fire() {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return
	}
	t.stopped = true
	f := t.f
	t.mu.Unlock()
	if f != nil {
		f()
	}
}

// --- Helper functions ---

func createTestEngine(buf *mockBuffer, prov *mockProvider, clock *mockClock) *Engine {
	eng, _ := NewEngine(prov, buf, EngineConfig{
		NsID:                1,
		CompletionTimeout:   5 * time.Second,
		IdleCompletionDelay: 500 * time.Millisecond,
		TextChangeDebounce:  100 * time.Millisecond,
		CursorPrediction: CursorPredictionConfig{
			Enabled:            true,
			AutoAdvance:        true,
			ProximityThreshold: 3,
		},
	}, clock)
	return eng
}

// --- Tests ---

func TestEngineCreation(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()

	eng := createTestEngine(buf, prov, clock)

	assert.NotNil(t, eng, "NewEngine")
	assert.Equal(t, stateIdle, eng.state, "initial state")
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state state
		want  string
	}{
		{stateIdle, "Idle"},
		{statePendingCompletion, "PendingCompletion"},
		{stateHasCompletion, "HasCompletion"},
		{stateHasCursorTarget, "HasCursorTarget"},
		{stateStreamingCompletion, "StreamingCompletion"},
		{state(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		assert.Equal(t, tt.want, got, "state String")
	}
}

func TestFindTransition(t *testing.T) {
	tests := []struct {
		from  state
		event EventType
		want  bool // whether a transition should exist
	}{
		{stateIdle, EventTextChangeTimeout, true},
		{stateIdle, EventIdleTimeout, true},
		{stateIdle, EventTextChanged, true},
		{stateIdle, EventTab, false}, // No Tab handler from Idle
		{statePendingCompletion, EventTextChanged, true},
		{statePendingCompletion, EventEsc, true},
		{stateHasCompletion, EventTab, true},
		{stateHasCompletion, EventEsc, true},
		{stateHasCompletion, EventTextChanged, true},
		{stateHasCursorTarget, EventTab, true},
		{stateStreamingCompletion, EventTextChanged, true},
		{stateStreamingCompletion, EventEsc, true},
	}

	for _, tt := range tests {
		trans := findTransition(tt.from, tt.event)
		got := trans != nil
		assert.Equal(t, tt.want, got, "findTransition")
	}
}

func TestEventTypeFromString(t *testing.T) {
	tests := []struct {
		input string
		want  EventType
	}{
		{"esc", EventEsc},
		{"text_changed", EventTextChanged},
		{"tab", EventTab},
		{"insert_enter", EventInsertEnter},
		{"insert_leave", EventInsertLeave},
		{"unknown_event", ""},
	}

	for _, tt := range tests {
		got := EventTypeFromString(tt.input)
		assert.Equal(t, tt.want, got, "EventTypeFromString")
	}
}

func TestCheckTypingMatchesPrediction_NoCompletions(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// No completions set
	matches, hasRemaining := eng.checkTypingMatchesPrediction()
	assert.False(t, matches, "matches when no completions")
	assert.False(t, hasRemaining, "hasRemaining when no completions")
}

func TestCheckTypingMatchesPrediction_MatchesPrefix(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"hello wo"}
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Set up completion state
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"hello world"},
	}}
	eng.completionOriginalLines = []string{"hello "}

	matches, hasRemaining := eng.checkTypingMatchesPrediction()
	assert.True(t, matches, "match when buffer is prefix of target")
	assert.True(t, hasRemaining, "hasRemaining when buffer hasn't fully matched target")
}

func TestCheckTypingMatchesPrediction_FullyTyped(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"hello world"}
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Set up completion state - user typed everything
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"hello world"},
	}}
	eng.completionOriginalLines = []string{"hello "}

	matches, hasRemaining := eng.checkTypingMatchesPrediction()
	assert.True(t, matches, "match when buffer matches target")
	assert.False(t, hasRemaining, "hasRemaining when buffer fully matches target")
}

func TestCheckTypingMatchesPrediction_NoMatch(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"hello universe"}
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Set up completion state - user typed something different
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"hello world"},
	}}
	eng.completionOriginalLines = []string{"hello "}

	matches, _ := eng.checkTypingMatchesPrediction()
	assert.False(t, matches, "match when buffer diverges from target")
}

func TestReject(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Set up some state
	eng.state = stateHasCompletion
	eng.completions = []*types.Completion{{StartLine: 1, EndLineInc: 1, Lines: []string{"test"}}}
	eng.cursorTarget = &types.CursorPredictionTarget{LineNumber: 5}

	eng.reject()

	assert.Equal(t, stateIdle, eng.state, "state after reject")
	assert.Nil(t, eng.completions, "completions after reject")
	assert.Nil(t, eng.cursorTarget, "cursorTarget after reject")
	assert.Greater(t, buf.clearUICalls, 0, "ClearUI should have been called")
}

func TestClearState_Options(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Set up state
	eng.completions = []*types.Completion{{StartLine: 1, EndLineInc: 1, Lines: []string{"test"}}}
	eng.stagedCompletion = &types.StagedCompletion{CurrentIdx: 0}
	eng.cursorTarget = &types.CursorPredictionTarget{LineNumber: 5}

	// Clear with ClearStaged=false should preserve stagedCompletion
	eng.clearState(ClearOptions{
		ClearStaged:       false,
		ClearCursorTarget: true,
		CallOnReject:      true,
	})

	if eng.stagedCompletion == nil {
		assert.NotNil(t, eng.stagedCompletion, "stagedCompletion should be preserved when ClearStaged=false")
	}
	assert.Nil(t, eng.cursorTarget, "cursorTarget should be cleared when ClearCursorTarget=true")
	assert.Nil(t, eng.completions, "completions should always be cleared")
}

func TestAbsFunction(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
	}

	for _, tt := range tests {
		got := abs(tt.input)
		assert.Equal(t, tt.want, got, "abs")
	}
}

func TestCopyLines(t *testing.T) {
	original := []string{"a", "b", "c"}
	copied := copyLines(original)

	assert.Equal(t, len(original), len(copied), "copied length")

	// Modify original
	original[0] = "modified"

	assert.NotEqual(t, original[0], copied[0], "copyLines should create a deep copy")
}

func TestCopyLines_Nil(t *testing.T) {
	copied := copyLines(nil)
	assert.Nil(t, copied, "copyLines(nil)")
}

func TestIsFileStateValid(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	tests := []struct {
		name         string
		state        *FileState
		currentLines []string
		want         bool
	}{
		{
			name:         "empty original lines",
			state:        &FileState{OriginalLines: []string{}},
			currentLines: []string{"a", "b"},
			want:         false,
		},
		{
			name:         "same content",
			state:        &FileState{OriginalLines: []string{"a", "b", "c"}},
			currentLines: []string{"a", "b", "c"},
			want:         true,
		},
		{
			name:         "minor difference",
			state:        &FileState{OriginalLines: []string{"a", "b", "c"}},
			currentLines: []string{"a", "b", "c", "d"},
			want:         true,
		},
		{
			name:         "major line count difference",
			state:        &FileState{OriginalLines: []string{"a", "b", "c"}},
			currentLines: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eng.isFileStateValid(tt.state, tt.currentLines)
			assert.Equal(t, tt.want, got, "isFileStateValid")
		})
	}
}

func TestTrimFileStateStore(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Add 5 file states
	for i := 0; i < 5; i++ {
		eng.fileStateStore[string(rune('a'+i))+".go"] = &FileState{
			LastAccessNs: int64(i * 1000),
		}
	}

	eng.trimFileStateStore(2)

	assert.Equal(t, 2, len(eng.fileStateStore), "file state store size")

	// Should keep the most recently accessed (highest LastAccessNs)
	_, existsD := eng.fileStateStore["d.go"]
	assert.True(t, existsD, "should keep d.go (second most recent)")
	_, existsE := eng.fileStateStore["e.go"]
	assert.True(t, existsE, "should keep e.go (most recent)")
}

func TestPrefetchStateString(t *testing.T) {
	tests := []struct {
		state prefetchState
		want  string
	}{
		{prefetchNone, "None"},
		{prefetchInFlight, "InFlight"},
		{prefetchWaitingForTab, "WaitingForTab"},
		{prefetchWaitingForCursorPrediction, "WaitingForCursorPrediction"},
		{prefetchReady, "Ready"},
		{prefetchState(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		assert.Equal(t, tt.want, got, "prefetchState String")
	}
}

// --- Token Streaming Keep Partial Tests ---

func TestTokenStreamingKeepPartial_TypingMatchesPartial(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"hello wo"} // User typed "wo" which matches partial stream
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Simulate token streaming state with partial result
	eng.state = stateStreamingCompletion
	eng.tokenStreamingState = &TokenStreamingState{
		AccumulatedText: "world",
		LinePrefix:      "hello ",
		LineNum:         1,
	}
	// This would have been set by handleTokenChunk
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"hello world"},
	}}
	eng.completionOriginalLines = []string{"hello "}
	eng.tokenStreamChan = make(chan string) // Non-nil to indicate active stream

	// Trigger text change during streaming
	eng.doRejectStreamingAndDebounce(Event{Type: EventTextChanged})

	// Should transition to HasCompletion state since typing matches
	assert.Equal(t, stateHasCompletion, eng.state, "state after matching typing during streaming")

	// Completions should be preserved
	assert.Greater(t, len(eng.completions), 0, "completions count")

	// Token streaming state should be cleared
	assert.Nil(t, eng.tokenStreamingState, "tokenStreamingState after cancellation")
}

func TestTokenStreamingKeepPartial_TypingDoesNotMatch(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"hello xyz"} // User typed something different
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Simulate token streaming state with partial result
	eng.state = stateStreamingCompletion
	eng.tokenStreamingState = &TokenStreamingState{
		AccumulatedText: "world",
		LinePrefix:      "hello ",
		LineNum:         1,
	}
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"hello world"},
	}}
	eng.completionOriginalLines = []string{"hello "}
	eng.tokenStreamChan = make(chan string)

	// Trigger text change during streaming
	eng.doRejectStreamingAndDebounce(Event{Type: EventTextChanged})

	// Should transition to Idle state since typing doesn't match
	assert.Equal(t, stateIdle, eng.state, "state after mismatching typing during streaming")

	// Completions should be cleared
	assert.Nil(t, eng.completions, "completions after mismatch")

	// ClearUI should have been called
	assert.Greater(t, buf.clearUICalls, 0, "ClearUI should have been called")
}

func TestTokenStreamingKeepPartial_FullyTyped(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"hello world"} // User typed the full completion
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Simulate token streaming state with partial result
	eng.state = stateStreamingCompletion
	eng.tokenStreamingState = &TokenStreamingState{
		AccumulatedText: "world",
		LinePrefix:      "hello ",
		LineNum:         1,
	}
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"hello world"},
	}}
	eng.completionOriginalLines = []string{"hello "}
	eng.tokenStreamChan = make(chan string)

	// Trigger text change during streaming
	eng.doRejectStreamingAndDebounce(Event{Type: EventTextChanged})

	// Should transition to Idle since fully typed
	assert.Equal(t, stateIdle, eng.state, "state after fully typing completion during streaming")
}

func TestLineStreamingReject_NoKeepPartial(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Simulate line streaming state (NOT token streaming)
	eng.state = stateStreamingCompletion
	eng.streamingState = &StreamingState{} // Line streaming state
	eng.tokenStreamingState = nil          // No token streaming
	eng.streamLinesChan = make(chan string)

	// Trigger text change during streaming
	eng.doRejectStreamingAndDebounce(Event{Type: EventTextChanged})

	// Should transition to Idle (line streaming doesn't keep partial)
	assert.Equal(t, stateIdle, eng.state, "state after rejecting line streaming")
}

func TestCancelTokenStreamingKeepPartial(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Set up token streaming state
	eng.tokenStreamChan = make(chan string)
	eng.tokenStreamingState = &TokenStreamingState{
		AccumulatedText: "test",
		LineNum:         1,
	}
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 1,
		Lines:      []string{"test line"},
	}}
	eng.completionOriginalLines = []string{""}

	// Cancel keeping partial
	eng.cancelTokenStreamingKeepPartial()

	// Token stream channel should be nil
	assert.Nil(t, eng.tokenStreamChan, "tokenStreamChan after cancel")

	// Token streaming state should be nil
	assert.Nil(t, eng.tokenStreamingState, "tokenStreamingState after cancel")

	// But completions should be preserved
	assert.NotNil(t, eng.completions, "completions after cancel")

	// And completionOriginalLines should be preserved
	assert.NotNil(t, eng.completionOriginalLines, "completionOriginalLines after cancel")
}

// --- Multi-line Completion Tests ---

func TestCheckTypingMatchesPrediction_MultiLine(t *testing.T) {
	buf := newMockBuffer()
	// User typed "co" on line 2, which is a prefix of "complete"
	buf.lines = []string{"line 1", "line 2 co"}
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Multi-line completion
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 2,
		Lines:      []string{"line 1", "line 2 complete"},
	}}
	eng.completionOriginalLines = []string{"line 1", "line 2 "}

	matches, hasRemaining := eng.checkTypingMatchesPrediction()
	assert.True(t, matches, "match for multi-line partial completion")
	assert.True(t, hasRemaining, "hasRemaining for multi-line partial completion")
}

func TestCheckTypingMatchesPrediction_DeletionNotSupported(t *testing.T) {
	buf := newMockBuffer()
	buf.lines = []string{"line 1"}
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	// Completion that deletes lines (fewer target lines than original)
	eng.completions = []*types.Completion{{
		StartLine:  1,
		EndLineInc: 2,
		Lines:      []string{"combined line"}, // 1 line instead of 2
	}}
	eng.completionOriginalLines = []string{"line 1", "line 2"} // 2 original lines

	matches, _ := eng.checkTypingMatchesPrediction()
	assert.False(t, matches, "match when completion deletes lines")
}

// --- Dispatch Tests ---

func TestDispatch_ValidTransition(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	eng.state = stateIdle

	result := eng.dispatch(Event{Type: EventTextChanged})

	assert.True(t, result, "dispatch valid transition")
}

func TestDispatch_InvalidTransition(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)

	eng.state = stateIdle

	// Tab from Idle has no transition defined
	result := eng.dispatch(Event{Type: EventTab})

	assert.False(t, result, "dispatch invalid transition")
}

// --- Handle Cursor Target Tests ---

func TestHandleCursorTarget_Disabled(t *testing.T) {
	buf := newMockBuffer()
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)
	eng.config.CursorPrediction.Enabled = false

	eng.cursorTarget = &types.CursorPredictionTarget{LineNumber: 10}
	eng.state = stateHasCursorTarget

	eng.handleCursorTarget()

	// Should transition to idle when cursor prediction disabled
	assert.Equal(t, stateIdle, eng.state, "state when cursor prediction disabled")
	// cursorTarget should be cleared
	assert.Nil(t, eng.cursorTarget, "cursorTarget when cursor prediction disabled")
}

func TestHandleCursorTarget_CloseEnough(t *testing.T) {
	buf := newMockBuffer()
	buf.row = 8
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)
	eng.config.CursorPrediction.Enabled = true
	eng.config.CursorPrediction.ProximityThreshold = 3

	// Cursor target is only 2 lines away (within threshold)
	eng.cursorTarget = &types.CursorPredictionTarget{LineNumber: 10}

	eng.handleCursorTarget()

	// Should go idle (no prediction shown when close)
	assert.Equal(t, stateIdle, eng.state, "state when close enough")
}

func TestHandleCursorTarget_FarAway(t *testing.T) {
	buf := newMockBuffer()
	buf.row = 1
	prov := newMockProvider()
	clock := newMockClock()
	eng := createTestEngine(buf, prov, clock)
	eng.config.CursorPrediction.Enabled = true
	eng.config.CursorPrediction.ProximityThreshold = 3

	// Cursor target is 9 lines away (beyond threshold)
	eng.cursorTarget = &types.CursorPredictionTarget{LineNumber: 10}

	eng.handleCursorTarget()

	// Should show cursor target
	assert.Equal(t, stateHasCursorTarget, eng.state, "state when far away")
	assert.Equal(t, 10, buf.showCursorTargetLine, "showCursorTargetLine")
}
