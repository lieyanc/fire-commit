package llm

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// OpenAIProvider implements Provider using the official OpenAI SDK.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates a provider for the official OpenAI API.
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{client: &client, model: model}
}

func (p *OpenAIProvider) GenerateCommitMessages(ctx context.Context, diff string, opts GenerateOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 64)

	stream := p.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(buildSystemPrompt(opts.Language)),
			openai.UserMessage(buildUserPrompt(diff)),
		},
	})

	go func() {
		defer close(ch)
		for stream.Next() {
			evt := stream.Current()
			if len(evt.Choices) > 0 {
				content := evt.Choices[0].Delta.Content
				if content != "" {
					ch <- StreamChunk{Content: content}
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
