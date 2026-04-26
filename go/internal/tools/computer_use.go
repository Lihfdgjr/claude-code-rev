package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"claudecode/internal/computeruse"
	"claudecode/internal/core"
)

type computerUseTool struct {
	c computeruse.Computer
}

// NewComputerUse returns the ComputerUse tool. If c is nil a stub backend is used.
func NewComputerUse(c computeruse.Computer) core.Tool {
	if c == nil {
		c = computeruse.New()
	}
	return &computerUseTool{c: c}
}

func (t *computerUseTool) Name() string { return "ComputerUse" }

func (t *computerUseTool) Description() string {
	return "Drive the local desktop: screenshot, mouse click/move/scroll, keyboard type/key. In this build the backend is a stub and every action returns 'not supported'."
}

func (t *computerUseTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["screenshot","left_click","right_click","type","key","mouse_move","scroll"]},
			"x": {"type": "integer"},
			"y": {"type": "integer"},
			"text": {"type": "string"},
			"key": {"type": "string"},
			"dx": {"type": "integer"},
			"dy": {"type": "integer"}
		},
		"required": ["action"],
		"additionalProperties": false
	}`)
}

func (t *computerUseTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Action string `json:"action"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Text   string `json:"text"`
		Key    string `json:"key"`
		DX     int    `json:"dx"`
		DY     int    `json:"dy"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	switch args.Action {
	case "screenshot":
		img, err := t.c.Screenshot(ctx)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(img), nil
	case "left_click":
		if err := t.c.Click(ctx, args.X, args.Y, "left"); err != nil {
			return "", err
		}
		return "ok", nil
	case "right_click":
		if err := t.c.Click(ctx, args.X, args.Y, "right"); err != nil {
			return "", err
		}
		return "ok", nil
	case "type":
		if err := t.c.Type(ctx, args.Text); err != nil {
			return "", err
		}
		return "ok", nil
	case "key":
		if err := t.c.Key(ctx, args.Key); err != nil {
			return "", err
		}
		return "ok", nil
	case "mouse_move":
		if err := t.c.Move(ctx, args.X, args.Y); err != nil {
			return "", err
		}
		return "ok", nil
	case "scroll":
		if err := t.c.Scroll(ctx, args.X, args.Y, args.DX, args.DY); err != nil {
			return "", err
		}
		return "ok", nil
	default:
		return "", fmt.Errorf("unknown action: %s", args.Action)
	}
}
