package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// commandBound is the loose upper bound the compressed focus runs in this file
// must finish within. Without it, a command path that lost the tick and fell
// back to real minutes would sit until go test timed out rather than failing on
// the thing that broke.
const commandBound = 5 * time.Second

// invoke calls run with captured buffers and a background context, which is
// what every non-focus case needs.
func invoke(t *testing.T, dataPath string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	return invokeCtx(t, context.Background(), dataPath, time.Millisecond, args...)
}

// invokeCtx calls run with a caller-supplied context and tick, for the focus
// cases that need to cancel the interval or compress it.
//
// The call is bounded and the test fails as soon as the bound passes, rather
// than the elapsed time being checked once run has already returned. A focus
// branch that dropped the tick it was handed and fell back to a real 25 minutes
// would otherwise sit until go test's own timeout expired, reporting a
// timeout instead of the thing that broke.
func invokeCtx(t *testing.T, ctx context.Context, dataPath string, tick time.Duration, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var out, errOut bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- run(ctx, args, dataPath, tick, &out, &errOut)
	}()

	select {
	case code = <-done:
		return out.String(), errOut.String(), code
	case <-time.After(commandBound):
		cancel()
		t.Fatalf("run %v did not return within %v: the tick it was given did not reach the interval", args, commandBound)
		return "", "", 0
	}
}

// T011 — an invocation the system cannot act on is reported, not ignored
// (FR-015, SC-005).
func TestRunUnrecognizedInvocation(t *testing.T) {
	for name, args := range map[string][]string{
		"no arguments":            {},
		"unknown first argument":  {"list"},
		"unknown with an operand": {"delete", "1"},
	} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := invoke(t, filepath.Join(t.TempDir(), "tasks.json"), args...)

			if code != 1 {
				t.Errorf("exit status = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, usageMessage) {
				t.Errorf("stderr = %q, want it to contain the usage message %q", stderr, usageMessage)
			}
		})
	}
}

// T012 — the default data path is the pomotask/tasks.json entry under the
// user's configuration directory. This computes a path and compares it; it
// reads and writes nothing, so it stays clear of real task data on every
// platform the CI matrix runs (research.md R1).
func TestDefaultDataPath(t *testing.T) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("os.UserConfigDir is unavailable on this machine: %v", err)
	}
	want := filepath.Join(configDir, "pomotask", "tasks.json")

	got, err := defaultDataPath()
	if err != nil {
		t.Fatalf("defaultDataPath() returned an unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("defaultDataPath() = %q, want %q", got, want)
	}
}

// T049 — the production tick is one minute.
//
// This does not verify SC-004 and does not pretend to. SC-004 claims a real
// interval lasts 25 minutes to within a second, and compression removes the
// wall-clock property being claimed, which is why plan.md gives the criterion
// no seam and checks it in the manual quickstart run instead.
//
// What this catches is the regression that run would otherwise be the only
// defence against, and it only runs once: changing this constant to
// time.Second leaves every other test in the suite green — they all pass their
// own compressed tick — while the shipped interval lasts 25 seconds. Together
// with focus.Ticks, this is what SC-004 reduces to given a working timer
// (SC-004, FR-010; quickstart.md "SC-004, as measured").
func TestProductionTickIsOneMinute(t *testing.T) {
	if productionTick != time.Minute {
		t.Errorf("productionTick = %v, want %v", productionTick, time.Minute)
	}
}

