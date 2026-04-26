package chat

import (
	"fmt"

	"claudecode/internal/core"
)

// Branch returns a copy of the session history truncated at atIndex (exclusive).
// atIndex must satisfy 0 <= atIndex <= len(history); otherwise an error is returned.
// The caller decides whether to apply the truncation (typically via
// session.ResetHistory + Append).
func Branch(s core.Session, atIndex int) ([]core.Message, error) {
	hist := s.History()
	if atIndex < 0 || atIndex > len(hist) {
		return nil, fmt.Errorf("branch: index %d out of range [0,%d]", atIndex, len(hist))
	}
	out := make([]core.Message, atIndex)
	copy(out, hist[:atIndex])
	return out, nil
}
