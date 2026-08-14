# Phase 1 Data Model: Task Manager with Focus Timer

**Feature**: `001-task-focus-timer` | **Date**: 2026-08-14

Two representations of the same information, deliberately kept apart:
the **domain type**, which carries meaning, and the **persisted record**, which carries
format. [ADR 0001](../../adr/0001-persist-tasks-to-local-json.md) requires the separation —
domain logic must not import `encoding/json` — and [research.md](research.md) R9 records why
struct tags on the domain type would satisfy the letter of that rule while defeating its
purpose.

---

## Domain: Task

Lives in `internal/task`. Imports neither `encoding/json` nor `os`.

| Field | Type | Meaning |
|-------|------|---------|
| `ID` | `int` | System-assigned identifier the user types to name this task |
| `Text` | `string` | The descriptive text the user supplied |
| `Done` | `bool` | Whether the task has been marked complete |

**Source**: spec Key Entities; FR-006 (identifier), FR-001 (text), FR-005 (completion).

There is no created-at, priority, due date, tag, or ordering field. Nothing in scope reads
them, and Principle V forbids adding them against future need.

### Invariants

| Invariant | Authority |
|-----------|-----------|
| `ID` is unique across all recorded tasks | FR-006 |
| `ID` never changes once disclosed to the user | FR-006 |
| `ID` ≥ 1 | research.md R3 |
| `Text` is non-empty after trimming surrounding whitespace | FR-002 |

### Assignment of `ID`

One greater than the largest `ID` present, or 1 when no tasks exist. Safe against reuse only
because deletion is out of scope — see research.md R3, which records this dependency so that
adding deletion later cannot silently break identifier stability.

### State transitions

```text
        add                    complete
  ∅ ──────────► Done = false ──────────► Done = true
                     │                        │
                     │                        │ complete again
                     │                        ▼
                     │              reported as no-op, unchanged (FR-008)
                     │
                     └── no transition removes a task; deletion is out of scope
```

`Done` moves in one direction only. Nothing in scope reverses it, and no requirement asks for
reversal.

---

## Domain operations

Pure functions over a `[]Task`. They neither read nor write files.

| Operation | Behavior | Authority |
|-----------|----------|-----------|
| `Add(tasks, text)` | Returns the extended collection and the new task. Rejects empty or whitespace-only text. | FR-001, FR-002, FR-003 |
| `Complete(tasks, id)` | Marks the identified task complete. Distinguishes "no such identifier" from "already complete". | FR-005, FR-007, FR-008 |

`Complete` must report its three outcomes distinctly — changed, already complete, not found —
because FR-007 and FR-008 require different messages and neither may report success. Returning
a single boolean would collapse two of them.

---

## Persisted document

Lives in `internal/storage`. This is the only package that imports `encoding/json`.

```json
{
  "schema_version": 1,
  "tasks": [
    { "id": 1, "text": "write the report", "done": false }
  ]
}
```

| Field | Type | Notes |
|-------|------|-------|
| `schema_version` | integer | Must be read and checked **before** `tasks` is decoded |
| `tasks` | array | Persisted task records; empty array when no tasks exist |

**Source**: ADR 0001 mandates both fields and the ordering of the version check. Version 1 is
the initial format.

### Load rules

| Condition | Behavior | Authority |
|-----------|----------|-----------|
| File does not exist | Empty task set, not an error | Spec edge case (first use) |
| `schema_version` ≠ 1 | Fail, naming the version found and the version supported | ADR 0001 |
| File present but not decodable | Fail, leaving the file untouched | research.md R2 |
| Decode succeeds | Convert records to domain tasks | — |

An absent file and an unreadable file are different situations and must not share a code path.
The first is ordinary; the second must never be mistaken for it, which is exactly what the spec
edge case forbids.

### Save rules

Serialize to a temporary file created by `os.CreateTemp` **in the destination's own directory**,
then replace the destination with `os.Rename`. Mandated verbatim by ADR 0001; see research.md
R5, including the recorded scope limit on cross-platform crash durability.

---

## The focus interval holds no state

The interval is not an entity and nothing about it is persisted. It exists for the life of the
command that runs it. FR-014 forbids it from reading or writing task data at all, which is what
lets User Story 3 be tested without any task fixture.

Its one duration parameter is the tick; the total length is derived as `25 × tick`. That the
total is derived rather than supplied is what satisfies FR-016's timing seam — see research.md
R6. The interval also accepts a `context.Context`. That is one half of FR-016's cancellation
seam; the other half is the context `run` accepts, and SC-008 depends on that one, since an exit
status exists only at the command level. See research.md R7.