// T017 — ADR 0001's behavioral check. It enters through run rather than calling
// the storage package, because a test that calls storage.Save proves the
// package writes JSON and proves nothing about whether the application reaches
// it (ADR 0001 Enforcement).
func TestAddWritesTheApprovedJSONDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	if _, _, code := invoke(t, path, "add", "write the report"); code != 0 {
		t.Fatalf("add exited %d, want 0", code)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the add path wrote no file at %s: %v", path, err)
	}

	var document struct {
		SchemaVersion *int `json:"schema_version"`
		Tasks         *[]struct {
			ID   int    `json:"id"`
			Text string `json:"text"`
			Done bool   `json:"done"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("the file the add path wrote does not decode through encoding/json: %v\n%s", err, data)
	}
	if document.SchemaVersion == nil {
		t.Error("the persisted document carries no schema_version")
	}
	if document.Tasks == nil {
		t.Fatal("the persisted document carries no tasks")
	}
	if len(*document.Tasks) != 1 || (*document.Tasks)[0].Text != "write the report" {
		t.Errorf("persisted tasks = %+v, want the one task that was added", *document.Tasks)
	}
}

// T018 — the add confirmation carries the assigned identifier. FR-003 makes
// this line the only occasion on which the identifier is disclosed, so an
// implementation that persisted correctly while printing nothing usable would
// pass every other test in this phase and still leave the user stuck.
func TestAddConfirmationCarriesTheIdentifier(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	stdout, stderr, code := invoke(t, path, "add", "write the report")

	if code != 0 {
		t.Fatalf("add exited %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	if want := "Added task 1: write the report\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

// T019 — the Independent Test for User Story 1, stated as code. Two separate
// run calls against one path, and the identifier advancing to 2 is what shows
// the first task was read back from disk rather than the file rewritten from
// empty (SC-002, FR-004, FR-006).
func TestAddPersistsAcrossInvocations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	if _, _, code := invoke(t, path, "add", "write the report"); code != 0 {
		t.Fatalf("first add exited %d, want 0", code)
	}

	stdout, stderr, code := invoke(t, path, "add", "review the draft")
	if code != 0 {
		t.Fatalf("second add exited %d, want 0 (stderr: %s)", code, stderr)
	}
	if want := "Added task 2: review the draft\n"; stdout != want {
		t.Errorf("second confirmation = %q, want %q", stdout, want)
	}
}

// T020 — a smoke bound on SC-001, not a benchmark. What it catches is a
// regression that changes the order of magnitude: a stray sleep, a retry loop,
// or a read that stops being proportional to a small file.
func TestAddReturnsWithinOneSecond(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	start := time.Now()
	_, _, code := invoke(t, path, "add", "write the report")
	elapsed := time.Since(start)

	if code != 0 {
		t.Fatalf("add exited %d, want 0", code)
	}
	if elapsed > time.Second {
		t.Errorf("add took %v against a warm temporary directory, want under one second", elapsed)
	}
}

// T021 — add takes exactly one argument, and the failure names the quoting form
// (FR-015; contracts/cli.md; Gap G4).
func TestAddArgumentCount(t *testing.T) {
	for name, args := range map[string][]string{
		"no argument":         {"add"},
		"two arguments":       {"add", "write", "the"},
		"unquoted multi-word": {"add", "write", "the", "report"},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.json")

			stdout, stderr, code := invoke(t, path, args...)

			if code != 1 {
				t.Errorf("exit status = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "exactly one argument") {
				t.Errorf("stderr = %q, want it to state that add takes exactly one argument", stderr)
			}
			if !strings.Contains(stderr, `pomotask add "write the report"`) {
				t.Errorf("stderr = %q, want it to show the quoting form; a generic usage line leaves the user to work it out", stderr)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("a rejected add recorded something at %s", path)
			}
		})
	}
}

// T021 — empty and whitespace-only text is rejected and nothing is recorded
// (FR-002).
func TestAddRejectsBlankText(t *testing.T) {
	for name, text := range map[string]string{"empty": "", "whitespace": "   "} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.json")

			stdout, stderr, code := invoke(t, path, "add", text)

			if code != 1 {
				t.Errorf("exit status = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr == "" {
				t.Error("stderr is empty, want a message naming what went wrong")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("a rejected add recorded something at %s", path)
			}
		})
	}
}

// T022 — an unrecognized schema version is reported on stderr with status 1 and
// the file is left exactly as found. T015 proves the storage package returns
// this error; this proves the command surfaces it on the right stream with the
// right status (contracts/cli.md Load failures; FR-015, SC-005).
func TestRunReportsUnreadableStoredData(t *testing.T) {
	for name, contents := range map[string]string{
		"unrecognized schema version": `{"schema_version": 2, "tasks": []}`,
		"truncated document":          `{"schema_version": 1, "tasks": [{"id": 1, "te`,
	} {
		for _, args := range [][]string{{"add", "write the report"}, {"done", "1"}} {
			t.Run(name+"/"+args[0], func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "tasks.json")
				if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
					t.Fatalf("writing the fixture: %v", err)
				}

				stdout, stderr, code := invoke(t, path, args...)

				if code != 1 {
					t.Errorf("exit status = %d, want 1", code)
				}
				if stdout != "" {
					t.Errorf("stdout = %q, want empty; a failure must not look like success", stdout)
				}
				if stderr == "" {
					t.Error("stderr is empty, want a message naming what went wrong")
				}

				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading the data file back: %v", err)
				}
				if string(after) != contents {
					t.Errorf("the data file changed:\n before: %q\n  after: %q", contents, after)
				}
			})
		}
	}
}

