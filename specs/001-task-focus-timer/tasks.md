---

description: "Task list for 001-task-focus-timer"
---

# Tasks: Task Manager with Focus Timer

**Input**: Design documents from `/specs/001-task-focus-timer/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/cli.md](contracts/cli.md),
[quickstart.md](quickstart.md)

**Tests**: Included, and not optional here. The Constitution requires that every testable
behavior a specification demands has corresponding automated coverage, which makes test tasks
binding rather than a stylistic preference.

**Traceability**: every task below cites the specification requirement, success criterion, or
Accepted ADR clause that authorizes it, as Constitution Principle III requires. A task with no
citation would be a decision nobody approved.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel — different files, no dependency on incomplete work
- **[Story]**: US1, US2, US3, mapping to the user stories in spec.md
- Paths follow the Go layout fixed in plan.md, not the template's `src/` convention

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: A buildable module with the architectural checks in place before any behavior
exists. The order below is deliberate: the linter configuration cannot be written correctly
until the installed version is known.

- [ ] T001 Run `go mod init pomotask` at the repository root, producing `go.mod` with a `go` directive of 1.26
- [ ] T002 Create the package skeleton — `cmd/pomotask/`, `internal/task/`, `internal/storage/`, `internal/focus/` — with a minimal compiling `cmd/pomotask/main.go` **and a placeholder `.go` file carrying a package clause and doc comment in each of the three internal directories**. Go recognizes no directory as a package until it holds at least one `.go` file, so depguard rules scoped to packages that do not yet exist would have nothing to bind to and T005 would pass vacuously (plan.md Project Structure)
- [ ] T003 Install `golangci-lint` and record the exact version reported by `golangci-lint --version`. **Determine whether that version reads schema v1 or v2 before writing any config** — the two are not interchangeable, and a v1 file under a v2 binary fails to parse rather than degrading gracefully
- [ ] T004 Write `.golangci.yml` in the schema the installed version requires, containing both depguard rules described in plan.md: **rule 1** scoped to `internal/storage`, an allowlist permitting only `$gostd` with explicit denials for `database/sql`, SQLite drivers, ORMs, key-value stores, and remote storage SDKs, each denial carrying a description naming ADR 0001; **rule 2** scoped to `internal/task` and `internal/focus`, denying `encoding/json`, `os`, `io/fs`, `path/filepath`, and additionally `pomotask/internal/storage` and `os/signal` within `internal/focus` (ADR 0001 Enforcement; Constitution Testing & Architectural Enforcement)
- [ ] T005 Run `golangci-lint run` against the skeleton and confirm the configuration parses, resolves all **three** scoped packages — `internal/storage`, `internal/task`, `internal/focus`; `cmd/pomotask` carries no rule — and exits cleanly. A config that fails to load is otherwise indistinguishable from one that passes everything. **This proves the rules are well-formed, not that they bite** — nothing here imports anything yet. T038 through T040 are what demonstrate they actually reject
- [ ] T006 Create `.github/workflows/ci.yml` running `gofmt`, `go vet ./...`, `golangci-lint run`, and `go test ./...`. **The gofmt step MUST fail on non-empty output, not on exit status** — `gofmt -l` exits zero even while naming unformatted files, so a plain `&&` chain reports a clean build over unformatted code (plan.md Verification strategy)
- [ ] T007 Verify each CI step locally in PowerShell with the equivalents given in quickstart.md, confirming the gofmt check actually fails when a deliberately unformatted file is present

**Checkpoint**: The module builds, both architectural rules are configured, and CI runs. No behavior exists yet.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The command seam every story dispatches through. Nothing story-specific belongs here.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T008 Define `run(ctx context.Context, args []string, dataPath string, tick time.Duration, stdout, stderr io.Writer) int` in `cmd/pomotask/main.go`, holding all dispatch, message text, and exit-status logic.

  **Both the context and the tick belong on this signature rather than inside the focus branch.** They are the two seams FR-016 names. Parts of FR-012, FR-013, SC-007, and SC-008 are command-level: the report sequence reaching the writer the command was handed, and the exit status on completion and on interruption. A package-level test produces no exit status at all. A branch that built its own signal context would leave a test no way to cancel it short of delivering a signal to the test process — which T033 rules out as non-deterministic — making the cancelled half of SC-008 unverifiable.

  Both parameters are set by `main` to fixed values and are reachable through no user-facing surface, so FR-011 is untouched. Branches that run no interval ignore the tick; their tests may pass any value (plan.md Structure Decision; FR-011, FR-016)
- [ ] T009 Reduce `func main()` to a single line, `os.Exit(cli())`, and put the process concerns in `cli() int`: resolving the default data path via `os.UserConfigDir()` joined with `pomotask/tasks.json`, building the interrupt context with `signal.NotifyContext(context.Background(), os.Interrupt)`, deferring its stop function, and calling `run` with that context and `time.Minute` as the production tick.

  **The split is not cosmetic.** `signal.NotifyContext` returns a stop function that must be called, and `os.Exit` runs no deferred function — a `defer stop()` written directly in `main` alongside `os.Exit` never executes. Returning a code from `cli` lets its defers run before the process ends, and keeps `os.Exit` at exactly one place.

  **Signal handling lives here and nowhere else**: `run` receives a context and never learns why it was cancelled (research.md R1, R7)
- [ ] T010 Implement the usage message and the unrecognized-invocation path in `cmd/pomotask/main.go` — no arguments or an unknown first argument prints usage to stderr and returns 1 (FR-015; contracts/cli.md)
- [ ] T011 Write the unrecognized-invocation test in `cmd/pomotask/main_test.go`, asserting stderr carries the usage text, stdout is empty, and the returned code is 1 (FR-015, SC-005)

**Checkpoint**: `run` exists and is directly callable by tests with captured buffers and a temporary path. User stories can now begin, and US3 does not depend on either of the others.

---

## Phase 3: User Story 1 - Capture a task (Priority: P1) 🎯 MVP

**Goal**: A user records a task and receives its identifier, and the task survives into a later session.

**Independent Test**: Add a task, end the session, start a new one, add a second task, and confirm the identifier advances — showing the first was read back from disk.

### Tests for User Story 1

- [ ] T012 [P] [US1] Write `internal/task/task_test.go` covering `Add`: identifier assigned as one greater than the largest present and starting at 1, empty and whitespace-only text rejected, existing tasks untouched, and **two tasks carrying identical text receiving different identifiers** (FR-001, FR-002, FR-006, spec duplicate-text edge case; research.md R3)
- [ ] T013 [P] [US1] Write `internal/storage/store_test.go` round-trip and empty-state cases against `t.TempDir()`: save then load returns what was saved; an absent file yields an empty task set rather than an error, and a save into a directory holding no data file succeeds without any prior initialization step (FR-004, SC-006, spec first-use edge case; Constitution isolated temp locations)
- [ ] T014 [US1] Extend `internal/storage/store_test.go` with the load-failure cases: `schema_version` of 2 fails naming both versions found and supported; a truncated document fails; **in both cases assert the file's bytes are unchanged afterwards** (ADR 0001; research.md R2)
- [ ] T015 [US1] Extend `internal/storage/store_test.go` to assert the write path leaves no stray temporary file in the destination directory and that the destination survives a save (ADR 0001 write mandate; research.md R5)
- [ ] T016 [US1] Write the ADR 0001 behavioral check in `cmd/pomotask/main_test.go`: call `run` with `add` arguments against `t.TempDir()`, then decode the resulting file through `encoding/json` and assert it carries `schema_version` and `tasks`. **This must enter through `run`, not call the storage package directly** — a test that calls `storage.Save` proves the package writes JSON and proves nothing about whether the application reaches it (ADR 0001 Enforcement; plan.md Structure Decision)
- [ ] T017 [US1] Extend `cmd/pomotask/main_test.go` to assert the **add confirmation on stdout carries the assigned identifier** — matching `Added task 1: write the report` for a first task — with stderr empty and code 0. FR-003 makes this line the only occasion on which the system ever discloses the identifier, so an implementation that persisted correctly while printing nothing usable would satisfy every other test in this phase and still leave the user unable to complete anything (FR-003; contracts/cli.md)
- [ ] T018 [US1] Extend `cmd/pomotask/main_test.go` with the persistence sequence: call `run` with `add` twice against the same temporary path, in separate calls, and assert the second confirmation carries identifier 2. This is the Independent Test for this story stated as code — the advancing identifier is what shows the first task was read back from disk rather than the file being rewritten from empty (SC-002, FR-004, FR-006)
- [ ] T019 [US1] Extend `cmd/pomotask/main_test.go` to assert an `add` call returns within one second against a warm temporary directory, recorded as a smoke bound rather than a benchmark (SC-001; plan.md Coverage of the timing criteria)
- [ ] T020 [US1] Extend `cmd/pomotask/main_test.go` with the argument-count cases: zero arguments and two or more each produce the quoting message on stderr and code 1, and record nothing (FR-015; contracts/cli.md; Gap G4)
- [ ] T021 [US1] Extend `cmd/pomotask/main_test.go` with the run-level failure paths: place a data file carrying `schema_version` 2 in `t.TempDir()` and assert `add` reports on **stderr** with code 1 and leaves the file unchanged; then point `dataPath` at a path that cannot be written — a directory created **inside `t.TempDir()`** serves, portably, since renaming a file onto a directory fails on every platform — and assert the write failure is reported rather than reported as success. Every path this test touches stays inside the temporary directory; nothing may reach a real location (Constitution isolated temp locations). T014 proves the storage package returns these errors; this proves the command surfaces them on the right stream with the right status (contracts/cli.md Load failures; FR-015, SC-005)

### Implementation for User Story 1

- [ ] T022 [US1] Implement `internal/task/task.go` — the `Task` type carrying `ID`, `Text`, `Done`, and `Add` as a pure function over `[]Task`. Doc comments on every exported identifier. This package imports neither `encoding/json` nor any filesystem package (data-model.md; Constitution Principle IV; depguard rule 2)
- [ ] T023 [US1] Implement `internal/storage/store.go` — a persisted record type carrying the JSON tags, conversion to and from `task.Task`, `Load` reading the file once and checking `schema_version` **before** decoding records, and `Save` writing through `os.CreateTemp` in the destination's own directory followed by `os.Rename`. Errors wrapped with context, never discarded (ADR 0001 Decision; research.md R4, R5, R9)
- [ ] T024 [US1] Implement the `add` branch of `run` in `cmd/pomotask/main.go`: load, apply `task.Add`, save, print `Added task N: text` to stdout, return 0. Failures print to stderr and return 1 without recording anything (FR-001 through FR-004, FR-015; contracts/cli.md)

**Checkpoint**: `pomotask add` works end to end and persists across invocations. This is the MVP.

---

## Phase 4: User Story 2 - Mark a task complete (Priority: P2)

**Goal**: A user marks a recorded task complete by its identifier, and the completion persists.

**Independent Test**: With one task recorded, mark it complete, start a new session, and confirm it is still complete.

**Shared groundwork**: `internal/task` and `internal/storage` serve both US1 and US2 and are built in US1 as the earlier story, per the task-organization rule for shared entities. T014, T015, and T021 in particular — the load-failure, atomic-write, and run-level failure cases — cover storage behavior US2 relies on just as much as US1 does. US2 is therefore *behaviorally* independent of US1, in that completing a task neither requires nor disturbs the add path, but it is not *buildable* before those tasks exist. The Dependencies section names them.

### Tests for User Story 2

- [ ] T025 [P] [US2] Extend `internal/task/task_test.go` covering `Complete` and its three distinct outcomes — changed, already complete, no such identifier — asserting they are distinguishable rather than collapsed into a boolean (FR-005, FR-007, FR-008; data-model.md)
- [ ] T026 [US2] Extend `cmd/pomotask/main_test.go` with the `done` success case, run as a **first-time-user journey**: starting from an empty `t.TempDir()` with no data file and no setup call of any kind, `add` then `done` in sequence, asserting stdout carries `Completed task N`, code 0, and a subsequent load shows the task complete. Starting from an empty directory rather than a seeded fixture is what makes this cover SC-006 — a fixture would hide any initialization step the tool secretly required (FR-005, FR-009, SC-003, SC-006)
- [ ] T027 [US2] Extend `cmd/pomotask/main_test.go` with the already-complete case: message on **stdout**, code **0**, and the stored data byte-identical to before (FR-008, FR-015; Gap G1)
- [ ] T028 [US2] Extend `cmd/pomotask/main_test.go` with the failure cases: an identifier matching no task, and a non-integer argument — each on stderr with code 1 and nothing changed (FR-007, FR-015, SC-005)

### Implementation for User Story 2

- [ ] T029 [US2] Implement `Complete` in `internal/task/task.go`, returning its three outcomes distinguishably. Doc comment stating the contract (FR-005, FR-007, FR-008)
- [ ] T030 [US2] Implement the `done` branch of `run` in `cmd/pomotask/main.go`, mapping each outcome to the stream, message, and exit status fixed in contracts/cli.md (FR-005 through FR-009)

**Checkpoint**: `pomotask add` and `pomotask done` both work and both persist.

---

## Phase 5: User Story 3 - Run a focus interval (Priority: P3)

**Goal**: A user runs a fixed 25-minute interval that reports its progress and signals its end.

**Independent Test**: Run the interval and observe the report sequence and exit status. Requires no recorded tasks and no data file — this story touches neither.

**Note**: This story depends on Phase 2 only. It can be built before US1 or US2 if preferred.

### Tests for User Story 3

- [ ] T031 [P] [US3] Write `internal/focus/focus_test.go` asserting a completed interval emits exactly 25 remaining-time reports, the first showing 25 and the last showing 1, none showing zero, followed by exactly one completion line (SC-007, FR-012)
- [ ] T032 [US3] Extend `internal/focus/focus_test.go` to run the interval at **two different compressed ticks** — say 1ms and 5ms — and assert two things: the emitted sequences are byte-identical, and each run **completes within a generous fixed bound** such as five seconds. Comparing against a production run is not an option; that would mean waiting 25 minutes, which is the cost FR-016 exists to remove.

  Together these establish what one source means. Identical sequences show the report count does not depend on the tick. Completion under the bound shows the total does: were the total hardcoded at 25 minutes while only the cadence compressed, the run would emit its reports and then hang past the bound rather than finishing.

  **Do not assert a ratio between the two elapsed times.** Windows default timer granularity is roughly 15.6ms, so both a 1ms and a 5ms tick round up to about the same real interval and a ratio assertion would measure the operating system's scheduler rather than this code. The bound above is an order-of-magnitude guard, deliberately loose, and carries no timing claim beyond "did not hang" (FR-016; research.md R6)
- [ ] T033 [US3] Extend `internal/focus/focus_test.go` with cancellation: cancel the context mid-interval and assert the elapsed figure is reported in whole minutes and no confirmation is sought. **Cancel the context directly — do not deliver a signal** (FR-013; research.md R7; Gap G2)
- [ ] T034 [P] [US3] Extend `cmd/pomotask/main_test.go` with the command-level focus behavior, driven at a compressed tick: a completed interval writes the **full 25-report sequence and the completion line to the stdout writer `run` was given** and returns 0; a cancelled one — cancelled **through the context the test itself passed to `run`**, never by delivering a signal — returns **1 specifically**, not merely non-zero, with its message also on stdout. T031 proves the focus package emits the sequence; this proves the command delivers it, which is the same distinction that puts the ADR behavioral check at T016 rather than inside the storage package (FR-012, FR-013, SC-007, SC-008; Gaps G3 and G5)
- [ ] T035 [US3] Extend `cmd/pomotask/main_test.go` asserting `focus` with any argument is **rejected** on stderr with code 1, not ignored. An ignored argument produces output identical to no argument and would prove nothing (FR-011, FR-015; contracts/cli.md)

### Implementation for User Story 3

- [ ] T036 [US3] Implement `internal/focus/focus.go` taking a `context.Context` and **exactly one** duration parameter, the tick, from which total length is derived as 25 × tick. **No second duration or count parameter may exist on the exported surface** — T032 rests on the mismatch it would allow being unrepresentable, not merely unused. Reports count remaining **ticks** rather than formatted wall-clock time, and the interval ends on cancellation. This package imports neither `os/signal` nor `internal/storage` (FR-010 through FR-014, FR-016; research.md R6; depguard rule 2)
- [ ] T037 [US3] Implement the `focus` branch of `run` in `cmd/pomotask/main.go`: forward the `ctx` and `tick` it received to the focus package, and write the interval's output to the `stdout` writer it was given — not to `os.Stdout` directly, which would leave T034 unable to observe it. **The branch creates no context of its own**; `signal.NotifyContext` belongs to `main()` under T009, and this branch cannot tell an interrupt from any other cancellation (FR-010, FR-012, FR-013; research.md R7)

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Confirming the enforcement mechanisms actually bite, and closing the Constitution's completion gate.

- [ ] T038 [P] Provoke depguard rule 1 — temporarily import `database/sql` in `internal/storage`, confirm `golangci-lint run` fails with a message naming ADR 0001, then remove it (quickstart.md)
- [ ] T039 [P] Provoke depguard rule 2 in `internal/task` — temporarily import `encoding/json`, confirm failure, remove it; repeat with `os`. Serialization and filesystem are separate denials and one passing does not imply the other (quickstart.md)
- [ ] T040 [P] Provoke depguard rule 2 in `internal/focus` — temporarily import `pomotask/internal/storage`, confirm failure, remove it; repeat with `os/signal`. The first is FR-014 as a build error, the second is what keeps signal handling in one place. This is also what covers the spec edge case about adding a task while an interval runs: the interval cannot reach task data, so the two cannot interfere (quickstart.md; FR-014)
- [ ] T041 Audit every exported identifier across all four packages for a doc comment describing its contract (Constitution Principle IV)
- [ ] T042 Audit every error path for wrapping with useful context, no silently discarded errors, and no `panic` on an expected failure (Constitution Principle IV)
- [ ] T043 Run the full quickstart.md walkthrough on Windows, including the manual 25-minute interval — the one check a compressed test cannot make (SC-004)
- [ ] T044 [P] Write `README.md` covering the three commands, where the data file lives on each platform, and **what the tool is not**. The word Pomodoro sets expectations this feature deliberately does not meet: there are no break intervals, no long breaks, no cycle counting, and no session history — `focus` is one fixed 25-minute interval and nothing more. Stating that plainly prevents the absences being read as unfinished work (spec Assumptions; Constitution Principle V)
- [ ] T045 Run the complete gate — gofmt output empty, `go vet ./...`, `golangci-lint run`, `go test ./...` — and confirm all pass (Constitution Development Workflow & Quality Gates)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies. T003 blocks T004 — the config schema cannot be chosen before the version is known
- **Foundational (Phase 2)**: Depends on Setup. Blocks all three stories
- **US1 (Phase 3)**: Depends on Foundational only
- **US2 (Phase 4)**: Depends on Foundational, and on T022 and T023 for the `Task` type and the storage package — plus the coverage in T014, T015, and T021, which exercise storage and failure-path behavior US2 relies on equally. Its own behavior is independent of US1: completing a task neither requires the add path nor disturbs it
- **US3 (Phase 5)**: Depends on Foundational only. Shares nothing with US1 or US2 — no task data, no storage package
- **Polish (Phase 6)**: T038 needs `internal/storage` to exist; T039 needs `internal/task`; T040 needs `internal/focus`. T043, T044, and T045 need all three stories complete — the README cannot describe commands that do not yet behave as documented

### Within Each User Story

- Tests are written before the implementation they cover and are expected to fail first
- `internal/task` before `internal/storage`, because storage converts to and from the domain type
- Both before the `run` branch that orchestrates them

### Parallel Opportunities

- T012 and T013 touch different files and can run together
- T022 (`internal/task`) is a prerequisite for T023, so those are sequential despite being different files
- T031 and T034 are in different files and can run together
- T038, T039, and T040 touch three different packages and can run together
- Tasks extending the same file — the several `main_test.go`, `store_test.go`, and `focus_test.go` entries — are sequential by definition and carry no `[P]`

### Story Independence

US3 is genuinely independent: FR-014 forbids the interval from reading or writing task data, and depguard rule 2 enforces it. A team could build US3 first, or in parallel with US1, without coordination.

---

## Parallel Example: User Story 1

```text
# Different files, no shared dependency:
T012  internal/task/task_test.go
T013  internal/storage/store_test.go
```

---

## Implementation Strategy

### MVP First

1. Phase 1: Setup — module, both depguard rules, CI
2. Phase 2: Foundational — the `run` seam
3. Phase 3: User Story 1
4. **Stop and validate**: add a task, start a new shell, add another, confirm the identifier advanced
5. The tool is useful at this point as a capture tool

### Incremental Delivery

Setup and Foundational, then US1 (MVP), then US2, then US3. Each story is separately
demonstrable and none breaks the ones before it.

### Order note

The user-specified sequence — module, linter version check, config, CI, then implementation — is
Phase 1 as written. Configuring enforcement before writing the code it governs means the first
violation is caught the moment it appears, rather than being discovered in a later audit of code
already believed finished.

---

## Notes

- `[P]` means different files with no incomplete dependency, nothing looser
- Every task cites its authorizing requirement or ADR clause; a task that cannot be traced is a decision nobody approved and should be raised rather than implemented
- Verify tests fail before implementing
- Commit after each task or logical group
- The five Gap register entries in plan.md are choices the developer may overturn. Each is encoded by exactly one task, so reversing a gap identifies its test directly:

  | Gap | Choice | Encoded by |
  |-----|--------|-----------|
  | G1 | already-complete exits 0 | T027 |
  | G2 | elapsed reported in whole minutes | T033 |
  | G3 | interruption exits 1 rather than 130 | T034 |
  | G4 | `add` takes exactly one argument | T020 |
  | G5 | interruption prints to stdout | T034 |

  T035 is **not** in this table: rejecting arguments to `focus` is FR-011, a requirement, not a choice open to reversal

- One coverage gap remains open by decision rather than oversight: no task asserts behavior when two task-modifying commands run at the same instant. The spec states outright that this is unspecified, and ADR 0001 accepts lost updates under concurrent writes as a consequence of the chosen format
