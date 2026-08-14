package focus_test

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"pomotask/internal/focus"
)

// testBound is the loose upper bound every compressed run in this file must
// finish within. It is an order-of-magnitude guard against a run that ignored
// its tick and fell back to real minutes, not a timing claim.
const testBound = 5 * time.Second

var remainingLine = regexp.MustCompile(`^(\d+) minutes? remaining$`)

// runBounded calls focus.Run and fails the test as soon as the bound passes,
// rather than measuring the elapsed time once the call has already returned.
// The difference matters: a run whose total length stopped deriving from its
// tick would sit for 25 real minutes, and a check made afterwards would report
// nothing until go test's own timeout expired on it.
func runBounded(t *testing.T, ctx context.Context, tick time.Duration, out io.Writer) (completed bool, elapsed time.Duration) {
	t.Helper()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type outcome struct {
		completed bool
		err       error
	}
	done := make(chan outcome, 1)

	start := time.Now()
	go func() {
		got, err := focus.Run(ctx, tick, out)
		done <- outcome{got, err}
	}()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("focus.Run at a %v tick returned an unexpected error: %v", tick, got.err)
		}
		return got.completed, time.Since(start)
	case <-time.After(testBound):
		cancel()
		t.Fatalf("focus.Run at a %v tick did not return within %v: its total length is not derived from the tick", tick, testBound)
		return false, 0
	}
}

// T049 — the interval is twenty-five ticks long.
//
// Every other test in this file is written against focus.Ticks rather than
// against the literal 25, which is what lets them stay correct if the constant
// moves — and is exactly why none of them would notice it moving. FR-010 fixes
// the count at 25 and SC-007 fixes the report sequence to match, so the
// constant is pinned here, once, against the requirement rather than against
// itself (FR-010, SC-007).
func TestIntervalIsTwentyFiveTicks(t *testing.T) {
	if focus.Ticks != 25 {
		t.Errorf("focus.Ticks = %d, want 25", focus.Ticks)
	}
}

// T032 — a completed interval emits exactly 25 remaining-time reports, the
// first showing 25 and the last showing 1, none showing zero, followed by
// exactly one completion line (SC-007, FR-012).
func TestRunEmitsTheFullSequence(t *testing.T) {
	var out bytes.Buffer

	completed, _ := runBounded(t, context.Background(), time.Millisecond, &out)
	if !completed {
		t.Fatal("Run reported the interval as incomplete, want completed")
	}

	lines := splitLines(out.String())
	if len(lines) != focus.Ticks+1 {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), focus.Ticks+1, out.String())
	}

	for i, line := range lines[:focus.Ticks] {
		match := remainingLine.FindStringSubmatch(line)
		if match == nil {
			t.Fatalf("line %d = %q, want a remaining-time report", i+1, line)
		}
		remaining, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatalf("line %d = %q: %v", i+1, line, err)
		}
		if want := focus.Ticks - i; remaining != want {
			t.Errorf("line %d reports %d remaining, want %d", i+1, remaining, want)
		}
		if remaining == 0 {
			t.Errorf("line %d reports zero remaining; no report may show zero", i+1)
		}
	}

	last := lines[focus.Ticks]
	if remainingLine.MatchString(last) {
		t.Errorf("the final line %q is a remaining-time report, want the completion message", last)
	}
	if !strings.Contains(strings.ToLower(last), "complete") {
		t.Errorf("the final line = %q, want it to announce that the interval ended", last)
	}
}

// T033 — the same sequence at two different compressed ticks, each run
// finishing well inside the bound. Identical sequences show the report count
// does not depend on the tick; finishing inside the bound shows the total
// length does (FR-016; research.md R6).
//
// No ratio is asserted between the two elapsed times. Windows default timer
// granularity is around 15.6ms, so both ticks round up to about the same real
// interval and a ratio would measure the operating system's scheduler.
func TestRunIsDrivenEntirelyByItsTick(t *testing.T) {
	sequences := make([]string, 0, 2)

	for _, tick := range []time.Duration{time.Millisecond, 5 * time.Millisecond} {
		var out bytes.Buffer

		completed, _ := runBounded(t, context.Background(), tick, &out)
		if !completed {
			t.Fatalf("Run at a %v tick reported the interval as incomplete", tick)
		}
		sequences = append(sequences, out.String())
	}

	if sequences[0] != sequences[1] {
		t.Errorf("the two ticks produced different output:\n1ms:\n%s\n5ms:\n%s", sequences[0], sequences[1])
	}
}

// T034 — a cancelled interval reports the elapsed figure in whole minutes and
// seeks no confirmation. The context is cancelled directly; no signal is
// delivered (FR-013; research.md R7; Gap G2).
func TestRunReportsElapsedOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer

	// Cancel a few ticks in, while the interval is still running.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	completed, _ := runBounded(t, ctx, 5*time.Millisecond, &out)
	cancel()

	if completed {
		t.Fatal("Run reported a cancelled interval as completed")
	}

	lines := splitLines(out.String())
	if len(lines) < 2 {
		t.Fatalf("got %d lines, want at least a report and a stopped message:\n%s", len(lines), out.String())
	}
	if len(lines) > focus.Ticks {
		t.Errorf("got %d lines, want fewer than a completed run's %d: the interval did not stop early", len(lines), focus.Ticks+1)
	}

	last := lines[len(lines)-1]
	stopped := regexp.MustCompile(`^Focus interval stopped after (\d+) minutes?\.$`)
	if !stopped.MatchString(last) {
		t.Fatalf("the final line = %q, want an elapsed figure in whole minutes", last)
	}
	if strings.ContainsAny(last, "?") || strings.Contains(strings.ToLower(last), "y/n") {
		t.Errorf("the final line = %q, want no confirmation prompt", last)
	}

	// The elapsed figure must agree with the countdown that preceded it. A
	// report is written at the start of each tick period, so R reports means
	// R-1 whole ticks completed and the cancellation fell inside the Rth.
	elapsedTicks, err := strconv.Atoi(stopped.FindStringSubmatch(last)[1])
	if err != nil {
		t.Fatalf("parsing the elapsed figure in %q: %v", last, err)
	}
	reports := len(lines) - 1
	if want := reports - 1; elapsedTicks != want {
		t.Errorf("reported %d minutes elapsed after %d reports, want %d", elapsedTicks, reports, want)
	}
}

// T034 — an already-cancelled context stops the interval within its first tick
// and reports zero elapsed, which is the boundary the quickstart's Ctrl-C step
// describes. The tick is deliberately long: it leaves cancellation the only
// ready case in the select, so the outcome does not depend on scheduling.
func TestRunWithAnAlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer

	completed, _ := runBounded(t, ctx, 50*time.Millisecond, &out)
	if completed {
		t.Fatal("Run reported a cancelled interval as completed")
	}
	if want := "Focus interval stopped after 0 minutes.\n"; !strings.HasSuffix(out.String(), want) {
		t.Errorf("output = %q, want it to end with %q", out.String(), want)
	}
}

func splitLines(s string) []string {
	trimmed := strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
