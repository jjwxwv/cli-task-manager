# Phase 1 Contract: `pomotask` command-line interface

**Feature**: `001-task-focus-timer` | **Date**: 2026-08-14

The CLI is the only interface this feature exposes. This document fixes the invocation forms,
exit statuses, and output streams so that acceptance tests assert against a stated contract
rather than against whatever the implementation happens to print.

Command names were supplied by the developer; the specification never named them. Message
wording below is normative in **shape** — which stream, which stream ordering, which
identifier appears — and illustrative in phrasing.

---

## Invocation summary

| Command | Purpose | Requirement |
|---------|---------|-------------|
| `pomotask add <text>` | Record a new task | FR-001 |
| `pomotask done <id>` | Mark a task complete | FR-005 |
| `pomotask focus` | Run a 25-minute focus interval | FR-010 |

No flags exist on any command. FR-011 forbids a user-facing means of changing the interval
duration, and no other requirement calls for an option.

---

## `pomotask add <text>`

Takes exactly one positional argument. Text containing spaces must be quoted by the shell;
`pomotask add write the report` is rejected rather than joined. **Requiring one argument rather
than joining the remainder is a planning choice** — see G4 in the Gap register in
[plan.md](../plan.md).

**Success** — stdout, exit `0`:

```text
Added task 3: write the report
```

The identifier is mandatory in this line. FR-003 makes this the only occasion on which the
system ever discloses it.

**Failures** — stderr, exit `1`:

| Condition | Requirement |
|-----------|-------------|
| No argument, or more than one | FR-015 |
| Text empty or whitespace-only | FR-002 |
| Stored data unreadable (see [Load failures](#load-failures)) | research.md R2 |
| Data file cannot be written | Spec edge case, FR-015 |

Nothing is recorded when any of these occurs.

The argument-count failure prints the quoting form rather than the generic usage line, since
the mistake it corrects is almost always an unquoted multi-word argument:

```text
pomotask add takes exactly one argument; quote text containing spaces:
    pomotask add "write the report"
```

This message is what G4 relies on. A generic `usage:` line here would leave the user to work
out the quoting themselves, and G4's rationale would not hold.

---

## `pomotask done <id>`

Takes exactly one positional argument: an identifier previously printed by `add`.

**Success** — stdout, exit `0`:

```text
Completed task 3
```

**Already complete** — stdout, exit `0`:

```text
Task 3 is already complete
```

FR-008 requires this be reported and nothing changed. It is not a failure: the state the user
asked for holds. FR-015 is satisfied because the line does not claim the operation took
effect. **This exit status is a planning choice, not a specification requirement** — see the
Gap register in [plan.md](../plan.md).

**Failures** — stderr, exit `1`:

| Condition | Requirement |
|-----------|-------------|
| No argument, more than one, or a non-integer | FR-015 |
| No task carries that identifier | FR-007 |
| Stored data unreadable | research.md R2 |
| Data file cannot be written | Spec edge case, FR-015 |

---

## `pomotask focus`

Takes no arguments. Blocks for the duration of the interval. Reads and writes no task data at
all (FR-014), so it succeeds even when the data file is missing or unreadable.

**Failure** — stderr, exit `1`:

| Condition | Requirement |
|-----------|-------------|
| Any argument is supplied, of any form | FR-011, FR-015 |

```text
pomotask focus takes no arguments; the 25-minute duration is fixed
```

Supplied arguments MUST be rejected rather than ignored. FR-011 forbids a user-facing means of
changing the duration, and silently accepting `pomotask focus 5` or `pomotask focus --minutes 5`
would leave a user unable to tell a rejected option from an accepted one that did nothing —
which is the state FR-011 exists to prevent. Rejection is also what makes the prohibition
observable to a test; an ignored argument produces the same output as no argument and proves
nothing.

**Completion** — stdout, exit `0`:

```text
25 minutes remaining
24 minutes remaining
...
1 minute remaining
Focus interval complete.
```

Exactly 25 remaining-time lines, the first showing 25 and the last showing 1, followed by
exactly one completion line. No line shows zero. Fixed by SC-007 and FR-012.

**Interruption** — stdout, exit `1`:

```text
Focus interval stopped after 8 minutes.
```

Printed when the process receives an interrupt signal before the interval completes. No
confirmation is requested (FR-013). The elapsed figure counts whole elapsed minutes, so an
interruption during the first minute reports `0 minutes`. **Reporting elapsed time in whole
minutes rather than exactly is a planning choice** — G2 in the Gap register in
[plan.md](../plan.md).

**Routing this message to stdout, and treating it as outside SC-005's failure framing, is also
a planning choice** — G5 in the same register. It is the one non-zero exit in this contract
that does not print to stderr.

The non-zero status is what SC-008 requires: a completed interval and an interrupted one must
be distinguishable by exit status alone.

---

## Unrecognized invocation

`pomotask` with no arguments, or with an unknown first argument, prints usage to stderr and
exits `1`. FR-015 requires every failure to be reported, and an invocation the system cannot
act on is one.

```text
usage: pomotask <add|done|focus> [argument]
```

---

## Load failures

`add` and `done` both read stored data before acting. Three outcomes, and they must not be
conflated:

| Situation | Behavior | Authority |
|-----------|----------|-----------|
| File absent | Proceed with an empty task set | Spec edge case (first use) |
| `schema_version` unrecognized | Report the version found and the version supported, exit `1` | ADR 0001 |
| File present but not decodable | Report that the data could not be read, exit `1` | research.md R2 |

In both failure cases the data file is left exactly as found. Nothing is moved, truncated, or
rewritten. This is what makes the spec's guarantee — never silently start from an empty set,
never discard the user's data — observable.

---

## Exit status summary

| Status | Meaning |
|--------|---------|
| `0` | The operation took effect, or the requested state already held |
| `1` | The operation did not take effect, or a focus interval was interrupted |

Every non-zero exit is accompanied by a message naming what went wrong, on stderr, except the
focus interruption, which is a deliberate user action rather than a fault and reports on
stdout.

---

## Stream discipline

Normal output goes to stdout; failure messages go to stderr. No requirement assigns streams —
this is convention, recorded as a choice in research.md R10 — but it is contractual here so
tests can assert against the right stream.