// T022 — a destination that cannot be written is reported rather than reported
// as success. A directory serves as the unwritable destination: renaming a file
// onto a directory fails on every platform. Every path touched stays inside the
// temporary directory (Constitution isolated temp locations; FR-015, SC-005).
func TestRunReportsAnUnwritableDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("preparing the unwritable destination: %v", err)
	}

	stdout, stderr, code := invoke(t, path, "add", "write the report")

	if code != 1 {
		t.Errorf("exit status = %d, want 1", code)
	}
	if strings.Contains(stdout, "Added task") {
		t.Errorf("stdout = %q, want no success confirmation for a write that failed", stdout)
	}
	if stderr == "" {
		t.Error("stderr is empty, want a message naming what went wrong")
	}
}

// T027 — the done success case run as a first-time-user journey. It starts from
// an empty directory with no data file and no setup call of any kind, which is
// what makes it cover SC-006: a seeded fixture would hide any initialization
// step the tool secretly required (FR-005, FR-009, SC-003, SC-006).
func TestDoneFromAnEmptyDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	if _, stderr, code := invoke(t, path, "add", "write the report"); code != 0 {
		t.Fatalf("add exited %d, want 0 (stderr: %s)", code, stderr)
	}

	stdout, stderr, code := invoke(t, path, "done", "1")
	if code != 0 {
		t.Fatalf("done exited %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	if want := "Completed task 1\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}

	assertStoredDone(t, path, 1, true)
}

// T028 — marking an already-complete task reports on stdout with status 0 and
// leaves the stored data byte-identical (FR-008, FR-015; Gap G1).
func TestDoneOnAnAlreadyCompleteTask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	if _, _, code := invoke(t, path, "add", "write the report"); code != 0 {
		t.Fatalf("add exited %d, want 0", code)
	}
	if _, _, code := invoke(t, path, "done", "1"); code != 0 {
		t.Fatalf("first done exited %d, want 0", code)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the data file: %v", err)
	}

	stdout, stderr, code := invoke(t, path, "done", "1")

	if code != 0 {
		t.Errorf("exit status = %d, want 0: the state the user asked for holds", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty: this is not a failure", stderr)
	}
	if want := "Task 1 is already complete\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the data file back: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("the data file changed:\n before: %q\n  after: %q", before, after)
	}
}

// T029 — done's failure cases: an identifier matching no task, a non-integer
// argument, and the wrong argument count. Each reports on stderr with status 1
// and changes nothing (FR-007, FR-015, SC-005).
func TestDoneFailures(t *testing.T) {
	tests := map[string]struct {
		args []string
		want string
	}{
		"no such task":     {args: []string{"done", "99"}, want: "99"},
		"non-integer":      {args: []string{"done", "one"}, want: "one"},
		"no argument":      {args: []string{"done"}, want: "exactly one argument"},
		"two arguments":    {args: []string{"done", "1", "2"}, want: "exactly one argument"},
		"empty identifier": {args: []string{"done", ""}, want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.json")
			if _, _, code := invoke(t, path, "add", "write the report"); code != 0 {
				t.Fatalf("add exited %d, want 0", code)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the data file: %v", err)
			}

			stdout, stderr, code := invoke(t, path, tc.args...)

			if code != 1 {
				t.Errorf("exit status = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr == "" {
				t.Error("stderr is empty, want a message naming what went wrong")
			}
			if tc.want != "" && !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr = %q, want it to name %q", stderr, tc.want)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the data file back: %v", err)
			}
			if string(after) != string(before) {
				t.Errorf("the data file changed:\n before: %q\n  after: %q", before, after)
			}
		})
	}
}

