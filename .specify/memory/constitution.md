<!--
Sync Impact Report
==================
Version change: [CONSTITUTION_VERSION] (unfilled template) → 1.0.0
Bump rationale: MAJOR — initial ratification. The prior file was the unmodified
scaffold with no adopted governance, so this is the first binding version.

Modified principles:
- [PRINCIPLE_1_NAME] → I. ADR Authority (NON-NEGOTIABLE)
- [PRINCIPLE_2_NAME] → II. Approved Persistence Architecture (NON-NEGOTIABLE)
- [PRINCIPLE_3_NAME] → III. Traceable Decisions
- [PRINCIPLE_4_NAME] → IV. Standard-Library-First Code Quality
- [PRINCIPLE_5_NAME] → V. Necessary Simplicity

Added sections:
- Testing & Architectural Enforcement (was [SECTION_2_NAME])
- Development Workflow & Quality Gates (was [SECTION_3_NAME])
- Governance (populated from [GOVERNANCE_RULES])

Removed sections: none

Deferred / follow-up TODOs: none. RATIFICATION_DATE set to the date of this
initial adoption (2026-08-13), matching the date recorded in ADR 0001.
-->

# CLI Task Manager & Focus Timer Constitution

## Core Principles

### I. ADR Authority (NON-NEGOTIABLE)

Architecture Decision Records live in `adr/`. Before planning, generating tasks, or
implementing any code affected by an architectural decision, every ADR in that directory
MUST be read. ADRs with status `Accepted` are binding on specifications, plans, tasks, and
implementation. ADRs with status `Superseded` or `Rejected` are historical context only and
MUST NOT be applied as current constraints.

When an Accepted ADR conflicts with a new requirement, or appears to require revision, the
affected work MUST stop and the conflict MUST be reported explicitly to the developer. The
architecture and the implementation MUST NOT be changed until the developer provides a new
or superseding ADR. Reinterpreting, working around, or silently narrowing an Accepted ADR is
a violation of this principle.

**Rationale**: This project's implementation is AI-generated under specification-driven
constraints and reviewed by a single developer. ADRs are the developer's control surface over
architecture; an agent that may revise them unilaterally removes that control.

### II. Approved Persistence Architecture (NON-NEGOTIABLE)

Per ADR 0001 (Accepted), task state MUST be persisted to a single local JSON file through the
approved Go persistence implementation.

- Persistence code MUST use only the Go standard library.
- No application package may introduce an alternative persistence backend. The persistence
  package MUST NOT import `database/sql`, SQLite drivers, ORMs, key-value databases, or
  remote storage SDKs.
- Persistence logic MUST remain behind a dedicated persistence package. Domain logic MUST NOT
  import `encoding/json` and MUST NOT perform filesystem operations directly.

**Rationale**: The persisted data must stay directly readable by the developer so that
verifying AI-written code is a matter of inspection rather than trust, and the boundary must
stay narrow enough for a static dependency check to prove it.

### III. Traceable Decisions

Every implementation-plan decision and every implementation task MUST be traceable to either a
specification requirement or an Accepted ADR. When neither authorizes a proposed behavior or
architectural choice, the gap MUST be surfaced to the developer instead of being decided
silently.

**Rationale**: Untraceable decisions accumulate architecture nobody agreed to. Surfacing the
gap costs one question; discovering it after implementation costs a rewrite.

### IV. Standard-Library-First Code Quality

- Application code SHOULD use the Go standard library by default. Third-party application
  dependencies require explicit developer approval before use.
- Development and CI tooling — including `golangci-lint` and `depguard` — is permitted and is
  NOT considered an application runtime dependency.
- Errors MUST be returned and wrapped with useful context. Errors MUST NOT be silently
  discarded. Library code MUST NOT use `panic` for expected failures.
- Exported identifiers MUST have doc comments describing their contract.
- Code MUST pass `gofmt`, `go vet`, and the project's configured `golangci-lint` checks before
  implementation work is considered complete.

**Rationale**: A standalone CLI binary is easiest to build, review, and ship when its runtime
dependency set is the standard library; tooling carries no such cost because it never ships.

### V. Necessary Simplicity

Only behavior required by the current specification or by an Accepted ADR may be implemented.
Speculative features, configuration, abstraction layers, and extension points for anticipated
future requirements MUST NOT be introduced.

Interfaces MUST NOT be introduced solely for hypothetical future implementations. An interface
MAY be used when it defines a required architectural boundary, including the persistence
boundary required by ADR 0001. Abstractions required to enforce an Accepted ADR, to enable
test isolation, or to establish an explicitly required architectural boundary are likewise
allowed. Clear Go standard-library solutions are preferred over unnecessary complexity.

**Rationale**: The allowed abstractions share one trait — an existing, written justification.
That is the line between structure that is governed and structure that is guessed at.

## Testing & Architectural Enforcement

- Every testable behavior required by a specification MUST have corresponding automated test
  coverage.
- Persistence tests MUST use isolated temporary data locations and MUST NOT read or modify the
  user's real task data.
- Architectural constraints MUST be verified by automated CI checks rather than by review
  alone.
- Dependency restrictions for the persistence package MUST be enforced with `depguard` under
  `golangci-lint`.
- The approved JSON persistence path MUST also be verified behaviorally, so that dependency
  checks alone are never treated as proof of architectural compliance.

The static and behavioral checks are complementary and both are required: the dependency check
proves no prohibited backend was introduced, and the behavioral check proves the approved path
is the one actually used at runtime.

## Development Workflow & Quality Gates

1. **Read ADRs first.** Planning, task generation, and implementation of architecture-affected
   work MUST begin by reading `adr/`. Persistence-related work MUST read
   `adr/0001-persist-tasks-to-local-json.md` before persistence principles are established or
   applied.
2. **Trace before deciding.** Each plan decision and task cites its authorizing specification
   requirement or Accepted ADR. Unauthorized items are reported as gaps, not resolved by the
   agent.
3. **Stop on conflict.** An apparent ADR conflict halts the affected work and is reported
   immediately; work resumes only after the developer supplies a new or superseding ADR.
4. **Gate completion.** Implementation work is complete only when `gofmt`, `go vet`,
   `golangci-lint` (including the `depguard` persistence rules), and the full test suite —
   behavioral persistence check included — all pass.

## Governance

This constitution supersedes all other development practices for this project. Where this
constitution and an Accepted ADR both apply, both are binding; where they appear to conflict,
work stops and the developer decides.

**Amendment procedure.** Amendments MUST be made by updating this document, and MUST record the
resulting version, the rationale for the bump, and the amendment date. Architectural changes
MUST additionally be recorded as a new or superseding ADR in `adr/`; an existing Accepted ADR is
never edited retroactively to reflect a new decision. Only the developer may authorize an
amendment.

**Versioning policy.** This constitution uses semantic versioning:

- **MAJOR** — backward-incompatible governance changes: a principle removed or redefined in a
  way that invalidates prior compliance.
- **MINOR** — a new principle or section added, or materially expanded guidance.
- **PATCH** — clarifications, wording, and non-semantic refinements.

**Compliance review.** Every plan, task set, and implementation change MUST be checked against
these principles before it is considered complete. Automated CI checks are the primary
enforcement mechanism for architectural constraints; review alone is not sufficient evidence of
compliance. Any complexity that is not authorized by a specification requirement or an Accepted
ADR MUST be justified to the developer or removed.

**Version**: 1.0.0 | **Ratified**: 2026-08-13 | **Last Amended**: 2026-08-13
