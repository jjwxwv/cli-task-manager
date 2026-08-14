package focus

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Ticks is the number of ticks one focus interval runs for.
//
// The interval's total length is Ticks × tick, derived rather than supplied.
// That is what makes FR-016's "one source" structural rather than a matter of
// discipline: there is no second duration or count on the exported surface for
// a compressed run to disagree with, so compressing the cadence compresses the
// total with it.
const Ticks = 25

// Run runs one focus interval and reports whether it ran to completion.
//
// A remaining-time report is written to out before each tick, counting down
// from Ticks to 1 — remaining ticks, always worded in minutes, so the sequence
// reads identically however short the tick is (research.md R6). On reaching the
// full length Run writes a completion line and reports true. If ctx is
// cancelled first, Run writes the whole minutes elapsed and reports false,
// prompting for nothing. Run never learns why ctx was cancelled, which is what
// lets a test cancel it by ordinary function call rather than by signalling its
// own process (FR-010 through FR-014, FR-016).
//
// The error result reports a failure to write to out. It is never a way of
// reporting that the interval was cut short; that is the boolean.
func Run(ctx context.Context, tick time.Duration, out io.Writer) (bool, error) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for remaining := Ticks; remaining > 0; remaining-- {
		if err := writeLine(out, "%s remaining", minutes(remaining)); err != nil {
			return false, err
		}

		select {
		case <-ctx.Done():
			if err := writeLine(out, "Focus interval stopped after %s.", minutes(Ticks-remaining)); err != nil {
				return false, err
			}
			return false, nil
		case <-ticker.C:
		}
	}

	if err := writeLine(out, "Focus interval complete."); err != nil {
		return false, err
	}
	return true, nil
}

// writeLine writes one formatted line and wraps any failure with context, so
// that a caller unable to report progress says so rather than continuing
// silently.
func writeLine(out io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(out, format+"\n", args...); err != nil {
		return fmt.Errorf("reporting focus interval progress: %w", err)
	}
	return nil
}

// minutes renders a tick count as a minute figure, singular at one.
func minutes(n int) string {
	if n == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", n)
}