// T035 — a completed interval writes the full sequence to the stdout writer run
// was given and returns 0. T032 proves the focus package emits the sequence;
// this proves the command delivers it (FR-012, SC-007, SC-008).
func TestFocusCompletionAtTheCommandLevel(t *testing.T) {
	stdout, stderr, code := invokeCtx(t, context.Background(), filepath.Join(t.TempDir(), "tasks.json"), time.Millisecond, "focus")

	if code != 0 {
		t.Errorf("exit status = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	lines := splitLines(stdout)
	if len(lines) != 26 {
		t.Fatalf("got %d lines on stdout, want 26:\n%s", len(lines), stdout)
	}
	if want := "25 minutes remaining"; lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
	if want := "1 minute remaining"; lines[24] != want {
		t.Errorf("line 25 = %q, want %q", lines[24], want)
	}
	if want := "Focus interval complete."; lines[25] != want {
		t.Errorf("final line = %q, want %q", lines[25], want)
	}
	for i, line := range lines[:25] {
		if strings.HasPrefix(line, "0 ") {
			t.Errorf("line %d = %q, want no report showing zero remaining", i+1, line)
		}
	}
}

// T035 — an interrupted interval returns 1 specifically, not merely non-zero,
// with its message on stdout. Cancellation comes through the context the test
// itself passed to run; no signal is delivered (FR-013, SC-008; Gaps G3, G5).
func TestFocusInterruptionAtTheCommandLevel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	stdout, stderr, code := invokeCtx(t, ctx, filepath.Join(t.TempDir(), "tasks.json"), 5*time.Millisecond, "focus")

	if code != 1 {
		t.Errorf("exit status = %d, want 1 specifically", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty: an interruption is a deliberate user action, not a fault", stderr)
	}

	stopped := regexp.MustCompile(`(?m)^Focus interval stopped after \d+ minutes?\.$`)
	if !stopped.MatchString(stdout) {
		t.Errorf("stdout = %q, want it to carry the elapsed figure in whole minutes", stdout)
	}
	if strings.Contains(stdout, "Focus interval complete.") {
		t.Errorf("stdout = %q, want no completion message for an interrupted interval", stdout)
	}
}

// T036 — focus rejects any argument rather than ignoring it. An ignored
// argument produces output identical to no argument and would prove nothing
// (FR-011, FR-015; contracts/cli.md).
func TestFocusRejectsArguments(t *testing.T) {
	for name, args := range map[string][]string{
		"a bare number": {"focus", "5"},
		"a flag":        {"focus", "--minutes", "5"},
		"anything else": {"focus", "please"},
	} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := invoke(t, filepath.Join(t.TempDir(), "tasks.json"), args...)

			if code != 1 {
				t.Errorf("exit status = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty: the interval must not have run", stdout)
			}
			if !strings.Contains(stderr, "takes no arguments") {
				t.Errorf("stderr = %q, want it to state that focus takes no arguments", stderr)
			}
			if !strings.Contains(stderr, "fixed") {
				t.Errorf("stderr = %q, want it to state that the duration is fixed", stderr)
			}
		})
	}
}

// assertStoredDone decodes the data file and checks the completion state of one
// task, without going back through the command under test.
func assertStoredDone(t *testing.T, path string, id int, want bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the data file: %v", err)
	}

	var document struct {
		Tasks []struct {
			ID   int  `json:"id"`
			Done bool `json:"done"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decoding the data file: %v", err)
	}
	for _, task := range document.Tasks {
		if task.ID == id {
			if task.Done != want {
				t.Errorf("task %d done = %v, want %v", id, task.Done, want)
			}
			return
		}
	}
	t.Errorf("no task carries identifier %d in %s", id, data)
}

func splitLines(s string) []string {
	trimmed := strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
