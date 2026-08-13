# Feature Specification: Task Manager with Focus Timer

**Feature Branch**: `001-task-focus-timer`

**Created**: 2026-08-14

**Status**: Draft

**Input**: User description: "A single-user CLI task manager with a Pomodoro focus timer, implemented in Go. Scope is limited to exactly three behaviors: (1) add a task, (2) mark a task as complete, (3) run a fixed 25-minute focus interval. No list, delete, edit, pause, or resume commands are in scope. Task state persists across separate CLI process invocations. Do not decide open questions silently — surface as clarification gaps: how the user identifies a task when marking it complete given there is no list command, what happens when the focus interval completes, and what happens if the focus interval is interrupted."

## Clarifications

### Session 2026-08-14

- Q: When the focus interval reaches its full duration, how should the system signal that to the user? → A: A printed message only, with no audible alert.
- Q: When the user interrupts a focus interval before the full duration elapses, what should the system do? → A: Print the elapsed time and exit with a non-zero status, without prompting for confirmation.
- Q: How often should the system report the time remaining while a focus interval runs? → A: Once per minute, each report printed on a new line.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Capture a task (Priority: P1)

A person working alone wants to record something they need to do, without leaving the
command line and without losing it when they close the terminal. They issue a single
command with the task text and receive immediate confirmation that it was recorded.

**Why this priority**: Nothing else in the feature has value without captured tasks. This
is the smallest slice that delivers value on its own — a durable place to put work items.

**Independent Test**: Add a task, end the session, start a new session, and confirm the
task is still recorded. Delivers value as a capture tool even if no other story ships.

**Acceptance Scenarios**:

1. **Given** no tasks have been recorded yet, **When** the user adds a task with the text
   "write the report", **Then** the system confirms the task was recorded and prints the
   identifier assigned to it.
2. **Given** a task was added in an earlier session, **When** the user starts a new
   session, **Then** the previously added task is still recorded.
3. **Given** the user supplies no task text, **When** they attempt to add a task,
   **Then** the system reports the problem and records nothing.

---

### User Story 2 - Mark a task complete (Priority: P2)

Having finished a piece of work, the person marks the corresponding task as complete so
that their recorded state matches reality.

**Why this priority**: Completion is only meaningful once tasks exist, so it depends on
Story 1. It is the second half of the capture-and-close loop.

**Independent Test**: With one task already recorded, mark it complete, then start a new
session and confirm the completion persisted.

**Acceptance Scenarios**:

1. **Given** a recorded task that is not complete, **When** the user marks it complete by its
   identifier, **Then** the system confirms the change and the task is recorded as complete.
2. **Given** a task was marked complete in an earlier session, **When** the user starts a
   new session, **Then** the task is still recorded as complete.
3. **Given** the user supplies an identifier matching no recorded task, **When** they attempt
   to mark it complete, **Then** the system reports that no such task was found and changes
   nothing.
4. **Given** a task that is already complete, **When** the user marks it complete again by its
   identifier, **Then** the system reports the task is already complete and changes nothing.

---

### User Story 3 - Run a focus interval (Priority: P3)

The person wants a bounded, uninterrupted stretch of work. They start a focus interval and
work until it ends, using the command line itself as the timer.

**Why this priority**: Independently valuable and entirely separate from task records — it
neither reads nor writes tasks — so it can ship before or after the other stories.

**Independent Test**: Start a focus interval and confirm it runs for the fixed duration and
signals its end. Requires no recorded tasks.

**Acceptance Scenarios**:

1. **Given** the user is at the command line, **When** they start a focus interval,
   **Then** the system prints 25 minutes remaining, and repeats that report on a new line at
   each subsequent elapsed minute for as long as time remains, the last of them showing 1
   minute remaining.
2. **Given** a focus interval is running, **When** the full duration elapses, **Then** the
   system prints a message announcing that the interval has ended.
