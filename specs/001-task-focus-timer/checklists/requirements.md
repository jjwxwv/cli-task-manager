# Specification Quality Checklist: Task Manager with Focus Timer

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-14
**Last reviewed**: 2026-08-14 (ADR 0001 conformance audit, a coherence pass on the identifier
concept, then re-validated after `/speckit-clarify`)
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

**Status: 16 of 16 items pass. The spec is ready for `/speckit-plan`.**

### Clarification markers, now closed

Two decisions were deliberately left unresolved rather than defaulted, in line with
Constitution Principle III (surface the gap instead of deciding silently), and were answered
by the developer during `/speckit-clarify` on 2026-08-14:

- **FR-012** — the completed interval is announced by a printed message, with no audible
  alert. A third question surfaced during the same session settled the reporting cadence: the
  time remaining is printed at the start and once per elapsed minute, each on its own line.
  Without that cadence the requirement stated a behavior no test could judge.
- **FR-013** — an interrupted interval prints the elapsed time and exits with a non-zero
  status, without prompting for confirmation. The non-zero status is what makes interruption
  distinguishable from the completion FR-012 describes, which exits zero.

User Story 3 Acceptance Scenarios 1, 2, and 3 now state concrete outcomes rather than
deferring to their requirements, which is what closed the two Feature Readiness and
Requirement Completeness items that previously failed.

### FR-006 closed by derivation, not by choice

FR-006 was previously an open marker offering three options. It is now resolved, and it was
not a free choice — two of the three options contradicted constraints the specification had
already committed to:

- **Descriptive text** fails the duplicate-text edge case: two tasks sharing text cannot be
  told apart by text.
- **Ordinal position** cannot be used at all. An ordinal is a task's place within a
  collection, and no command in scope ever displays that collection, so there is no occasion
  on which a user learns a position. Were the system to print a number at add time, what the
  user reads is an assigned identifier that happens to be sequential — not an ordinal. This
  option is not a costlier alternative to an identifier; it is inoperable.

That leaves one answer consistent with the uniqueness and stability the spec requires: an
identifier the system assigns and prints at add time. Presenting it as an open question would
have implied a latitude that did not exist. FR-006 now states the obligation directly, and the
derivation is recorded in the spec's Assumptions so a later reader does not reopen it as
though it were arbitrary.

Downstream updates applied with it: FR-003 discloses the identifier, Key Entities carries it
as a defined Task attribute, User Story 1 Scenario 1 and User Story 2 Scenarios 1, 3, and 4
name it concretely, the duplicate-text and lost-identifier edge cases refer to it, and SC-003
is conditioned on the user holding it.

### ADR 0001 conformance re-audit

Checked the spec clause by clause against [ADR 0001](../../../adr/0001-persist-tasks-to-local-json.md).
No contradiction found. The spec's Assumptions align with the ADR's recorded assumptions on
every point: single local user, separate short-lived invocations, small dataset suited to
whole-file reads and writes, concurrency not a requirement, no external database service.

The spec correctly stays silent on storage format, file location, and write mechanics, all of
which the ADR assigns to implementation or planning. The ADR's atomic-write mandate
(`os.CreateTemp` + `os.Rename`) has no corresponding spec requirement, which is correct — the
ADR places cross-platform crash durability out of scope, so that task traces to the ADR rather
than to this spec.

The audit surfaced three issues where the spec was tighter than its authorizing documents
warranted. All three were corrected on 2026-08-14 with the developer's approval:

1. **Edge case on duplicate text silently constrained FR-006 before it was answered.** The
   edge case "two tasks added with identical text ... MUST remain separately identifiable"
   rules out identifying a task by its exact text — one of the three options FR-006 then posed.
   *Resolved*: the constraint moved into FR-006 itself, which subsequently made clear that the
   question had only one admissible answer. See "FR-006 closed by derivation" above.

2. **Edge case on unreadable data narrowed a decision ADR 0001 defers to planning.** The ADR
   offers three candidate behaviors for unreadable data — fail loudly, quarantine the file, or
   reconstruct from a retained copy — and assigns the choice to the plan. The spec's blanket
   "MUST report the problem and stop" eliminated reconstruction before planning began.
   *Resolved*: the edge case is now split. Data in an unrecognized format MUST report and stop,
   which the ADR mandates directly. Malformed data carries only the guarantee that nothing is
   silently discarded and no false success is reported; the recovery behavior itself is left to
   planning, as the ADR intends.

