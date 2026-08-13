# 1. Persist tasks to a local JSON file

Date: 2026-08-13

## Status

Accepted

## Context

We are building a greenfield CLI Task Manager & Focus Timer. The required behavior is limited to adding tasks, marking tasks as complete, and running a fixed 25-minute focus interval. The persistence strategy is not prescribed by the requirements and must be selected and recorded before implementation begins.

Persistence is an architecturally significant requirement here: it determines how task state survives between CLI invocations, where validation responsibility sits, and how easily the delivered implementation can be checked against the documented architecture.

This project prioritizes architectural clarity, simple operation, and clear alignment between the ADR, specification, and generated source code. The project is maintained by one developer, with an AI coding agent performing implementation under specification-driven constraints. This shapes the decision in two specific ways. Because there is no second engineer to absorb ongoing specialist maintenance such as schema migration or index tuning, operational simplicity is a staffing constraint rather than a preference. And because the owner's role is to govern an AI-generated implementation rather than to author it, a format the owner can read directly makes verification a matter of inspection instead of trust.

Go has been selected as the implementation language. The developer is proficient in Go, which reduces the risk of review errors in code the developer did not write. Go compiles the application into a standalone executable, which suits a CLI distributed as a tool rather than operated as a service. By keeping the persistence layer within the Go standard library and avoiding cgo-dependent persistence packages, the project minimizes runtime and deployment dependencies. Go also resolves package imports at compile time and has no general-purpose runtime import mechanism, making dependency-based architectural constraints strongly enforceable at build time.

Because workload and concurrency are not specified, this decision uses the following assumptions:

- The application is used locally by one interactive user.
- Task commands run as separate CLI process invocations.
- The task dataset is small enough for whole-file reads and writes to remain acceptable for interactive use.
- Concurrent writes are uncommon and are not a current requirement.
- No external database service is required.

The following persistence options were considered:

**In-memory storage.** Simple, but state is lost when the process exits. It does not satisfy the need to keep tasks available across separate CLI invocations.

**SQLite.** Provides transactions, indexes, relational constraints, and stronger concurrent-access support. These capabilities are useful for larger or more relational data models, but they introduce additional structure that the current requirements do not need. In Go, SQLite also requires a database driver that is not provided by the standard library. A cgo-based driver may introduce native build dependencies, while a pure-Go driver still introduces a third-party dependency. Either choice gives up the standard-library-only persistence constraint adopted by this project.

**Local JSON file.** Human-readable, easy to inspect, requires no database engine, and is sufficient for a small local dataset. Its limitations are whole-file reads and writes, weak concurrent-write support, and application-managed validation and schema compatibility.

Given the current scope and assumptions, Local JSON provides the best balance of simplicity, transparency, and sufficient persistence capability.

## Decision

Persist task state to a single local JSON file.

Use the standard `encoding/json` package for serialization and the standard `os` package for file management. No module outside the Go standard library participates in persistence.

The persisted document must contain:

- `schema_version`: an integer identifying the persisted data format.
- `tasks`: an array containing the persisted task records.

`schema_version` must be read and checked before task records are decoded. An unrecognized value is a load-time failure, not a value to be ignored.

All persistence logic must remain behind a dedicated persistence package. Domain logic must not import `encoding/json` and must not perform filesystem operations directly.

No database engine, database driver, ORM, key-value store, or remote storage service is part of the accepted persistence architecture.

Writes must be performed by creating a temporary file in the same directory as the destination using `os.CreateTemp`, serializing into it, and then replacing the destination with `os.Rename`. Keeping both files in the same directory avoids cross-filesystem rename issues. On Unix-like platforms this provides atomic replacement semantics; Go does not guarantee atomic `os.Rename` behavior on all non-Unix platforms. Cross-platform crash durability is outside the current scope.

The exact local file path is an implementation or configuration detail and is not decided by this ADR.

## Consequences

### What becomes easier

- Task data is human-readable and easy to inspect during development, testing, and review.
- No database server, database driver, ORM, or migration framework is required.
- Persistence uses only the Go standard library, minimizing runtime and deployment dependencies.
- The persistence boundary is simple and easy to verify against the documented architecture. Static dependency checks can reliably prevent prohibited persistence packages from being introduced.
- Simple additive data changes can be handled for backward compatibility by applying zero values or explicit defaults when older records are read.

### What becomes more difficult

- Each update may require reading and rewriting the complete JSON file.
- Concurrent processes may cause lost updates.
- JSON provides no database-managed transactions, indexes, relational constraints, or query optimization.
- The application must handle malformed or incompatible JSON.
- The persisted shape is defined by Go struct definitions, so `encoding/json` silently discards fields present in the file but absent from the struct. A document written by a newer build can therefore lose those fields if an older build reads and rewrites it. Forward compatibility is not automatic, and the `schema_version` check converts an unsupported format into an explicit load-time failure.
- Incompatible format changes may require explicit compatibility or migration logic written by hand.

### Subsequent decisions now required

Choosing an application-managed file format without a database-enforced schema transfers several responsibilities to the application that a database engine would otherwise carry. The following decisions were not applicable before this ADR and must be settled before the implementation is complete:

- **Data-file location** — whether to honor `$XDG_DATA_HOME` and platform-specific conventions, or to use a fixed path. This does not change the structure of the system and is recorded as a configuration decision rather than a separate ADR.
- **Corruption and recovery behavior** — how malformed, externally edited, or version-mismatched JSON is detected, and whether the application fails loudly, quarantines the file, or reconstructs from a retained copy. This determines what the persistence boundary guarantees to its callers, so it warrants its own decision record if the chosen behavior turns out to be non-obvious; otherwise it is specified as implementation behavior in the plan.

One further decision becomes necessary only if the scope grows: whether completed focus sessions are persisted, and if so whether they share this file. That would introduce a second entity into a format chosen for a single flat collection, and is therefore deferred rather than pre-decided here.

### Enforcement

A single CI fitness function asserts two invariants and fails the build if either is violated.

The negative invariant is that the persistence package introduces no alternative backend. This is enforced declaratively with `depguard` under `golangci-lint`, configured as an allowlist that permits only the Go standard library (`$gostd`) within the persistence package, with explicit deny entries for `database/sql`, SQLite drivers, and ORM packages. Each deny entry carries a description naming this ADR, so a violation reports the governing decision rather than an anonymous lint error. An allowlist is used in preference to a blocklist because it closes every dependency path not explicitly opened.

The positive invariant is that the running application actually persists through the approved JSON implementation. This is checked behaviorally by a Go test that invokes the add-task path against a temporary directory and asserts that the resulting file decodes through `encoding/json` into a document carrying `schema_version` and `tasks`. The static check prevents prohibited persistence dependencies from being introduced; the behavioral check verifies that the approved JSON path is actually used.

Any future change of persistence strategy must therefore first be documented by a superseding ADR, since the build will otherwise reject it.

### Review

Independently of any trigger below, review this ADR one month after implementation, comparing its recorded assumptions against observed usage — actual dataset size, measured command latency, and whether concurrent access occurred in practice.

Revisit it sooner if any assumption in the Context becomes invalid, or if concurrency, remote synchronization, relational queries, large datasets, or stronger transactional guarantees become requirements.

If the persistence strategy changes, preserve this ADR as historical context and record the replacement decision in a new ADR rather than modifying this decision retroactively.
