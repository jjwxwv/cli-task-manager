// Command pomotask records tasks, marks them complete, and runs a fixed
// 25-minute focus interval.
//
// main is a one-line wrapper around cli, which holds the process concerns, so
// that run — where every dispatch, message, and exit status lives — can be
// called directly by a test with a temporary data path, a compressed tick, a
// cancellable context, and captured buffers.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"pomotask/internal/focus"
	"pomotask/internal/storage"
	"pomotask/internal/task"
)

// usageMessage is printed when pomotask is invoked with no arguments or with an
// unrecognized first argument (FR-015; contracts/cli.md).
const usageMessage = "usage: pomotask <add|done|focus> [argument]"

// addArgumentMessage is printed when add is given anything other than exactly
// one argument. It names the quoting form rather than repeating the generic
// usage line, because the mistake it corrects is almost always an unquoted
// multi-word argument (contracts/cli.md; Gap G4).
const addArgumentMessage = `pomotask add takes exactly one argument; quote text containing spaces:
    pomotask add "write the report"`

// doneArgumentMessage is printed when done is given anything other than exactly
// one argument (FR-015; contracts/cli.md).
const doneArgumentMessage = `pomotask done takes exactly one argument, the identifier printed when the task was added:
    pomotask done 3`

// focusArgumentMessage is printed when focus is given any argument at all.
// Arguments are rejected rather than ignored: an ignored argument would leave a
// user unable to tell a refused option from an accepted one that did nothing,
// which is the state FR-011 exists to prevent (FR-011, FR-015).
const focusArgumentMessage = "pomotask focus takes no arguments; the 25-minute duration is fixed"

// productionTick is the tick the interval runs at outside tests. The interval's
// total length is derived as 25 ticks, so this is the single source FR-016
// requires; nothing user-facing can reach it (FR-011, FR-016).
const productionTick = time.Minute

func main() {
	os.Exit(cli())
}

// report writes one formatted line to w.
//
// This is the single place in the command where a write error is dropped, and
// it is dropped deliberately: w is the process's own stdout or stderr, so a
// command that cannot write its output has no second channel on which to say
// so, and no exit status it returned would reach a user who is not receiving
// its output either. Confining the drop to one documented function is what
// keeps it from being repeated, unexplained, at every call site.
//
// internal/focus does propagate its write errors, because there the failure is
// actionable — it ends an interval that would otherwise run on reporting
// nothing — and runFocus surfaces it here.
func report(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

// cli holds the concerns that belong to the process rather than to any command:
// resolving the default data path, wiring the interrupt signal to a context,
// and choosing the production tick. It returns the exit status instead of
// calling os.Exit itself, so that its deferred stop function actually runs —
// os.Exit runs no deferred function.
func cli() int {
	dataPath, err := defaultDataPath()
	if err != nil {
		report(os.Stderr, "%v", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return run(ctx, os.Args[1:], dataPath, productionTick, os.Stdout, os.Stderr)
}

// defaultDataPath reports where task data lives when no other path is supplied:
// the pomotask/tasks.json entry under the user's configuration directory. It
// resolves a path and touches no file, so a test may call it without reaching
// real task data (research.md R1).
func defaultDataPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving the user configuration directory: %w", err)
	}
	return filepath.Join(dir, "pomotask", "tasks.json"), nil
}

// run executes one pomotask invocation and reports the process exit status.
// args excludes the program name. dataPath names the task data file; tick is
// the focus interval's tick, from which its total length is derived. Normal
// output is written to stdout and failure messages to stderr, except the focus
// interruption message, which contracts/cli.md routes to stdout. ctx cancels a
// running focus interval and is ignored by every other command; run never
// learns why cancellation happened.
func run(ctx context.Context, args []string, dataPath string, tick time.Duration, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		report(stderr, "%s", usageMessage)
		return 1
	}

	switch args[0] {
	case "add":
		return runAdd(args[1:], dataPath, stdout, stderr)
	case "done":
		return runDone(args[1:], dataPath, stdout, stderr)
	case "focus":
		return runFocus(ctx, args[1:], tick, stdout, stderr)
	default:
		report(stderr, "%s", usageMessage)
		return 1
	}
}

// runAdd records a new task and prints the identifier assigned to it, which
// FR-003 makes the only occasion on which the system discloses that identifier.
// Every failure prints to stderr and records nothing (FR-001 through FR-004,
// FR-015).
func runAdd(args []string, dataPath string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		report(stderr, "%s", addArgumentMessage)
		return 1
	}

	tasks, err := storage.Load(dataPath)
	if err != nil {
		report(stderr, "%v", err)
		return 1
	}

	tasks, added, err := task.Add(tasks, args[0])
	if err != nil {
		report(stderr, "%v", err)
		return 1
	}

	if err := storage.Save(dataPath, tasks); err != nil {
		report(stderr, "%v", err)
		return 1
	}

	report(stdout, "Added task %d: %s", added.ID, added.Text)
	return 0
}

// runDone marks a task complete, mapping each of Complete's three outcomes to
// the stream, message, and exit status contracts/cli.md fixes. An already
// complete task is not a failure — the state the user asked for holds — and no
// save occurs on that path, so the stored data is left byte-identical
// (FR-005 through FR-009, FR-015).
func runDone(args []string, dataPath string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		report(stderr, "%s", doneArgumentMessage)
		return 1
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		report(stderr, "%q is not a task identifier; pomotask done takes the number printed when the task was added", args[0])
		return 1
	}

	tasks, err := storage.Load(dataPath)
	if err != nil {
		report(stderr, "%v", err)
		return 1
	}

	updated, outcome := task.Complete(tasks, id)
	switch outcome {
	case task.NotFound:
		report(stderr, "no task carries identifier %d", id)
		return 1
	case task.AlreadyComplete:
		report(stdout, "Task %d is already complete", id)
		return 0
	}

	if err := storage.Save(dataPath, updated); err != nil {
		report(stderr, "%v", err)
		return 1
	}

	report(stdout, "Completed task %d", id)
	return 0
}

// runFocus runs the focus interval, forwarding the context and tick run was
// given and writing the interval's output to the stdout writer run was given
// rather than to os.Stdout, which a test could not observe. It creates no
// context of its own and cannot tell an interrupt from any other cancellation
// (FR-010 through FR-013; research.md R7).
func runFocus(ctx context.Context, args []string, tick time.Duration, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		report(stderr, "%s", focusArgumentMessage)
		return 1
	}

	completed, err := focus.Run(ctx, tick, stdout)
	if err != nil {
		report(stderr, "%v", err)
		return 1
	}
	if !completed {
		// An interrupted interval exits non-zero so that SC-008 can tell it
		// from a completed one. Its message went to stdout, which is the one
		// non-zero exit in this contract that does not report on stderr
		// (Gaps G3 and G5).
		return 1
	}
	return 0
}
