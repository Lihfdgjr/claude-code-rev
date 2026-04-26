package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type askUserTool struct{}

func NewAskUser() core.Tool { return &askUserTool{} }

func (t *askUserTool) Name() string { return "AskUserQuestion" }

func (t *askUserTool) Description() string {
	return "Pause the turn and ask the user a question. Returns the user's answer."
}

func (t *askUserTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"question": {"type": "string"}
		},
		"required": ["question"],
		"additionalProperties": false
	}`)
}

func (t *askUserTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(args.Question) == "" {
		return "", errors.New("ask user: empty question")
	}

	ch := core.UIEvents(ctx)
	if ch == nil {
		return "", errors.New("ask user: no UI channel attached")
	}

	reply := make(chan string, 1)
	ev := core.UIAskUserEvent{Question: args.Question, Reply: reply}

	select {
	case ch <- ev:
	case <-ctx.Done():
		return "<cancelled>", nil
	}

	select {
	case answer := <-reply:
		if strings.TrimSpace(answer) == "" {
			return "<cancelled>", nil
		}
		return answer, nil
	case <-ctx.Done():
		return "<cancelled>", nil
	}
}
