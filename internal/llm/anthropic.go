package llm

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements Provider using the official Anthropic SDK.
type AnthropicProvider struct {
	client *anthropic.Client
	model  string
}

// NewAnthropicProvider creates a provider for the Anthropic API.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{client: &client, model: model}
}

func (p *AnthropicProvider) GenerateCommitMessages(ctx context.Context, diff string, opts GenerateOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 64)

	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		MaxTokens: 1024,
		Model:     anthropic.Model(p.model),
		System: []anthropic.TextBlockParam{
			{Text: buildSystemPrompt(opts.Language)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(buildUserPrompt(diff)),
			),
		},
	})

	go func() {
		defer close(ch)
		for stream.Next() {
			event := stream.Current()
			switch variant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch delta := variant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					if delta.Text != "" {
						ch <- StreamChunk{Content: delta.Text}
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			ch <- StreamChunk{Err: err}
			return
		}
		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}
