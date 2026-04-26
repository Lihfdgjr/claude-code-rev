package sessions

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RecoveryCandidate describes a session eligible for resumption.
type RecoveryCandidate struct {
	ID            string
	Title         string
	Summary       string
	Modified      time.Time
	MessageCount  int
	HasTranscript bool
}

// Recover scans persisted sessions and returns recent candidates for resume.
// Only sessions modified in the last 24 hours with at least one message are
// returned. transcriptDir is checked for a `<id>.jsonl` companion file.
func (s *Store) Recover(transcriptDir string) ([]*RecoveryCandidate, error) {
	metas, err := s.List()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	out := make([]*RecoveryCandidate, 0, len(metas))
	for _, m := range metas {
		if m == nil {
			continue
		}
		if m.MessageCount < 1 {
			continue
		}
		if m.LastModified.Before(cutoff) {
			continue
		}
		hasTr := false
		if transcriptDir != "" {
			tp := filepath.Join(transcriptDir, m.ID+".jsonl")
			if st, err := os.Stat(tp); err == nil && !st.IsDir() {
				hasTr = true
			}
		}
		out = append(out, &RecoveryCandidate{
			ID:            m.ID,
			Title:         m.Summary,
			Summary:       m.Summary,
			Modified:      m.LastModified,
			MessageCount:  m.MessageCount,
			HasTranscript: hasTr,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Modified.After(out[j].Modified)
	})
	return out, nil
}
