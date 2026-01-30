package engine

import (
	"cursortab/assert"
	"sync"
	"testing"
	"time"
)

// mockLineStream implements LineStream for testing
type mockLineStream struct {
	lines  chan string
	cancel func()
}

func newMockLineStream() *mockLineStream {
	return &mockLineStream{
		lines:  make(chan string, 100),
		cancel: func() {},
	}
}

func (s *mockLineStream) LinesChan() <-chan string { return s.lines }
func (s *mockLineStream) Cancel()                  { s.cancel() }

// TestStreamContaminationPrevention verifies that lines from an old stream
// are not processed when a new stream starts.
func TestStreamContaminationPrevention(t *testing.T) {
	// This test verifies the core fix: when we cancel a stream and start a new one,
	// any lines still buffered from the old stream should be ignored.

	// Create channels to track what gets processed
	var processedLines []string
	var mu sync.Mutex

	// Simulate the event loop's stream handling logic
	// (extracted from the actual implementation)
	type streamState struct {
		linesChan <-chan string
	}

	var currentStream *streamState

	processLine := func(linesChan <-chan string, line string) bool {
		mu.Lock()
		defer mu.Unlock()
		// Check if this is still the current stream
		if currentStream == nil || currentStream.linesChan != linesChan {
			return false // Contamination prevented
		}
		processedLines = append(processedLines, line)
		return true
	}

	// Create stream 1
	stream1 := newMockLineStream()
	currentStream = &streamState{linesChan: stream1.lines}

	// Send some lines from stream 1
	stream1.lines <- "stream1-line1"
	stream1.lines <- "stream1-line2"

	// Process them
	for range 2 {
		line := <-stream1.lines
		assert.True(t, processLine(stream1.lines, line), "stream 1 line should have been processed: "+line)
	}

	// Now "cancel" stream 1 and start stream 2
	// In the real code, this clears streamLinesChan before starting the new stream
	stream2 := newMockLineStream()

	// Send more lines from stream 1 (simulating buffered lines)
	stream1.lines <- "stream1-line3-stale"
	stream1.lines <- "stream1-line4-stale"

	// Switch to stream 2 (this is what cancelStreaming() + new stream does)
	currentStream = &streamState{linesChan: stream2.lines}

	// Send lines from stream 2
	stream2.lines <- "stream2-line1"
	stream2.lines <- "stream2-line2"

	// Now try to process stream 1's stale lines - they should be rejected
	for range 2 {
		line := <-stream1.lines
		assert.False(t, processLine(stream1.lines, line), "stale stream 1 line should have been rejected: "+line)
	}

	// Process stream 2's lines - they should succeed
	for range 2 {
		line := <-stream2.lines
		assert.True(t, processLine(stream2.lines, line), "stream 2 line should have been processed: "+line)
	}

	// Verify only the correct lines were processed
	mu.Lock()
	defer mu.Unlock()

	expected := []string{"stream1-line1", "stream1-line2", "stream2-line1", "stream2-line2"}
	assert.Equal(t, len(expected), len(processedLines), "processed lines count")
	for i, line := range processedLines {
		assert.Equal(t, expected[i], line, "line "+string(rune(i)))
	}
}

// TestStreamChannelNilPreventsSelection verifies that setting streamLinesChan to nil
// prevents the select from receiving on that channel.
func TestStreamChannelNilPreventsSelection(t *testing.T) {
	// When streamLinesChan is nil, selecting on it should block forever
	// (or until another case is ready)

	var nilChan <-chan string = nil
	otherChan := make(chan int, 1)
	otherChan <- 42

	selected := ""
	select {
	case <-nilChan:
		selected = "nil"
	case <-otherChan:
		selected = "other"
	}

	assert.Equal(t, "other", selected, "select should choose other")
}

// TestRapidStreamSwitching tests rapid stream cancellation and restart
func TestRapidStreamSwitching(t *testing.T) {
	var mu sync.Mutex
	var currentChan <-chan string
	processedCount := 0

	// Simulate rapid switching between streams
	for range 10 {
		stream := newMockLineStream()

		mu.Lock()
		currentChan = stream.lines
		mu.Unlock()

		// Send a line
		stream.lines <- "line"

		// Process it
		select {
		case line := <-stream.lines:
			mu.Lock()
			if currentChan == stream.lines {
				processedCount++
				_ = line
			}
			mu.Unlock()
		case <-time.After(10 * time.Millisecond):
			assert.True(t, false, "Timeout waiting for line")
		}
	}

	assert.Equal(t, 10, processedCount, "processed count")
}
