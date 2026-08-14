# pomotask

A single-user command-line task manager with a fixed 25-minute focus interval,
written in Go with no dependency outside the standard library.

Three commands. Nothing else.

```bash
go build ./cmd/pomotask
```

## Commands

### `pomotask add <text>`

Records a task and prints the identifier assigned to it.

```text
$ pomotask add "write the report"
Added task 1: write the report
```

**Keep that number.** There is no list command, so the confirmation above is the
only occasion on which the tool ever tells you a task's identifier. Once it
scrolls out of the terminal, nothing in the tool will show it to you again.

Text containing spaces must be quoted. `add` takes exactly one argument and does
not join the remainder, so `pomotask add write the report` is rejected rather
than silently reinterpreted.

Empty or whitespace-only text is rejected and nothing is recorded.

### `pomotask done <id>`

Marks a recorded task complete, naming it by the identifier `add` printed.

```text
$ pomotask done 1
Completed task 1
```

Running it again on the same task reports that nothing needed doing and exits
`0` — the state you asked for holds:

```text
$ pomotask done 1
Task 1 is already complete
```

An identifier matching no task is an error, exits `1`, and changes nothing.

### `pomotask focus`

Runs one 25-minute interval, printing the time remaining once a minute and a
line announcing the end.

```text
$ pomotask focus
25 minutes remaining
24 minutes remaining
...
1 minute remaining
Focus interval complete.
```

Exactly 25 remaining-time lines, counting from 25 down to 1 — no line shows
zero — followed by one completion line. Exit status `0`.

Press Ctrl-C to stop early. It prints the whole minutes elapsed, asks for no
confirmation, and exits `1`:

```text
Focus interval stopped after 8 minutes.
```

The differing exit status is the point: a completed interval and an interrupted
one are distinguishable without reading the output.

`focus` takes no arguments and the duration cannot be changed — not by an
argument, a flag, an environment variable, or a configuration file. Supplying
one is rejected rather than ignored, so you can always tell a refused option
from an accepted one that did nothing.

The interval reads and writes no task data at all, so it runs whether or not
your data file exists, and adding a task in another shell while one is running
cannot interfere with it.

## Exit statuses

| Status | Meaning |
|--------|---------|
| `0` | The operation took effect, or the requested state already held |
| `1` | The operation did not take effect, or a focus interval was interrupted |

Failure messages go to stderr. The one exception is the focus interruption
message, which goes to stdout: stopping an interval is a deliberate action
rather than a fault.

## Where your tasks are stored

One JSON file, under your platform's user configuration directory:

| Platform | Path |
|----------|------|
| Windows | `%AppData%\pomotask\tasks.json` |
| macOS | `~/Library/Application Support/pomotask/tasks.json` |
| Linux | `${XDG_CONFIG_HOME:-$HOME/.config}/pomotask/tasks.json` |

It is meant to be read on sight:

```json
{
  "schema_version": 1,
  "tasks": [
    { "id": 1, "text": "write the report", "done": true }
  ]
}
```

No setup step creates it. The first `add` does.

If the file carries a `schema_version` this build does not support, or cannot be
decoded at all, the command says so and stops. Your file is left exactly as it
was — nothing is moved, truncated, or rewritten, and the tool will not repair it
for you. Restore your own copy.

## What this is not

The word Pomodoro sets expectations this tool deliberately does not meet.
`focus` is one fixed 25-minute interval and nothing more:

- **No break intervals** and no long breaks. When the interval ends, it ends.
- **No cycle counting.** Nothing tracks how many intervals you have run.
- **No session history, statistics, or reporting.** Completed intervals are not
  recorded anywhere; the interval leaves no trace once the command exits.
- **No notifications and no sound.** The end is announced by a printed line.
- **No pause or resume.** Ctrl-C stops the interval; it does not suspend it.

Nor is there a list, delete, or edit command, and tasks carry no priority, due
date, tag, or ordering.

These absences are the scope, not unfinished work.

## Assumptions

One person, one machine. Commands are separate short-lived invocations, and
behavior when two task-modifying commands run at the same instant is not
specified — the last write wins.

## Development

The quality gate, which must pass before any change is complete:

```bash
test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run && go test ./...
```

PowerShell:

```powershell
$unformatted = gofmt -l .; if ($unformatted) { Write-Error "unformatted: $unformatted" } else { go vet ./... && golangci-lint run && go test ./... }
```

`gofmt -l` exits zero whether or not it names unformatted files, so both forms
test its output rather than its exit code.

CI runs one step this gate does not: `go test -race ./...`, on the Linux runner
only. The race detector needs cgo and so a C compiler, which the Windows
development machine does not carry. A commit that passes the gate above can
therefore still go red in CI on a check you could not run first — expected, and
explained in [research.md R11](specs/001-task-focus-timer/research.md).

`golangci-lint` carries two `depguard` rules that enforce the architecture at
build time rather than by review: no alternative persistence backend inside
`internal/storage`, and no serialization, filesystem access, task data, or
signal handling inside `internal/task` and `internal/focus`. The version is
pinned in [.github/workflows/ci.yml](.github/workflows/ci.yml), which is the
single place it lives; the config is written in schema v2 and a v1 binary
cannot parse it.

Design documents live in [specs/001-task-focus-timer/](specs/001-task-focus-timer/),
and the persistence decision in [adr/0001-persist-tasks-to-local-json.md](adr/0001-persist-tasks-to-local-json.md).
