package sweep

// AutocompleteRequest represents the request body for Sweep's autocomplete endpoint
// Based on Zed's implementation at crates/edit_prediction/src/sweep_ai.rs
type AutocompleteRequest struct {
	DebugInfo            string       `json:"debug_info"`
	RepoName             string       `json:"repo_name"`
	Branch               *string      `json:"branch"`
	FilePath             string       `json:"file_path"`
	FileContents         string       `json:"file_contents"`
	RecentChanges        string       `json:"recent_changes"`
	CursorPosition       int          `json:"cursor_position"`
	OriginalFileContents string       `json:"original_file_contents"`
	FileChunks           []FileChunk  `json:"file_chunks"`
	RetrievalChunks      []FileChunk  `json:"retrieval_chunks"`
	RecentUserActions    []UserAction `json:"recent_user_actions"`
	MultipleSuggestions  bool         `json:"multiple_suggestions"`
	PrivacyModeEnabled   bool         `json:"privacy_mode_enabled"`
	ChangesAboveCursor   bool         `json:"changes_above_cursor"`
	UseBytes             bool         `json:"use_bytes"`
}

// FileChunk represents a chunk of file content for context
type FileChunk struct {
	FilePath  string  `json:"file_path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Content   string  `json:"content"`
	Timestamp *uint64 `json:"timestamp,omitempty"`
}

// UserAction represents a user action record
type UserAction struct {
	ActionType string `json:"action_type"`
	LineNumber int    `json:"line_number"`
	Offset     int    `json:"offset"`
	FilePath   string `json:"file_path"`
	Timestamp  uint64 `json:"timestamp"`
}

// AutocompleteResponse represents the response from Sweep's autocomplete endpoint
type AutocompleteResponse struct {
	AutocompleteID string             `json:"autocomplete_id"`
	StartIndex     int                `json:"start_index"`
	EndIndex       int                `json:"end_index"`
	Completion     string             `json:"completion"`
	Confidence     float64            `json:"confidence"`
	Logprobs       interface{}        `json:"logprobs,omitempty"`
	FinishReason   *string            `json:"finish_reason,omitempty"`
	ElapsedTimeMs  int                `json:"elapsed_time_ms"`
	Completions    []CompletionChoice `json:"completions,omitempty"`
}

// CompletionChoice represents a single completion choice
type CompletionChoice struct {
	StartIndex     int         `json:"start_index"`
	EndIndex       int         `json:"end_index"`
	Completion     string      `json:"completion"`
	Confidence     float64     `json:"confidence"`
	AutocompleteID string      `json:"autocomplete_id"`
	Logprobs       interface{} `json:"logprobs,omitempty"`
	FinishReason   *string     `json:"finish_reason,omitempty"`
}

// MetricsRequest represents the request body for Sweep's metrics endpoint
type MetricsRequest struct {
	EventType          string  `json:"event_type"`
	SuggestionType     string  `json:"suggestion_type"`
	Additions          int     `json:"additions"`
	Deletions          int     `json:"deletions"`
	AutocompleteID     string  `json:"autocomplete_id"`
	EditTracking       string  `json:"edit_tracking"`
	EditTrackingLine   *int    `json:"edit_tracking_line,omitempty"`
	Lifespan           *uint64 `json:"lifespan,omitempty"`
	DebugInfo          string  `json:"debug_info"`
	DeviceID           string  `json:"device_id"`
	PrivacyModeEnabled bool    `json:"privacy_mode_enabled"`
}
