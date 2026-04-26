package tools

import (
	"io"
	"os/exec"
	"sync"
)

type bgJob struct {
	ID         string
	Cmd        *exec.Cmd
	StdoutPipe io.ReadCloser
	StderrPipe io.ReadCloser
	Buffer     []byte
	BufMu      sync.Mutex
	Done       bool
	ExitCode   int
	Err        error

	readOff int
}

var (
	bgJobsMu sync.Mutex
	bgJobs   = map[string]*bgJob{}
)

func registerJob(j *bgJob) {
	bgJobsMu.Lock()
	defer bgJobsMu.Unlock()
	bgJobs[j.ID] = j
}

func lookupJob(id string) (*bgJob, bool) {
	bgJobsMu.Lock()
	defer bgJobsMu.Unlock()
	j, ok := bgJobs[id]
	return j, ok
}

func removeJob(id string) {
	bgJobsMu.Lock()
	defer bgJobsMu.Unlock()
	delete(bgJobs, id)
}

func appendOutput(j *bgJob, p []byte) {
	j.BufMu.Lock()
	defer j.BufMu.Unlock()
	j.Buffer = append(j.Buffer, p...)
}

func drainNew(j *bgJob) []byte {
	j.BufMu.Lock()
	defer j.BufMu.Unlock()
	if j.readOff >= len(j.Buffer) {
		return nil
	}
	out := append([]byte(nil), j.Buffer[j.readOff:]...)
	j.readOff = len(j.Buffer)
	return out
}

func killJob(j *bgJob) error {
	if j.Cmd == nil || j.Cmd.Process == nil {
		return nil
	}
	return j.Cmd.Process.Kill()
}
