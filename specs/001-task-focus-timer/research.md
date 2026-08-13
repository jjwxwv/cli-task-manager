# Phase 0 Research: Task Manager with Focus Timer

**Feature**: `001-task-focus-timer` | **Date**: 2026-08-14

Every decision below cites the specification requirement or the Accepted ADR that authorizes
it, as Constitution Principle III requires. Two of them — R1 and R2 — are the decisions
[ADR 0001](../../adr/0001-persist-tasks-to-local-json.md) explicitly defers to planning.
Decisions resting on convention rather than on an authorizing document are marked as such.

---

## R1. Where the task data file lives

**Authority**: ADR 0001 — "The exact local file path is an implementation or configuration
detail and is not decided by this ADR." Constitution — persistence tests MUST use isolated
temporary data locations.

**Decision**: The persistence package accepts the file path as a construction parameter. It
resolves nothing on its own. `main` computes the default path as
`os.UserConfigDir()` + `/pomotask/tasks.json`; tests pass `t.TempDir()`.

**Rationale**: Passing the path in is what makes the ADR's own enforcement test possible — it
requires invoking the add-task path "against a temporary directory" — without introducing any
user-facing configuration. An environment variable or flag would also isolate tests, but it
would add a configuration surface no requirement asks for, which Principle V forbids. A
parameter costs nothing and is invisible to users.

`os.UserConfigDir()` is stdlib, resolves per-platform without hand-written branching
(`%AppData%` on Windows, `~/Library/Application Support` on macOS, `$XDG_CONFIG_HOME` or
`~/.config` on Linux), and therefore keeps persistence inside the standard library as ADR 0001
requires.

**Alternatives considered**:

- `os.UserHomeDir()` + a dotted directory. Also stdlib, but ignores platform convention on
  Windows and macOS.
- Hand-rolled XDG_DATA_HOME resolution. Semantically the better fit — tasks are user data, not
  configuration — but the standard library ships no `UserDataDir`, so this means writing
  platform branching by hand for a distinction no user will observe. Rejected under
  Principle V.

**Known imperfection**: task records are data stored under a configuration directory. Recorded
here rather than papered over. It buys stdlib-only resolution, which ADR 0001 does require.

---

## R2. What happens when stored data cannot be read

**Authority**: ADR 0001 offers three behaviors — fail loudly, quarantine the file, or restore
from a retained copy — and assigns the choice to planning. Spec edge case: the system MUST NOT
silently start from an empty set, discard the user's data, or report success. FR-015, SC-005.

**Decision**: Fail loudly. The command reports what was wrong with the file, changes nothing,
and exits non-zero. The data file is left exactly as found.

**Rationale**: This is the only one of the three that needs no machinery beyond what other
requirements already demand. Restoring from a retained copy requires maintaining a retained
copy — an entire backup mechanism no requirement asks for. Quarantining requires moving the
user's file, which is itself a mutation the spec never authorizes, performed at the moment the
system understands the data least. Failing loudly adds nothing and satisfies the spec's
guarantee directly.

It also unifies two paths: ADR 0001 already mandates that an unrecognized `schema_version` is a
load-time failure. Choosing fail-loud for malformed data means one behavior covers both, rather
than two shapes of response to two shapes of unreadable file.

**Consequence for SC-005**: SC-005 deliberately withheld malformed data from its list of failed
operations pending this decision. With fail-loud chosen, malformed data now *is* an operation
the system cannot complete, and SC-005 covers it through its general clause.

---

## R3. Identifier scheme

**Authority**: FR-006 — unique across recorded tasks, stable once disclosed, system-assigned.
FR-003 — printed in the add confirmation.

**Decision**: A positive integer, assigned as one greater than the largest identifier present,
starting at 1.

**Rationale**: Short enough to read off a confirmation line and retype, which FR-003 makes the
only way a user ever obtains one. Uniqueness holds by construction. Stability holds because
nothing in scope removes a task, so no identifier is ever freed for reuse.

**Alternatives considered**:

- UUID. Unique, but nothing requires uniqueness beyond this one file, and a user who must
  retype it from a scrollback line is poorly served. Rejected under Principle V.
- A persisted `next_id` counter. Equivalent in behavior to max-plus-one while nothing is
  removed, at the cost of an extra persisted field. Rejected as unnecessary.

**Dependency to record**: max-plus-one is safe only because deletion is out of scope. If
deletion is ever added, this scheme must be revisited — max-plus-one would still avoid reuse,
but only if the largest identifier survives, which deletion does not guarantee.

---

## R4. Reading `schema_version` before decoding records

**Authority**: ADR 0001 — "`schema_version` must be read and checked before task records are
decoded. An unrecognized value is a load-time failure, not a value to be ignored."

**Decision**: Read the file once into memory. Unmarshal into a struct carrying only
`schema_version`. If it is not 1, fail with a message naming the version found and the version
supported. Only then unmarshal the same bytes into the full document.

**Rationale**: Two passes over bytes already in memory is the simplest way to satisfy the
ordering the ADR mandates. Version 1 is the initial format.

---

## R5. How writes are performed

**Authority**: ADR 0001 — create a temporary file in the destination's directory with
`os.CreateTemp`, serialize into it, then replace the destination with `os.Rename`.

**Decision**: Exactly as mandated. The temporary file is created in the destination directory,
never in the system temp directory, so the rename stays within one filesystem.

**Rationale**: Mandated, not chosen. Recorded so the corresponding task is traceable.

