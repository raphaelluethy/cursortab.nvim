package sweep

// AutocompleteRequest represents the request body for Sweep's autocomplete endpoint
type AutocompleteRequest struct {
	FilePath     string        `json:"file_path"`
	Language     string        `json:"language"`
	Prefix       string        `json:"prefix"`
	Suffix       string        `json:"suffix"`
	CursorIndex  int           `json:"cursor_index"`
	DiffHistory  []DiffEntry   `json:"diff_history,omitempty"`
	LinterErrors []LinterError `json:"linter_errors,omitempty"`
}

// DiffEntry represents a single diff in the history
type DiffEntry struct {
	FileName string `json:"file_name"`
	Original string `json:"original"`
	Updated  string `json:"updated"`
}

// LinterError represents a linter error
type LinterError struct {
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
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
	AutocompleteID string  `json:"autocomplete_id"`
	Accepted       bool    `json:"accepted"`
	LatencyMs      int     `json:"latency_ms"`
	Confidence     float64 `json:"confidence"`
}
