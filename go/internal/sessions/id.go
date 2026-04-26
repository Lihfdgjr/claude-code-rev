package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID returns a sortable session id like 2026-04-25T22-30-15-<rand6>.
func NewID() string {
	now := time.Now().UTC()
	var buf [3]byte
	_, _ = rand.Read(buf[:])
	return fmt.Sprintf("%s-%s",
		now.Format("2006-01-02T15-04-05"),
		hex.EncodeToString(buf[:]),
	)
}
