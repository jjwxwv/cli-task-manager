package task

import (
	"errors"
	"strings"
)

// ErrBlankText is returned by Add when the supplied text is empty or consists
// only of whitespace. FR-002 requires such an attempt be rejected with a stated
// reason and nothing recorded.
var ErrBlankText = errors.New("task text must not be empty")

// Task is a single unit of work the user has recorded.
//
// ID is assigned by the system, is unique across recorded tasks, and never
// changes once disclosed, because the confirmation printed when a task is added
// is the only occasion on which the user learns it (FR-003, FR-006). Text is
// the descriptive text the user supplied, and Done reports whether the task has
// been marked complete.
type Task struct {
	ID   int
	Text string
	Done bool
}

// Outcome distinguishes the three results Complete can produce. FR-007 and
// FR-008 require different messages for a missing task and an already-complete
// one, and neither may be reported as success, so a boolean would collapse two
// results that must stay apart.
type Outcome int

const (
	// Changed reports that the named task was incomplete and is now complete.
	Changed Outcome = iota
	// AlreadyComplete reports that the named task was already complete and
	// nothing was changed (FR-008).
	AlreadyComplete
	// NotFound reports that no task carries the named identifier and nothing
	// was changed (FR-007).
	NotFound
)

// Outcome carries no String method. Rendering an outcome as text is something
// only a failing test ever needed, and Principle V admits no surface that
// traces to neither a specification requirement nor an Accepted ADR. The
// formatting lives in the test that wants it (T051).

// Add records a new task carrying text and returns the extended collection
// along with the task as recorded. The identifier assigned is one greater than
// the largest present, or 1 when no tasks exist; that is safe against reuse
// only because nothing in scope removes a task (research.md R3).
//
// Text that is empty or whitespace-only is rejected with ErrBlankText and
// nothing is recorded (FR-002). The collection passed in is never modified: the
// returned collection is a fresh one, so a caller that fails to save leaves the
// tasks it loaded exactly as they were.
func Add(tasks []Task, text string) ([]Task, Task, error) {
	if strings.TrimSpace(text) == "" {
		return nil, Task{}, ErrBlankText
	}

	added := Task{ID: nextID(tasks), Text: text}

	extended := make([]Task, len(tasks), len(tasks)+1)
	copy(extended, tasks)
	return append(extended, added), added, nil
}

// Complete marks the task carrying id as complete and returns the resulting
// collection along with the outcome. The collection passed in is never
// modified. When the outcome is anything other than Changed the returned
// collection is equal to the one supplied, which is what lets a caller skip
// the save entirely and leave stored data byte-identical (FR-005, FR-007,
// FR-008).
func Complete(tasks []Task, id int) ([]Task, Outcome) {
	for i, t := range tasks {
		if t.ID != id {
			continue
		}
		if t.Done {
			return tasks, AlreadyComplete
		}

		updated := make([]Task, len(tasks))
		copy(updated, tasks)
		updated[i].Done = true
		return updated, Changed
	}
	return tasks, NotFound
}

// nextID reports one greater than the largest identifier present, or 1 when the
// collection is empty.
func nextID(tasks []Task) int {
	largest := 0
	for _, t := range tasks {
		if t.ID > largest {
			largest = t.ID
		}
	}
	return largest + 1
}
