package llm

import "context"

// StreamChunk represents a piece of streamed text from the LLM.
type StreamChunk struct {
	Content string
	Done    bool
	Err     error
}

// GenerateOptions holds options for commit message generation.
type GenerateOptions struct {
	NumSuggestions int
	Language       string
}

// Provider is the interface that all LLM providers must implement.
type Provider interface {
	// GenerateCommitMessages generates multiple commit message suggestions.
	// It returns a channel of StreamChunk for streaming the response.
	GenerateCommitMessages(ctx context.Context, diff string, opts GenerateOptions) (<-chan StreamChunk, error)
}