3. **Given** a focus interval is running, **When** the user interrupts it before the
   duration elapses, **Then** the system prints the elapsed time, exits with a non-zero
   status, and does not prompt for confirmation.

---

### Edge Cases

- What happens on the very first use, when no task data has ever been recorded? The system
  MUST treat this as an empty task set, not as an error.
- How does the system handle recorded task data written in a format this version does not
  recognize? The system MUST report the problem and stop rather than proceeding.
- How does the system handle recorded task data that is present but malformed? The system
  MUST NOT silently start from an empty set, discard the user's data, or report success.
  Which recovery behavior applies beyond that guarantee — reporting and stopping, setting
  the unreadable data aside, or restoring from a retained copy — is deliberately left open
  here and is settled during planning.
- What happens when two tasks are added with identical text? Their identifiers differ, so
  each remains separately addressable and completing one does not affect the other.
- What happens when the user no longer has a task's identifier, because the confirmation from
  FR-003 has scrolled out of the terminal or the session was closed? No command in scope
  recovers it, and none is required to. Recovering a lost identifier lies outside what this
  feature offers. This is an accepted consequence of excluding a list command from scope,
  recorded here so it is not mistaken for a defect.
- What happens if the user adds a task while another session is running a focus interval?
  The focus interval does not read or write task data, so the two MUST NOT interfere.
- What happens if the storage location cannot be written to (missing permissions, full
  disk)? The system MUST report the failure rather than reporting a false success.

## Requirements *(mandatory)*

### Functional Requirements

**Adding tasks**

- **FR-001**: Users MUST be able to record a new task by supplying its descriptive text in
  a single command.
- **FR-002**: System MUST reject an attempt to add a task with empty or whitespace-only
  text, report the reason, and record nothing.
- **FR-003**: System MUST confirm a successful add, and that confirmation MUST carry the
  identifier assigned under FR-006. With no list command in scope, this confirmation is the
  only occasion on which the system discloses that identifier, and therefore the user's only
  opportunity to record it.
- **FR-004**: System MUST retain recorded tasks so that they remain available in later,
  separate command-line sessions.

**Completing tasks**

- **FR-005**: Users MUST be able to mark an existing task as complete in a single command.
- **FR-006**: System MUST assign every recorded task an identifier, unique across recorded
  tasks and stable once disclosed, and users MUST name a task by that identifier when marking
  it complete. The identifier is assigned by the system, never supplied by the user.
- **FR-007**: System MUST report an error and change nothing when the user names a task
  that does not exist.
- **FR-008**: System MUST report that no change was needed, and change nothing, when the
  named task is already complete.
- **FR-009**: System MUST retain completion state so that it remains in effect in later,
  separate command-line sessions.

**Focus interval**

- **FR-010**: Users MUST be able to start a focus interval of exactly 25 minutes in a
  single command.
- **FR-011**: System MUST NOT expose any user-facing means of changing the 25-minute
  duration — no command argument, flag, environment variable, or configuration file. This
  bounds the user-facing surface only; FR-016 governs what may exist behind it.
- **FR-012**: System MUST print the time remaining when the interval starts and once at each
  subsequent elapsed minute for as long as time remains, each report on its own line. The
  final elapsed minute produces no such report; on reaching the full duration the system MUST
  instead print a message announcing that the interval has ended, and MUST exit with a zero
  status. The system MUST NOT emit an audible alert.
- **FR-013**: When the interval is interrupted before the full duration elapses, the system
  MUST print how much time had elapsed and MUST exit with a non-zero status, distinguishing
  an interrupted interval from the completion described in FR-012. The system MUST NOT prompt
  for confirmation before stopping.
- **FR-014**: The focus interval MUST NOT read or modify recorded task data.

**Error reporting**

- **FR-015**: System MUST report every failure with a message that names what went wrong,
  and MUST NOT report success when an operation did not take effect.

**Verifiability**

