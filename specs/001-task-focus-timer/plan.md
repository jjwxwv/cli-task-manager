# Implementation Plan: Task Manager with Focus Timer

**Branch**: `001-task-focus-timer` | **Date**: 2026-08-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-task-focus-timer/spec.md`

## Summary

A single-user Go CLI, `pomotask`, offering exactly three commands: record a task, mark a task
complete, and run a fixed 25-minute focus interval. Task state persists to one local JSON file
through a dedicated persistence package using only the standard library, as
[ADR 0001](../../adr/0001-persist-tasks-to-local-json.md) requires.

The design turns on three decisions. Persistence is a package boundary rather than an
interface, because depguard enforces it at build time and nothing varies behind it. The
interval takes a single tick parameter from which its total length is derived, which makes
FR-016's "one source" structural rather than a matter of discipline. And unreadable stored data
fails loudly and leaves the file untouched, which is the only one of ADR 0001's three candidate
behaviors that needs no machinery beyond what other requirements already demand.

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: Go standard library only, for everything that ships. `golangci-lint`
with `depguard` is development and CI tooling, which the Constitution states is not an
application runtime dependency.

**Storage**: One local JSON file carrying `schema_version` and `tasks`. Path passed into the
persistence package; `main` resolves the default under `os.UserConfigDir()`.

**Testing**: `go test`. Persistence tests use `t.TempDir()`. Interval tests drive the tick seam
rather than waiting.

**Target Platform**: Cross-platform CLI — Windows, macOS, Linux. Developed on Windows 11.

**Project Type**: Single project, command-line tool.

**Performance Goals**: Confirmation of an add visible in under one second (SC-001). Interval
length accurate to within one second (SC-004).

**Constraints**: Persistence confined to one package, standard library only. Domain logic must
not import `encoding/json` and must not touch the filesystem. No user-facing means of changing
the interval duration. No third-party application dependency without developer approval.

**Scale/Scope**: One user, one machine, a task set small enough that whole-file reads and
writes stay acceptable. Sixteen functional requirements, eight success criteria, three commands.

## Constitution Check

*GATE: evaluated before Phase 0, re-evaluated after Phase 1 design. Both passes recorded.*

| Principle | Gate | Pre-design | Post-design |
|-----------|------|-----------|-------------|
| I. ADR Authority | All ADRs in `adr/` read; Accepted ones treated as binding | Pass — ADR 0001 read in full; the two decisions it defers are settled in research.md R1 and R2, not assumed | Pass — no design element conflicts with it; nothing required a superseding ADR |
| II. Approved Persistence | Single local JSON file; stdlib only; dedicated package; domain free of `encoding/json` and filesystem access | Pass | Pass — `internal/storage` is the sole importer of `encoding/json`; neither `internal/task` nor `internal/focus` imports it or any filesystem package, and depguard rule 2 makes that a build error rather than a review finding (research.md R8, R9) |
| III. Traceable Decisions | Every plan decision cites a spec requirement or an Accepted ADR | Pass | Pass — every entry in research.md carries its authority; the five choices resting on neither are named in the Gap register below, and two further conventions are recorded at their point of use, rather than any of them being passed off as requirements |
| IV. Standard-Library-First | Stdlib by default; errors wrapped, never discarded; no panic for expected failures; doc comments on exported identifiers; gofmt, go vet, golangci-lint clean | Pass | Pass — no third-party import proposed; failure paths return errors, including the load failures that would be easiest to swallow |
| V. Necessary Simplicity | Only what the spec or an Accepted ADR requires; no speculative abstraction; interfaces only for required boundaries | Pass | Pass — no interface introduced (R8), no injected clock (R6), no flag parser for a CLI with no flags (R10), no `next_id` field (R3), no backup mechanism (R2) |
| Testing & Enforcement | Automated coverage for every testable behavior; isolated temp locations; CI-verified architecture; depguard; behavioral persistence check | Pass | Pass — two depguard rules: one barring alternative backends inside `internal/storage`, one holding `internal/task` and `internal/focus` clear of serialization, filesystem, storage, and signal imports. Behavioral check entered through `run`, not through the storage package. See Verification strategy |

**No violations. Complexity Tracking is therefore omitted, as the template directs.**

## Project Structure

### Documentation (this feature)

```text
specs/001-task-focus-timer/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 — ten decisions with their authorities
├── data-model.md        # Phase 1 — domain type, persisted document, transitions
├── quickstart.md        # Phase 1 — how to run and validate
├── contracts/
│   └── cli.md           # Phase 1 — invocation, exit statuses, streams
└── checklists/
    └── requirements.md  # Spec quality checklist, 16/16
