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

// IndexedMessageEvent is a streamed event from one parallel LLM request.
// Delta carries incremental text chunks. A terminal event has Done=true or Err.
type IndexedMessageEvent struct {
	Index   int
	Delta   string
	Content string
	Done    bool
	Err     error
}

// GenerateMultiple launches n independent requests in parallel and streams
// chunk-level events for each request.
func GenerateMultiple(ctx context.Context, provider Provider, diff string, opts GenerateOptions, n int) <-chan IndexedMessageEvent {
	buffer := n * 16
	if buffer < 64 {
		buffer = 64
	}
	ch := make(chan IndexedMessageEvent, buffer)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			streamCh, err := provider.GenerateCommitMessages(ctx, diff, opts)
			if err != nil {
				ch <- IndexedMessageEvent{Index: index, Err: err}
				return
			}

			var buf strings.Builder
			for chunk := range streamCh {
				if chunk.Err != nil {
					ch <- IndexedMessageEvent{Index: index, Err: chunk.Err}
					return
				}

				if chunk.Done {
					msg := parseMessage(buf.String())
					if msg == "" {
						ch <- IndexedMessageEvent{Index: index, Err: context.Canceled}
						return
					}
					ch <- IndexedMessageEvent{
						Index:   index,
						Content: msg,
						Done:    true,
					}
					return
				}

				if chunk.Content != "" {
					buf.WriteString(chunk.Content)
					ch <- IndexedMessageEvent{
						Index: index,
						Delta: chunk.Content,
					}
				}
			}

			// Defensive fallback: if provider closes without an explicit Done marker.
			msg := parseMessage(buf.String())
			if msg == "" {
				ch <- IndexedMessageEvent{Index: index, Err: context.Canceled}
				return
			}
			ch <- IndexedMessageEvent{
				Index:   index,
				Content: msg,
				Done:    true,
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}
