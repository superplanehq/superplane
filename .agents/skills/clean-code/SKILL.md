---
name: clean-code
description: >-
  Clean-code engineering standards for writing, refactoring, and reviewing code
  in any programming language. Use this whenever the user asks to write clean
  code, follow clean-code principles, refactor for clarity, improve naming,
  reduce complexity or duplication, separate concerns, tighten error handling,
  work test-first or do TDD, or otherwise raise code quality, readability, and
  maintainability. Apply these rules by default when producing or changing code
  for a quality-conscious user.
---

# Clean Code

These are mandatory engineering standards for any code you write, change, or
review while this skill is active. They are language-agnostic: apply them in
whatever language the project uses, respecting that language's idioms.

If there is tension between speed and cleanliness, choose the clean solution
unless there is a demonstrated operational emergency. If a shortcut is
unavoidable, contain it, make it obvious, and leave the code safe to clean up
later.

For detailed rationale, examples, and design heuristics, read
[reference.md](reference.md) — consult it when a rule needs interpretation or
when you face a non-obvious design decision.

## Core Rules

### 1. Leave the code cleaner

Every change must improve at least one of: clarity, naming, simplicity,
cohesion, testability, error handling, duplication removal, or separation of
concerns. Do not add code that is merely functional — add code that is
understandable.

### 2. Optimize for readers

- Write code for the next engineer, not for impressiveness.
- Prefer straightforward code over clever code.
- Keep control flow obvious.
- If code is hard to explain, treat that as a design problem.

### 3. Use strong names

- Names must reveal intent. Use domain language consistently.
- Prefer specific names over generic ones. Avoid `data`, `info`, `thing`,
  `stuff`, `manager`, `helper`, `util`, `misc`, `handle`, `temp`, `obj` unless
  they are genuinely correct in the domain or an external protocol.
- Boolean names read like predicates (`active?`, `isActive`, `hasItems`,
  `retryable?`), following the language's convention.
- Collection names are plural.
- Function names describe the action performed, not vague activity
  (`calculate_invoice_total`, not `do_invoice`).

### 4. Keep methods small and focused

- A method should do one thing at one level of abstraction.
- Do not mix business rules, formatting, persistence, external calls, and error
  recovery in one method. Extract helpers when a method grows multiple phases.
- Prefer zero, one, or two parameters; three should trigger scrutiny; more
  usually means a missing value object or domain object.
- Do not use boolean parameters to switch behavior when separate methods would
  express intent more clearly.

### 5. Keep abstraction levels separate

- Do not mix high-level policy with low-level mechanics in one method.
- Keep orchestration separate from implementation detail, and domain logic
  separate from I/O, framework glue, and serialization.

### 6. Make side effects explicit

- Avoid methods that both answer a question and perform an action.
- Side effects must be visible in naming and placement; avoid hidden mutation.
- Prefer explicit dependencies over implicit globals.

### 7. Control conditionals

- Prefer guard clauses over deep nesting; keep branching shallow.
- Extract complex predicates into intention-revealing names.
- Avoid repeated conditional logic across multiple places.
- Prefer positive conditions where they read more clearly.

### 8. Treat errors as design

- Fail fast on invalid assumptions; do not swallow errors silently.
- Raise or return errors with useful, diagnostic context.
- Keep the happy path readable.
- Do not blur validation errors, domain errors, and infrastructure failures
  without reason.

### 9. Keep modules and classes cohesive

- Each unit should have one clear reason to change. Keep public surfaces small.
- Avoid god objects, god modules, and catch-all utility namespaces.
- Prefer composition over tangled inheritance.

### 10. Remove duplication at the right level

- Remove duplicated logic, decisions, business rules, and meaningful literals.
- Do not invent premature or unstable abstractions just to merge superficially
  similar lines. Deduplicate at the right level of meaning.

### 11. Work test-first

- Default to test-driven development. Write or update a failing test first when
  changing behavior, fixing a bug, or adding a feature.
- Test behavior, not incidental implementation details.
- Tests must be readable, deterministic, independent of order, and named by the
  expected outcome. Keep setup small and intention-revealing.
- Add regression coverage for bugs before fixing them whenever practical.
- If strict test-first is genuinely blocked (missing harness, legacy
  constraints), say so explicitly and add the nearest useful coverage
  immediately, leaving the area more testable than before.

### 12. Refactor while you work

- Improve weak names, break up long methods, and remove dead code in the area
  you touch.
- Keep tests passing while refactoring.
- If a full cleanup is out of scope, improve the nearest meaningful boundary and
  leave a clear note.

### 13. Respect language idioms — without hiding intent

- Use the target language's idioms only when they remain obvious to a competent
  engineer in that language.
- Prefer small, expressive units over procedural scripts with shared mutable
  state.
- Avoid metaprogramming, magic, or DSLs when plain code would be clearer at both
  the call site and the definition site.
- Isolate external systems (databases, APIs, queues, frameworks) behind clear
  boundaries; keep vendor-specific knowledge from leaking everywhere.

### 14. Definition of done

A task is done only when:

- the implementation is correct
- the design is understandable and the names are clear
- the touched code is cleaner than before
- relevant tests exist and pass (test-first unless a real constraint was noted)
- no obvious duplication or dead code remains in the changed area