3. **FR-011 collided with the test coverage SC-004 requires.** FR-011 forbade "any way" to
   change the 25-minute duration, while SC-004 requires verifying that duration to within one
   second — which no practical automated test achieves at full length without a duration or
   clock seam. Constitution Principle V permits abstractions that enable test isolation, but the
   original wording would have forced a needless ADR-conflict stop during planning.
   *Resolved*: FR-011 now bounds the user-facing surface only and states explicitly that a
   test-only seam is permitted and necessary.

No further ADR conformance issues outstanding.

### Coherence pass on the task reference

Successive reviews found the concept FR-006 introduces — the identifier a user types to name a
task — was stated in one place and assumed in several others. Six corrections followed:

1. **FR-006 stated its constraint backwards.** The requirement described what a future *answer*
   must do and then offered a route to adopt an answer that violated it, by amending the edge
   case. A requirement that explains how to escape itself is not a requirement.
   *Resolved*: FR-006 now states the system obligation directly — every task carries an
   identifier, unique across tasks and stable once disclosed. Making the obligation explicit is
   what exposed that only one form of identifier could satisfy it, which closed the marker
   entirely rather than merely narrowing it.

2. **Key Entities was left behind.** The Task entity still described "a means of referring to
   it individually (pending FR-006)", which understated what FR-006 now guarantees.
   *Resolved*: the identifier is a defined attribute of Task, required to be unique across
   tasks and stable once disclosed.

3. **FR-003 was not tied to FR-006.** It required the add confirmation to include "the means by
   which the user can later refer to the task" without connecting that to the identifier FR-006
   defines, leaving two descriptions of one thing.
   *Resolved*: FR-003 now names FR-006 explicitly and states that this confirmation is the sole
   occasion on which the identifier is disclosed.

4. **The spec never admitted that the user must keep the identifier themselves.** Excluding a
   list command means an identifier lost to a cleared terminal cannot be recovered through any
   command in scope — a real usability cost that appeared nowhere in the document.
   *Resolved*: recorded as an edge case with an explicit statement that no in-scope command
   recovers it, as an assumption placing retention on the user, and folded into SC-003, which
   now reads "a user holding a task's identifier".

5. **The lost-identifier edge case bound the user's workflow to storage.** It told the user
   their "only recourse is to inspect the stored task data outside this tool", which both
   leaks an implementation concern into a specification that otherwise stays clear of storage,
   and quietly promises that stored data will be inspectable by hand. Whether it is remains an
   ADR 0001 and planning matter, not something the spec may presuppose.
   *Resolved*: the edge case now states only that no command in scope recovers a lost
   identifier and that recovery lies outside what the feature offers. It prescribes no
   workaround.

6. **Two assumptions covered the same ground.** One placed retention of the identifier on the
   user; another recorded how FR-006 was derived. Both leaned on the same property — that a
   system-assigned identifier cannot be reproduced from memory — so the reader met the point
   twice and neither statement owned it.
   *Resolved*: merged into a single assumption. It derives the identifier from the spec's own
   constraints and closes by placing retention on the user, stating the property once. Runtime
   behavior when an identifier is lost stays with the edge case, which is where it belongs.

This limitation is a settled cost, not an open trade-off. A system-assigned identifier is not
something a user can reproduce from memory, so an identifier lost with the terminal buffer is
lost as far as this feature is concerned. That cost follows from excluding a list command, and
no admissible answer to FR-006 would have avoided it — the eliminated options were eliminated
on grounds of contradiction and inoperability, which no offsetting convenience could have
redeemed. The spec records the cost as knowingly accepted. Should it prove unacceptable in
practice, the remedy is a scope change admitting some way to look identifiers up, not a
reopening of FR-006.

### Post-clarification coherence pass

Integrating the three clarification answers introduced three further defects, found on review
and corrected:

7. **This checklist contradicted itself.** The closing paragraph above stated that a lost
   identifier "means opening the stored data by hand" — the exact claim item 5 records
   removing from the spec, and for the exact reason given there: it presumes stored data is
   inspectable by hand, which is an ADR 0001 and planning matter.
   *Resolved*: the paragraph now says only that a lost identifier is lost as far as this
   feature is concerned, prescribing no workaround.

