package tools

import "sync"

type taskEntry struct {
	ID          int    `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description,omitempty"`
	ActiveForm  string `json:"activeForm,omitempty"`
	Status      string `json:"status"`
}

var (
	taskMu     sync.Mutex
	nextTaskID = 1
	taskList   []*taskEntry
)

func addTask(subject, description, activeForm string) *taskEntry {
	taskMu.Lock()
	defer taskMu.Unlock()
	t := &taskEntry{
		ID:          nextTaskID,
		Subject:     subject,
		Description: description,
		ActiveForm:  activeForm,
		Status:      "pending",
	}
	nextTaskID++
	taskList = append(taskList, t)
	return cloneTask(t)
}

func findTask(id int) *taskEntry {
	taskMu.Lock()
	defer taskMu.Unlock()
	for _, t := range taskList {
		if t.ID == id {
			return cloneTask(t)
		}
	}
	return nil
}

func updateTask(id int, mutate func(*taskEntry)) *taskEntry {
	taskMu.Lock()
	defer taskMu.Unlock()
	for _, t := range taskList {
		if t.ID == id {
			mutate(t)
			return cloneTask(t)
		}
	}
	return nil
}

func removeTask(id int) bool {
	taskMu.Lock()
	defer taskMu.Unlock()
	for i, t := range taskList {
		if t.ID == id {
			taskList = append(taskList[:i], taskList[i+1:]...)
			return true
		}
	}
	return false
}

func listTasks() []*taskEntry {
	taskMu.Lock()
	defer taskMu.Unlock()
	out := make([]*taskEntry, 0, len(taskList))
	for _, t := range taskList {
		out = append(out, cloneTask(t))
	}
	return out
}

func cloneTask(t *taskEntry) *taskEntry {
	c := *t
	return &c
}