**Scope note**: ADR 0001 states that Go does not guarantee atomic `os.Rename` on all non-Unix
platforms and places cross-platform crash durability out of scope. No requirement or test
covers crash durability, and none will be written.

---

## R6. Making the interval testable without waiting 25 minutes

**Authority**: FR-016, first seam — duration and reporting cadence MUST be driven from one
source, and the seam MUST NOT be reachable from the user-facing surface FR-011 bounds.
FR-012, SC-007. Not SC-004: FR-016 excludes it, because compressing the interval removes the
very wall-clock property SC-004 asserts.

**Decision**: The interval is defined as 25 ticks. The tick duration is the single parameter.
Total length is derived as `25 × tick`, never supplied separately. Production passes
`time.Minute`. Tests pass a small value such as `10 * time.Millisecond`.

Reports count down remaining **ticks**, not wall-clock time: the sequence is "25 minutes
remaining" through "1 minute remaining" regardless of the tick duration in force.

**Rationale**: Deriving total from tick makes FR-016's "one source" a structural property
rather than a discipline someone must remember. It is impossible to compress the duration
without compressing the cadence, because there is only one number.

Counting ticks rather than formatting elapsed wall time is what keeps SC-007 verifiable. Were
the report to format actual remaining duration, a compressed test interval would print
milliseconds and the assertion "the first report shows 25 minutes remaining" could not be
checked at speed — precisely the untestable state FR-016 exists to prevent.

**Alternatives considered**:

- Separate `duration` and `interval` parameters. Permits the exact mismatch FR-016 names as a
  trap: a shortened duration still reporting once per real minute, leaving a total-length check
  green while FR-012's sequence goes untested. Rejected.
- An injected clock interface. More general, and none of that generality is required. Rejected
  under Principle V — the tick parameter is sufficient.

---

## R7. Interruption and exit status

**Authority**: FR-016, second seam — an interrupted interval MUST be reachable by cancelling
something a test can hold, not only by signalling the process. FR-013 — print elapsed time,
exit non-zero, no confirmation prompt. SC-008. FR-012 —
exit zero on completion. SC-008.

**Decision**: `signal.NotifyContext` with `os.Interrupt` produces a context cancelled on Ctrl-C.
The wiring lives in `func main()` and nowhere else. Both `run` and `internal/focus` accept a
`context.Context` and end when it is cancelled, neither importing `os/signal` — a constraint
depguard rule 2 enforces on the focus package rather than leaving it to discipline. Neither
learns why cancellation happened, which is what lets a test cancel by ordinary function call
instead of by signalling its own process. On cancellation the command prints the elapsed time
and returns exit code 1. Normal completion returns 0. `main` calls `os.Exit` at exactly one
place, so no deferred cleanup is skipped.

**Rationale**: `signal.NotifyContext` is stdlib and removes the need to manage a signal channel
by hand. Keeping it out of the interval package means interval tests cancel a context directly —
deterministic — rather than delivering a signal to a test process, which is not. Exit code 1 is
chosen over the shell convention of 130 because SC-008 requires only that completion and
interruption be distinguishable, and 1 is the smaller commitment.

**Convention, not requirement**: nothing specifies 1 in particular. Recorded here so a later
change to 130 is understood as a free choice rather than a correction.

---

## R8. Shape of the persistence boundary

**Authority**: ADR 0001 — persistence logic behind a dedicated package; domain logic MUST NOT
import `encoding/json` or touch the filesystem. Constitution Principle V — interfaces MUST NOT
be introduced for hypothetical implementations, but MAY define a required architectural
boundary.

**Decision**: No interface. The domain package exposes pure functions over a collection of
tasks. The storage package exposes load and save. `main` loads, applies the domain operation,
and saves.

**Rationale**: The boundary ADR 0001 requires is a *package* boundary, and it is enforced by
depguard at build time, not by an interface at runtime. Adding one would introduce indirection
with a single implementation on either side and nothing that varies. Principle V permits an
interface here; it does not call for one, and nothing in the design needs it.

The domain package under this arrangement imports neither `encoding/json` nor `os`, which is
the constraint ADR 0001 actually states.

---

## R9. Persisted record type separate from the domain type

**Authority**: ADR 0001 — domain logic MUST NOT import `encoding/json`.

**Decision**: The storage package declares its own record type carrying the JSON struct tags
and converts to and from the domain type at the boundary.

**Rationale**: Tagging the domain type would not by itself import `encoding/json`, so it would
satisfy the letter of the rule while placing the serialized shape inside the domain — which is
what the rule exists to prevent. A separate record type keeps the persisted format changeable
without touching domain code, at the cost of a short conversion in one file.

---

## R10. Argument parsing and output streams

**Authority**: FR-001, FR-005, FR-010 — each behavior invoked in a single command. FR-015,
SC-005 — every failure reported.

**Decision**: Dispatch on `os.Args` directly. No flags exist, so the `flag` package is not
used. An unrecognized or malformed invocation prints a usage message and exits non-zero.
Normal output goes to stdout, failure messages to stderr.

**Rationale**: The three commands take a positional argument or none. Introducing a flag parser
for a CLI with no flags is complexity Principle V rejects. Reporting an unusable invocation is
required by FR-015, which makes the usage message a requirement rather than a convenience.

**Convention, not requirement**: no requirement assigns failures to stderr. It is the universal
CLI convention and keeps failure text out of piped output. Recorded as a choice.

---

## Open items carried into implementation

None. No `NEEDS CLARIFICATION` remains in the Technical Context, and both decisions ADR 0001
deferred are settled above.
