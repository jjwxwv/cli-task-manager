# Phase 1 Quickstart: validating `pomotask`

**Feature**: `001-task-focus-timer` | **Date**: 2026-08-14

How to run the feature and confirm it does what the specification requires. Message shapes come
from [contracts/cli.md](contracts/cli.md); stored-data structure from
[data-model.md](data-model.md).

Commands are given for **PowerShell**, the shell on the development machine, with the POSIX
equivalent beneath where the two differ. On Windows the built binary is `pomotask.exe`; on macOS
and Linux it is `pomotask`.

---

## Prerequisites

- Go 1.26 or later — `go version`
- `golangci-lint` on `PATH`, at the version pinned in `.github/workflows/ci.yml` — not needed
  to run the tool, needed for the quality gate. The workflow is the single place that version
  lives; matching it locally is what keeps a config written for one config schema from failing
  to parse under the other. Installing it is a setup task in `tasks.md`, not a step here.

---

## Build

```powershell
go build ./cmd/pomotask
```

---

## Automated checks

The full gate the Constitution requires before implementation is considered complete:

```powershell
$unformatted = gofmt -l .
if ($unformatted) { Write-Error "unformatted: $unformatted" } else { go vet ./... && golangci-lint run && go test ./... }
```

POSIX equivalent:

```bash
test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run && go test ./...
```

`gofmt -l` **exits zero whether or not it finds anything** — it reports by printing filenames,
not by its status. Chaining it with `&&` would therefore pass a build with unformatted files in
it. The forms above test its output rather than its exit code. Anything the same trap applies
to in CI must be written the same way.

### Confirming the architectural checks actually bite

A dependency rule that has never failed is a rule nobody has tested. There are **two** depguard
rules; the provocations below **sample** them rather than exhaust them. Rule 1 has one
provocation and is fully covered by it. Rule 2 carries several denials across two packages, and
four of them are provoked here — enough to show the rule is live in both packages and in each
category of denial, not enough to claim every denied import has been individually confirmed.

- **Rule 1, backends** — temporarily add `import "database/sql"` to a file in
  `internal/storage`, run `golangci-lint run`, and confirm it fails with a message naming
  ADR 0001.
- **Rule 2, serialization** — temporarily add `import "encoding/json"` to a file in
  `internal/task`, run `golangci-lint run`, and confirm it fails. This one passes rule 1
  untouched, which is why the two rules exist separately.
- **Rule 2, filesystem** — temporarily add `import "os"` to a file in `internal/task` and
  confirm it fails. Serialization and filesystem access are separate denials and a rule
  covering one does not imply the other.
- **Rule 2, task data out of the interval** — temporarily import `internal/storage` from a file
  in `internal/focus`, run `golangci-lint run`, and confirm it fails. This is FR-014 enforced
  as a build error rather than as a promise: the interval cannot reach task data, rather than
  merely not reaching it.
- **Rule 2, signal containment** — temporarily add `import "os/signal"` to a file in
  `internal/focus`, run `golangci-lint run`, and confirm it fails. This is what keeps the
  interval driven by a context it does not own, so its tests cancel deterministically instead
  of delivering a signal to a test process.

Remove every temporary import afterwards.

These confirm the enforcement mechanism. They are separate from the behavioral persistence
test, which proves the approved JSON path is the one actually used — ADR 0001 requires both, and
neither substitutes for the other.

---

## Validating User Story 1 — capture a task

```powershell
.\pomotask.exe add "write the report"
```

Expected on stdout, exit `0`: `Added task 1: write the report`

Note the identifier. It is the only time the system will show it.

**Confirmation speed (SC-001)**: the response should appear immediately. The automated bound is
asserted at the `run` level; this run is the human check that nothing about the real filesystem
path makes it feel slow.

**Persistence across sessions** — run again in a new shell:

```powershell
.\pomotask.exe add "review the draft"
```

Expected: `Added task 2: review the draft`. The identifier advancing to 2 shows the first task
was read back from disk, which is SC-002.

**Rejecting empty text**:

```powershell
.\pomotask.exe add ""
```

Expected on stderr, exit `1`: a message stating the text must not be empty. Nothing recorded
(FR-002).

**Rejecting an unquoted multi-word argument**:

```powershell
.\pomotask.exe add write the report
```

Expected on stderr, exit `1`, naming the quoting form rather than a generic usage line:

```text
pomotask add takes exactly one argument; quote text containing spaces:
    pomotask add "write the report"
```

`add` takes exactly one argument and does not join the remainder — a planning choice recorded
as G4 in [plan.md](plan.md), whose rationale rests on this message teaching the fix rather than
merely reporting the fault.

---

## Validating User Story 2 — mark a task complete

```powershell
.\pomotask.exe done 1
```

Expected on stdout, exit `0`: `Completed task 1`

**Idempotence** — run the same command again:

```powershell
.\pomotask.exe done 1
```

Expected on stdout, exit `0`: `Task 1 is already complete`. Nothing changes (FR-008).

**Unknown identifier**:

```powershell
.\pomotask.exe done 99
```

Expected on stderr, exit `1`: a message stating no task carries that identifier (FR-007).

---

## Validating User Story 3 — run a focus interval

```powershell
.\pomotask.exe focus
```

Expected: `25 minutes remaining` immediately, then one line per minute counting down to
`1 minute remaining`, then `Focus interval complete.` — 25 remaining-time lines and one
completion line, no line showing zero (SC-007), exit status `0`.

Check the status without sitting through it again:

```powershell
$LASTEXITCODE
```

POSIX equivalent: `echo $?`

**Rejecting an attempt to change the duration**:

```powershell
.\pomotask.exe focus 5
```

Expected on stderr, exit `1`: a message stating `focus` takes no arguments and the duration is
fixed. Rejection rather than silent ignoring is what makes FR-011 observable — an ignored
argument produces identical output to no argument and proves nothing.

**Interruption** — start it again and press Ctrl-C after a moment:

```powershell
.\pomotask.exe focus
```

Expected: `Focus interval stopped after 0 minutes.` and exit status `1`, with no confirmation
prompt (FR-013). The differing exit status is SC-008.

Waiting 25 minutes is not how this gets tested in CI. The automated tests drive the interval
through the tick seam described in [research.md](research.md) R6, which compresses the cadence
along with the duration and produces the identical 25-line sequence in milliseconds. This manual
run confirms the production tick is genuinely one minute — the one thing the compressed test
cannot show.

---

## Inspecting stored data

```powershell
Get-Content "$env:AppData\pomotask\tasks.json"
```

On macOS the file is at `~/Library/Application Support/pomotask/tasks.json`; on Linux at
`${XDG_CONFIG_HOME:-$HOME/.config}/pomotask/tasks.json`.

Expected: a JSON document carrying `schema_version` and `tasks`, readable without tooling. That
it is legible on sight is the property ADR 0001 chose this format for.

**Version rejection** — edit `schema_version` to `2`, then run any command:

```powershell
.\pomotask.exe add "anything"
```

Expected on stderr, exit `1`: a message naming the version found and the version supported. The
file is left exactly as it was (ADR 0001).

**Malformed data** — truncate the file mid-document and run any command. Expected on stderr,
exit `1`: a message that the data could not be read, with the file untouched. Restore your own
copy afterwards; the tool will not do it for you, which is the behavior chosen in
[research.md](research.md) R2.