8. **FR-012 gained three obligations and no way to measure two of them.** After clarification
   FR-012 required a report at start, a report each elapsed minute, and a closing message, but
   SC-004 measured only the interval's length and that its end is signalled. The reporting
   cadence — the substance of the third clarification answer — had no measurable outcome at
   all, so "Feature meets measurable outcomes" was passing on an incomplete reading.
   *Resolved*: SC-007 fixes the expected output of a completed interval at exactly 25
   time-remaining reports followed by exactly one ending message.

   *Follow-up*: the first version of SC-007 stated 25 but described "one when it starts and
   one at each subsequent elapsed minute", which sums to 26 — a 25-minute interval has 25
   minute boundaries after its start. The arithmetic could not be settled either way, because
   FR-012 never said whether the twenty-fifth boundary produces a final report or is displaced
   by the ending message. FR-012 now states that reports continue only while time remains and
   that the final minute yields the ending message instead, so the sequence runs from 25
   minutes remaining down to 1. SC-007 spells out both endpoints and adds that no report ever
   shows zero remaining, which is the assertion that pins the count.

9. **Exit-zero on normal completion existed only as a subordinate clause.** FR-013 required a
   non-zero exit and explained it was distinguishable "from the completion described in
   FR-012, which exits zero" — but FR-012 never said so. The baseline the whole distinction
   rests on was asserted only in passing, inside the requirement that depends on it.
   *Resolved*: FR-012 now states the zero exit directly, FR-013 drops the parenthetical, and
   SC-008 makes the distinction measurable. The zero baseline is not an unauthorized addition:
   the developer's answer to the interruption question specified a non-zero status, which is
   meaningless except against it.

10. **SC-005 narrowed itself by example.** It opened with "every rejected operation" and then
    listed four in em-dashes, which reads as the set rather than a sample. Any failure mode
    not among the four — and the spec reaches several — escaped the criterion.
    *Resolved*: the criterion now binds every failure the system can reach, and the four cases
    are marked as instances rather than the limit.

11. **User Story 3 Scenario 1 kept the off-by-one FR-012 had just shed.** It still said the
    report repeats "once per elapsed minute", with no endpoint, so it implied a report at the
    twenty-fifth minute — exactly the reading FR-012 was amended to exclude.
    *Resolved*: the scenario now names both endpoints, 25 minutes remaining down to 1, and
    ends the sequence while time remains.

12. **SC-005 re-committed the very error ADR audit item 2 had corrected.** Rewriting it to bind
    every reachable failure was right, but the illustrative list kept "unreadable stored data".
    If planning resolves malformed data by restoring from a retained copy — one of the three
    behaviors ADR 0001 leaves open — the operation succeeds, and naming it as an instance of
    something "the system rejects or cannot complete" would have foreclosed that option from
    the Success Criteria instead of from the Edge Cases. Same defect, different section.
    *Resolved*: the list now names task data in an unrecognized format, which ADR 0001 does
    mandate as a load-time failure, and states outright that malformed data is withheld because
    its status depends on a decision planning has yet to make.

13. **FR-011 had grown a second job.** It prohibited a user-facing duration control and, in the
    same breath, mandated properties of an internal test seam. The prohibition and the
    obligation have different subjects and different audiences, and a reader looking for what
    the seam must do would not think to look under a requirement about what users cannot do.
    *Resolved*: FR-011 is prohibition only and points to FR-016 for what may exist behind the
    surface it bounds. FR-016, under a new Verifiability heading, carries the seam obligation:
    duration and cadence driven from one source, unreachable from the user-facing surface. The
    trap it guards against is recorded with it — a seam compressing only the duration leaves
    SC-004 passing while FR-012's sequence goes untested and SC-007's count becomes
    unverifiable, which is a green build that proves nothing.

    How the seam is built remains a planning decision; FR-016 states only what it must satisfy.

### Exit condition

Met. No clarification markers remain and all sixteen items pass. The spec is ready for
`/speckit-plan`, where the two decisions ADR 0001 defers — the data-file location and the
recovery behavior for malformed data — are settled.
