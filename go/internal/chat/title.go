package chat

import (
	"context"
	"strings"

	"claudecode/internal/core"
)

const titlePrompt = "Summarize the user's request in 4-7 words as a title. Output only the title text."

// GenerateTitle calls the model with a one-shot prompt and returns a short
// session title derived from the conversation so far.
func GenerateTitle(ctx context.Context, transport core.Transport, model string, history []core.Message) (string, error) {
	h := append([]core.Message(nil), history...)
	h = append(h, core.Message{
		Role:   core.RoleUser,
		Blocks: []core.Block{core.TextBlock{Text: titlePrompt}},
	})

	ch, err := transport.Stream(ctx, core.CallOptions{
		Model:     model,
		MaxTokens: 64,
	}, h)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for ev := range ch {
		switch e := ev.(type) {
		case core.TextDeltaEvent:
			sb.WriteString(e.Text)
		case core.ErrorEvent:
			return "", e.Err
		}
	}

	title := strings.TrimSpace(sb.String())
	title = strings.Trim(title, "\"'`")
	title = strings.TrimSpace(title)
	return title, nil
}
