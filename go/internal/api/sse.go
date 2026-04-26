package api

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// sseEvent is a single decoded SSE record before JSON parsing.
type sseEvent struct {
	Name string
	Data string
}

// sseReader parses a text/event-stream into discrete events. Records are
// separated by a blank line; a record can have multiple `data:` lines which
// are joined with newlines per the spec.
type sseReader struct {
	r *bufio.Reader
}

func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{r: bufio.NewReaderSize(r, 64*1024)}
}

func (s *sseReader) Next() (sseEvent, error) {
	var ev sseEvent
	var data strings.Builder
	hasData := false

	for {
		line, err := s.r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && (hasData || ev.Name != "" || len(line) > 0) {
				if line != "" {
					processSSELine(line, &ev, &data, &hasData)
				}
				if hasData {
					ev.Data = data.String()
					return ev, nil
				}
			}
			return sseEvent{}, err
		}
		// Trim trailing \r\n or \n.
		line = strings.TrimRight(line, "\r\n")

		// Blank line dispatches the event.
		if line == "" {
			if hasData || ev.Name != "" {
				ev.Data = data.String()
				return ev, nil
			}
			continue
		}

		processSSELine(line+"\n", &ev, &data, &hasData)
	}
}

func processSSELine(line string, ev *sseEvent, data *strings.Builder, hasData *bool) {
	// Strip the trailing newline added back for uniformity.
	line = strings.TrimRight(line, "\r\n")
	if line == "" || strings.HasPrefix(line, ":") {
		return
	}
	field, value, ok := splitField(line)
	if !ok {
		return
	}
	switch field {
	case "event":
		ev.Name = value
	case "data":
		if *hasData {
			data.WriteByte('\n')
		}
		data.WriteString(value)
		*hasData = true
	}
}

func splitField(line string) (string, string, bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, "", true
	}
	field := line[:idx]
	value := line[idx+1:]
	// Per spec, a single leading space is stripped.
	value = strings.TrimPrefix(value, " ")
	return field, value, true
}