- **FR-016**: System MUST be verifiable against FR-012, SC-004, and SC-007 without a test
  waiting out the full 25 minutes. Any internal seam serving that purpose MUST drive the
  interval's duration and its reporting cadence from one source, so that compressing the
  duration compresses the cadence with it. A seam that shortened only the duration would leave
  a compressed interval still reporting once per real minute, exhibiting neither the sequence
  FR-012 describes nor the count SC-007 fixes, while SC-004 continued to pass. The seam is
  internal and MUST NOT be reachable through the user-facing surface FR-011 bounds.

### Key Entities

- **Task**: A single unit of work the user has recorded. Carries the descriptive text the
  user supplied, whether it is complete, and the system-assigned identifier by which the user
  names it. That identifier MUST be unique across recorded tasks and MUST remain stable once
  disclosed, because it is the only means of naming a task after the session that created it
  has ended. Tasks are independent of one another; there is no grouping, ordering, priority,
  or due date in scope.

The focus interval is not a recorded entity. It exists only for the duration of the command
that runs it, and nothing about it is retained afterward.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can record a new task in one command and see confirmation in under one
  second.
- **SC-002**: 100% of tasks recorded in one session are still present in a later session
  started after the first has fully ended.
- **SC-003**: A user holding a task's identifier can mark that task complete in one command,
  and the completion survives into later sessions in 100% of cases.
- **SC-004**: A focus interval runs for 25 minutes, accurate to within one second, and
  signals its end without the user having to check the clock.
- **SC-005**: Every operation the system rejects or cannot complete produces a message
  stating what went wrong, with zero silent failures and zero success reports for an
  operation that did not take effect. This holds for every failure the system can reach.
  Empty task text, an unknown task, task data in an unrecognized format, and unwritable
  storage are instances of it, not the limit of it. Malformed task data is deliberately not
  listed: whether it constitutes a failed operation depends on the recovery behavior settled
  during planning, and this criterion does not presume the answer.
- **SC-006**: A first-time user with no prior data can record and complete a task without
  any setup step beyond running the commands.
- **SC-007**: A focus interval run to completion produces exactly 25 time-remaining reports:
  the first when it starts, showing 25 minutes remaining, and one at each subsequent elapsed
  minute through to the last, which shows 1 minute remaining. Exactly one message announcing
  that the interval has ended follows, and no report shows zero minutes remaining.
- **SC-008**: A completed focus interval and an interrupted one are distinguishable by exit
  status in 100% of runs, the completed one exiting zero and the interrupted one non-zero.

## Assumptions

- A single person uses the tool on one machine. Multi-user access, sharing, and
  synchronization between machines are out of scope.
- Each command is a separate, short-lived command-line invocation; no background service or
  long-running process is involved, other than the focus interval command itself blocking
  for its duration.
- Concurrent invocations are uncommon and are not a requirement. Behavior when two
  task-modifying commands run at the same instant is not specified.
- The volume of tasks stays small enough that no search, filtering, or paging is needed —
  consistent with the absence of a list command in scope.
- FR-006 was not chosen from open options; it follows from constraints this specification
  already imposes. Naming a task by its descriptive text cannot distinguish two tasks that
  share text. Naming it by ordinal position is inoperable, because an ordinal is a position
  within a collection and no command in scope displays that collection, leaving no occasion
  on which a user could learn one. A system-assigned identifier is therefore the only form
  satisfying the uniqueness and stability this specification requires, and keeping that
  identifier once printed falls to the user, since nothing in scope will reproduce it.
- Reporting progress during the focus interval is treated as a baseline expectation rather
  than a decision needing approval: a command that blocks for 25 minutes with no output is
  indistinguishable from a hung program.
- Where recorded tasks are stored, how unreadable data is detected, and which recovery
  behavior applies to malformed data are decisions governed by ADR 0001 and settled during
  planning, not by this specification.
- No notification, reporting, history, or statistics capability is implied by the focus
  interval. Whether completed intervals are ever recorded is explicitly deferred by
  ADR 0001 and is out of scope here.
