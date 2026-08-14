package task

import (
	"errors"
	"testing"
)

// T013 — Add assigns an identifier one greater than the largest present,
// starting at 1 (FR-006; research.md R3).
func TestAddAssignsIdentifiers(t *testing.T) {
	tests := map[string]struct {
		existing []Task
		want     int
	}{
		"first task starts at 1":  {existing: nil, want: 1},
		"empty slice starts at 1": {existing: []Task{}, want: 1},
		"one greater than the largest": {
			existing: []Task{{ID: 1, Text: "a"}, {ID: 2, Text: "b"}},
			want:     3,
		},
		"largest need not be last": {
			existing: []Task{{ID: 7, Text: "a"}, {ID: 2, Text: "b"}},
			want:     8,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, added, err := Add(tc.existing, "write the report")
			if err != nil {
				t.Fatalf("Add returned an unexpected error: %v", err)
			}
			if added.ID != tc.want {
				t.Errorf("assigned ID = %d, want %d", added.ID, tc.want)
			}
			if added.Text != "write the report" {
				t.Errorf("Text = %q, want %q", added.Text, "write the report")
			}
			if added.Done {
				t.Error("a newly added task is complete, want incomplete")
			}
		})
	}
}

// T013 — empty and whitespace-only text is rejected and nothing is recorded
// (FR-002).
func TestAddRejectsBlankText(t *testing.T) {
	for name, text := range map[string]string{
		"empty":   "",
		"spaces":  "   ",
		"tab":     "\t",
		"newline": "\n",
		"mixed":   " \t\n ",
	} {
		t.Run(name, func(t *testing.T) {
			existing := []Task{{ID: 1, Text: "a"}}

			got, _, err := Add(existing, text)
			if !errors.Is(err, ErrBlankText) {
				t.Fatalf("Add error = %v, want ErrBlankText", err)
			}
			if got != nil {
				t.Errorf("Add returned %v on rejection, want nil", got)
			}
			if len(existing) != 1 || existing[0].Text != "a" {
				t.Errorf("existing tasks were modified: %v", existing)
			}
		})
	}
}

// T013 — Add leaves the tasks it was given untouched (FR-001).
func TestAddLeavesExistingTasksUntouched(t *testing.T) {
	existing := []Task{{ID: 1, Text: "a", Done: true}, {ID: 2, Text: "b"}}

	got, _, err := Add(existing, "c")
	if err != nil {
		t.Fatalf("Add returned an unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Add returned %d tasks, want 3", len(got))
	}
	for i, want := range existing {
		if got[i] != want {
			t.Errorf("task %d = %+v, want %+v", i, got[i], want)
		}
	}

	// Mutating the returned collection must not reach back into the original.
	got[0].Text = "mutated"
	if existing[0].Text != "a" {
		t.Errorf("the original collection shares storage with the returned one: %v", existing)
	}
}

// T013 — two tasks carrying identical text receive different identifiers, which
// is what keeps them separately addressable (spec duplicate-text edge case).
func TestAddGivesDuplicateTextDistinctIdentifiers(t *testing.T) {
	tasks, first, err := Add(nil, "write the report")
	if err != nil {
		t.Fatalf("first Add returned an unexpected error: %v", err)
	}
	_, second, err := Add(tasks, "write the report")
	if err != nil {
		t.Fatalf("second Add returned an unexpected error: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("both tasks were assigned ID %d, want distinct identifiers", first.ID)
	}
	if first.Text != second.Text {
		t.Errorf("text differs: %q and %q, want them identical for this case", first.Text, second.Text)
	}
}

// T026 — Complete reports three outcomes and they stay distinguishable
// (FR-005, FR-007, FR-008).
func TestCompleteOutcomes(t *testing.T) {
	tests := map[string]struct {
		tasks []Task
		id    int
		want  Outcome
	}{
		"changed":          {tasks: []Task{{ID: 1, Text: "a"}}, id: 1, want: Changed},
		"already complete": {tasks: []Task{{ID: 1, Text: "a", Done: true}}, id: 1, want: AlreadyComplete},
		"no such task":     {tasks: []Task{{ID: 1, Text: "a"}}, id: 99, want: NotFound},
		"no tasks at all":  {tasks: nil, id: 1, want: NotFound},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, outcome := Complete(tc.tasks, tc.id)
			if outcome != tc.want {
				t.Fatalf("outcome = %v, want %v", outcome, tc.want)
			}

			switch tc.want {
			case Changed:
				if !got[0].Done {
					t.Error("the task was not marked complete")
				}
				if tc.tasks[0].Done {
					t.Error("Complete modified the collection it was given")
				}
			case AlreadyComplete, NotFound:
				if len(got) != len(tc.tasks) {
					t.Errorf("collection length changed from %d to %d", len(tc.tasks), len(got))
				}
				for i := range tc.tasks {
					if got[i] != tc.tasks[i] {
						t.Errorf("task %d changed from %+v to %+v", i, tc.tasks[i], got[i])
					}
				}
			}
		})
	}
}

// T026 — the three outcomes are three distinct values, which is what stops the
// caller collapsing "already complete" into "not found" or into success.
func TestOutcomesAreDistinct(t *testing.T) {
	seen := map[Outcome]string{}
	for _, o := range []Outcome{Changed, AlreadyComplete, NotFound} {
		if other, ok := seen[o]; ok {
			t.Errorf("%s and %s are the same value", other, o)
		}
		seen[o] = o.String()
	}
	if len(seen) != 3 {
		t.Errorf("got %d distinct outcomes, want 3", len(seen))
	}
}

// T026 — Complete touches only the task named (FR-005).
func TestCompleteTouchesOnlyTheNamedTask(t *testing.T) {
	tasks := []Task{{ID: 1, Text: "a"}, {ID: 2, Text: "b"}, {ID: 3, Text: "c"}}

	got, outcome := Complete(tasks, 2)
	if outcome != Changed {
		t.Fatalf("outcome = %v, want Changed", outcome)
	}
	if got[0].Done || got[2].Done {
		t.Errorf("a task other than 2 was marked complete: %v", got)
	}
	if !got[1].Done {
		t.Error("task 2 was not marked complete")
	}
}
