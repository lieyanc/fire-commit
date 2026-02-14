package llm

import (
	"context"
	"strings"
	"sync"
)

// StreamChunk represents a piece of streamed text from the LLM.
type StreamChunk struct {
	Content string
	Done    bool
	Err     error
}

// GenerateOptions holds options for commit message generation.
type GenerateOptions struct {
	Language string
}

// Provider is the interface that all LLM providers must implement.
type Provider interface {
	// GenerateCommitMessages generates a single commit message suggestion.
	// It returns a channel of StreamChunk for streaming the response.
	GenerateCommitMessages(ctx context.Context, diff string, opts GenerateOptions) (<-chan StreamChunk, error)
}

// IndexedMessage holds the result of a single parallel LLM request.
type IndexedMessage struct {
	Index   int
	Content string
	Err     error
}

// GenerateMultiple launches n independent requests in parallel, each generating
// a single commit message. Results are sent on the returned channel as they
// complete. The channel is closed when all requests finish or the context is
// cancelled.
func GenerateMultiple(ctx context.Context, provider Provider, diff string, opts GenerateOptions, n int) <-chan IndexedMessage {
	ch := make(chan IndexedMessage, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			streamCh, err := provider.GenerateCommitMessages(ctx, diff, opts)
			if err != nil {
				ch <- IndexedMessage{Index: index, Err: err}
				return
			}

			var buf strings.Builder
			for chunk := range streamCh {
				if chunk.Err != nil {
					ch <- IndexedMessage{Index: index, Err: chunk.Err}
					return
				}
				if chunk.Done {
					break
				}
				buf.WriteString(chunk.Content)
			}

			msg := parseMessage(buf.String())
			if msg == "" {
				ch <- IndexedMessage{Index: index, Err: context.Canceled}
				return
			}
			ch <- IndexedMessage{Index: index, Content: msg}
		}(i)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}