```

### Source Code (repository root)

```text
cmd/pomotask/
├── main.go              # main is os.Exit(cli()); cli holds path, signal ctx, and the run call
└── main_test.go         # Drives run end to end; carries the ADR 0001 behavioral check

internal/task/
├── doc.go               # Package comment; created by T002 so depguard had a package to bind to
├── task.go              # Task type; Add and Complete as pure functions
└── task_test.go

internal/storage/
├── doc.go               # Package comment; same origin as the others
├── store.go             # Load and save; the only importer of encoding/json
└── store_test.go        # Uses t.TempDir()

internal/focus/
├── doc.go               # Package comment; same origin as the others
├── focus.go             # Interval driven by a tick duration and a context; total = 25 × tick
└── focus_test.go        # Cancels the context directly; no signals involved

README.md                # The three commands, the data file's location per platform, and what
                         # the tool is not — no breaks, no cycles, no history
.gitattributes           # Every text file checks out LF, so the CI gofmt step does not fail on
                         # Windows for line endings alone
.golangci.yml            # depguard: backends barred in storage; purity in task and focus
.github/workflows/ci.yml # gofmt, go vet, golangci-lint, go test — matrix over all three OSes,
                         # plus go test -race on the Linux runner alone
```

The three `doc.go` files exist because Go recognizes no directory as a package until it holds at
least one `.go` file, and T002 needed all three packages to exist before `.golangci.yml` was
written — depguard rules scoped to packages that do not yet exist bind to nothing and pass
vacuously. They now carry each package's doc comment, which is where Go convention puts it.

**Structure Decision**: A single Go module laid out conventionally — `cmd/` for the entry point,
`internal/` for packages that are not a public API. The three internal packages are not an
arbitrary split: `storage` exists because ADR 0001 requires a dedicated persistence package and
gives depguard a scope to bind to, `task` exists because the same ADR forbids domain logic from
importing `encoding/json`, and `focus` exists because FR-014 requires the interval to touch no
task data, which is easiest to guarantee when it cannot reach it.

The template's `src/models`, `src/services`, `src/cli` layout is not used; it is not the Go
convention and would place the persistence boundary somewhere depguard cannot cleanly address.

**`main` is a wrapper around a testable `run`.** `func main()` is one line, `os.Exit(cli())`.
`cli` holds the process concerns — resolving the default data path, building an interrupt
context with `signal.NotifyContext` and deferring its stop function, and calling

```go
run(ctx context.Context, args []string, dataPath string, tick time.Duration, stdout, stderr io.Writer) int
```

with `time.Minute` as the tick. The split exists because `os.Exit` runs no deferred function,
so a `defer stop()` written beside it would never execute; returning a code from `cli` lets its
defers run while keeping `os.Exit` at one place. Everything else — dispatch, load, domain
operation, save, message text, exit status —
lives inside `run`, which a test calls directly with `t.TempDir()`, a compressed tick, its own
cancellable context, and captured buffers.

**Why the context and the tick are parameters rather than internals.** FR-016 names two seams
and these are they. Parts of FR-012, FR-013, SC-007, and SC-008 are command-level: the report
sequence reaching the writer the command was handed, and the exit status on completion and on
interruption. A package-level test produces no exit status at all, so neither can be reached
from inside `internal/focus`.

The tick is the timing seam: total length derives from it, so compressing one compresses both.
The context is the cancellation seam: were `run` to build its own signal context, a test would
have only one way to cancel it — delivering a signal to the test process, which is not
deterministic and which the interval's own tests deliberately avoid. The cancelled half of
SC-008 would have no automated coverage. Passing the context in makes cancellation an ordinary
function call.

SC-004 needs no seam and gets none. Compressing an interval removes the wall-clock property
SC-004 asserts, so it is checked against the production configuration in the manual run.

Neither parameter is reachable through any user-facing surface; `main` sets both to fixed
values. FR-011 is unaffected — this is the internal seam FR-016 explicitly permits. Signal
handling exists in exactly one place, `main`, and `run` never learns why it was cancelled.

This is what makes ADR 0001's behavioral check the check the ADR actually describes. That check
must invoke "the add-task path", not the storage package: a test that calls `storage.Save`
directly proves the storage package writes valid JSON, and proves nothing about whether the
application reaches it. Only a test entering through `run` shows that adding a task, as a user
would, is what produces the file. The check therefore lives in `cmd/pomotask/main_test.go`.

The same seam gives the exit-status and stream assertions of SC-005 and SC-008 a home, since
those are properties of the command, not of any single package.

## Verification strategy

Constitution requirements, and what satisfies each:

| Requirement | Satisfied by |
|-------------|--------------|
| Automated coverage for every testable spec behavior | Unit tests per package, mapped to FRs; CLI-level tests for exit statuses and stream discipline (SC-005, SC-008) |
| Persistence tests isolated from real user data | `t.TempDir()` throughout `internal/storage`; the path is a parameter, so no test can reach the real file even by accident (research.md R1) |
| Architectural constraints verified by CI, not review | `.golangci.yml` and the workflow, both run on every push, across a Windows, Linux, and macOS matrix — the Target Platform above claims three platforms, and `os.UserConfigDir()`, `os.Rename` atomicity, and the unwritable-path case all differ among them. The workflow must test `gofmt -l` by its **output**, not its exit code — `gofmt -l` exits zero even when it names unformatted files, so a naive `&&` chain would report a clean build over unformatted code. Go and `golangci-lint` are pinned to the versions fixed locally, since an unpinned linter can land on the other config schema and fail to parse |
| Dependency restriction enforced with depguard | Two rules, described below |
| Approved JSON path verified behaviorally | A test in `cmd/pomotask/main_test.go` entering through `run` against a temporary directory and decoding the resulting file into a document carrying `schema_version` and `tasks` |
| SC-001 — confirmation in under one second | The same `run`-level add test asserts the call returns within one second against a warm temporary directory |

### The two depguard rules

Constitution states two separate architectural constraints, and both say architecture is
verified by CI rather than by review. One rule cannot carry both, because they are scoped to
different packages.

**Rule 1 — no alternative persistence backend.** Scoped to `internal/storage`. An allowlist
permitting only `$gostd`, with explicit denials for `database/sql`, SQLite drivers, ORMs,
key-value stores, and remote storage SDKs. Each denial carries a description naming ADR 0001, so
a violation reports the governing decision rather than an anonymous lint error. An allowlist is
used rather than a blocklist because it closes every dependency path not explicitly opened.

**Rule 2 — domain purity.** Scoped to `internal/task` **and `internal/focus`**. Denies
`encoding/json` and the filesystem packages — `os`, `io/fs`, `path/filepath` — in both, plus
two further denials in `internal/focus`:

- **`internal/storage`**, which is what turns FR-014 from a promise into a build error. FR-014
  requires the interval to neither read nor modify task data; denying it the persistence package
  means it cannot, rather than merely does not.
- **`os/signal`**, which keeps signal handling in `cmd/pomotask` where the process lives. The
  interval accepts a `context.Context` and ends when it is cancelled, indifferent to why. That
  keeps its tests free of signal delivery — cancelling a context is deterministic, sending
  SIGINT to a test process is not — and leaves one place in the codebase that knows about
  signals.

Constitution requires that domain logic neither import `encoding/json` nor perform filesystem
operations directly, and without this rule that requirement rests on review alone, which the
same document rules out as insufficient evidence of compliance. Rule 1 says nothing about any of
it: a domain package importing `encoding/json` introduces no alternative persistence backend and
passes rule 1 untouched.

ADR 0001 requires both a negative and a positive check and treats neither as a substitute for
the other. The rules above are the negative half: they prove no prohibited dependency was
introduced. The behavioral check is the positive half: it proves the approved path is the one
actually taken. A build can satisfy every dependency rule while never writing a file at all.

#### How rule 2 is configured: two entries, not one

*Recorded during implementation, and the reason is empirical rather than stylistic.*

Rule 2 is one rule in the sense that matters here — one constraint, the same four denials in
both domain packages — but `.golangci.yml` carries it as **two** depguard entries,
`domain-purity-task` and `domain-purity-focus`, alongside `persistence-backends`. Three named
lists implement the two rules described above.

The reason is that **depguard applies a single list to any given file**. Written the obvious way
— one shared entry naming both packages, plus a focus-only entry carrying the two extra denials —
the shared entry wins for every file in `internal/focus`, and the focus-only entry never fires.
An `os/signal` import there was then reported under rule 2's *filesystem* description, because
`os` is a prefix of `os/signal` and the shared list is the one being consulted. The rule still
rejected the import; it named the wrong reason for rejecting it, which defeats the point of
attaching a description at all. Writing one entry per package, with the focus entry carrying all
six denials, puts each denial under its own description. T041's provocation is what confirms it:
`os/signal` in `internal/focus` now reports signal containment rather than filesystem access.

A reviewer counting entries in the configuration against "two rules" here would otherwise find a
discrepancy with no reason attached — the same failure mode G6 in the Gap register exists to
close.

### Coverage of the timing criteria

SC-007's count of 25 reports and FR-012's sequence are tested through the tick seam. SC-004's
one-second accuracy is checked against the production tick in the manual quickstart run, since
that is the one property a compressed interval cannot demonstrate.

CI carries one step this table does not require: `go test -race ./...`, on the Linux runner
alone. It closes no Constitution gate — the gate is the four checks above — and rests on
FR-016's cancellation seam, which is what puts concurrency in the test surface at all. It runs
only in CI because the detector needs a C compiler the development machine does not have, so the
local gate and CI are knowingly not the same set of checks. Recorded with its reasoning and its
cost in research.md R11, and declared in quickstart.md so it is not a gate discovered on a red
build.

SC-001's one-second bound is asserted at the `run` level rather than measured precisely. Machine
and CI timing vary too much for a sub-second assertion to mean anything exact; what the check
does catch is a regression that changes the order of magnitude — a stray sleep, a retry loop, or
a read that stops being proportional to a small file. Recorded as a smoke bound, not a
benchmark.

## Gap register

Constitution Principle III requires surfacing choices that neither the specification nor an
Accepted ADR authorizes, rather than deciding them silently. Six qualify. All are recorded in
the artifacts that depend on them and none blocks implementation, but each is the developer's to
overturn.

G1 through G5 were identified during planning. G6 surfaced during implementation, when the
depguard configuration had to be written against a real package graph rather than described.

| # | Choice made | Why it is a gap | Where it lives |
|---|-------------|-----------------|----------------|
| G1 | `pomotask done <id>` on an already-complete task exits `0` | FR-008 requires the no-op be reported and nothing changed, but assigns no exit status. Exiting `0` reads it as "the requested state holds"; exiting non-zero would read it as a rejected operation. Both are defensible and FR-015 is satisfied either way, since the message does not claim the operation took effect | [contracts/cli.md](contracts/cli.md) |
| G2 | Interruption reports elapsed time in whole minutes | FR-013 requires printing "how much time had elapsed" without fixing the granularity. Whole minutes keeps the figure consistent with FR-012's countdown and testable under a compressed tick; exact elapsed time would be truthful to the second but would print milliseconds in tests | [contracts/cli.md](contracts/cli.md) |
| G3 | Interruption exits `1` rather than the shell convention `130` | SC-008 requires only that completion and interruption be distinguishable. Nothing selects a particular value | [research.md](research.md) R7 |
| G4 | `pomotask add` takes exactly one argument; the remainder is not joined | FR-001 requires the text be supplied "in a single command" and says nothing about argument count. Requiring one argument makes the contract crisp, and its failure message prints the quoting form rather than a generic usage line, but it rejects `pomotask add write the report`, which many users will type first. Joining the remainder would accept it, at the cost of a looser contract and of `add` silently normalizing runs of spaces | [contracts/cli.md](contracts/cli.md) |
| G5 | An interrupted interval prints to stdout and is not counted as an SC-005 failure, despite exiting non-zero | SC-005 binds "every operation the system rejects or cannot complete", and an interrupted interval did not complete — so it arguably falls inside, which would put the message on stderr alongside every other non-zero exit. The contract instead reads interruption as the user getting what they asked for rather than a fault, and routes it to stdout. Nothing in the spec settles which reading is right. The cost of the choice is that "non-zero implies stderr" stops being a rule a test can assert uniformly | [contracts/cli.md](contracts/cli.md) |
| G6 | depguard rule 1's allowlist admits `pomotask/internal/task` and `pomotask/internal/storage` alongside `$gostd` | ADR 0001's Enforcement section states the allowlist permits only `$gostd` within the persistence package. Read literally that forbids the very import the same ADR's Decision forces: keeping `encoding/json` out of the domain means the persistence package owns the conversion, and converting to the domain type means importing it. The two clauses cannot both hold as written for any design in which the persistence package persists tasks, and this is the side that gives. Opening two first-party paths introduces no alternative backend, so the invariant the rule enforces stands; the departure is from the ADR's wording, and nothing authorized it in advance. The second path is the external test package `storage_test`, which shares the directory and so falls inside the rule's file scope | [adr/0001-persist-tasks-to-local-json.md](../../adr/0001-persist-tasks-to-local-json.md), Enforcement → "Note on the allowlist's exact contents"; [.golangci.yml](../../.golangci.yml) |

**Why G6 is recorded in three places rather than one.** A code comment in `.golangci.yml` sits
where nobody reading the documents will pass it, so a reviewer diffing the ADR's text against
the configuration would meet the divergence with no reason attached. The note lives in the ADR
because that is the document being departed from, and here because the Gap register is what a
reviewer reads before the configuration.

Two further choices rest on convention rather than requirement and are recorded at their point
of use rather than here: failure messages on stderr (research.md R10), and storing task data
under `os.UserConfigDir()` despite tasks being data rather than configuration (research.md R1,
including why the semantically better alternative was rejected).

A third belongs with them and is likewise recorded at its point of use: running the race
detector in CI but not in the local gate (research.md R11, including the authority chain, which
runs through FR-016's cancellation seam rather than through the Constitution, and the accepted
asymmetry between the two check sets).

## Phase status

- **Phase 0** — complete. [research.md](research.md), ten decisions, no `NEEDS CLARIFICATION`
  outstanding.
- **Phase 1** — complete. [data-model.md](data-model.md), [contracts/cli.md](contracts/cli.md),
  [quickstart.md](quickstart.md).
- **Phase 2** — not started. `tasks.md` is produced by `/speckit-tasks`, not by this command.
