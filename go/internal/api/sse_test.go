package api

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSSEReaderParsesBasicEvents(t *testing.T) {
	stream := "event: message_start\ndata: {\"type\":\"message_start\"}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0}\n\n"
	r := newSSEReader(strings.NewReader(stream))

	ev, err := r.Next()
	if err != nil {
		t.Fatalf("first Next: %v", err)
	}
	if ev.Name != "message_start" {
		t.Errorf("first event name = %q", ev.Name)
	}
	if ev.Data != `{"type":"message_start"}` {
		t.Errorf("first event data = %q", ev.Data)
	}

	ev, err = r.Next()
	if err != nil {
		t.Fatalf("second Next: %v", err)
	}
	if ev.Name != "content_block_delta" {
		t.Errorf("second event name = %q", ev.Name)
	}
	if ev.Data != `{"type":"content_block_delta","index":0}` {
		t.Errorf("second event data = %q", ev.Data)
	}

	_, err = r.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestSSEReaderJoinsMultipleDataLines(t *testing.T) {
	stream := "event: e\ndata: a\ndata: b\n\n"
	r := newSSEReader(strings.NewReader(stream))
	ev, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if ev.Data != "a\nb" {
		t.Errorf("data = %q, want %q", ev.Data, "a\nb")
	}
}

func TestSSEReaderIgnoresCommentsAndCRLF(t *testing.T) {
	stream := ": comment\r\nevent: ping\r\ndata: x\r\n\r\n"
	r := newSSEReader(strings.NewReader(stream))
	ev, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if ev.Name != "ping" {
		t.Errorf("name = %q", ev.Name)
	}
	if ev.Data != "x" {
		t.Errorf("data = %q", ev.Data)
	}
}

func TestSSEReaderStripsLeadingSpaceFromValue(t *testing.T) {
	stream := "event: hello\ndata:no-space\n\n"
	r := newSSEReader(strings.NewReader(stream))
	ev, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if ev.Data != "no-space" {
		t.Errorf("data = %q", ev.Data)
	}
}
