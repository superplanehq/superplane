# Clean Code Reference

This document expands on the rules in [SKILL.md](SKILL.md). Treat `SKILL.md` as
the mandatory short operating contract and use this file when you need detailed
interpretation, examples, or design guidance.

Examples below use generic naming for illustration. Adapt them to the
conventions of the language you are working in (for example, a boolean accessor
may be `active?` in Ruby, `isActive` in JavaScript or Go, or `is_active` in
Python). The principle is what matters, not the surface form.

## What "Clean" Means

Code is clean only if it is:

- easy to read without mentally simulating hidden behavior
- organized around clear responsibilities
- named so that intent is obvious
- small in units of behavior
- light on branching and incidental complexity
- explicit about dependencies and side effects
- easy to test in isolation
- resilient to change in one place without surprising breakage elsewhere

If code is hard to name, hard to test, or hard to explain, that is a design
smell, not a documentation problem.

## Naming Guidance

- Use names that answer what this is, why it exists, and how it is used.
- Avoid misleading names or names that imply the wrong shape or behavior.
- Avoid unnecessary abbreviations; prefer searchable names over single letters.
- Preserve domain language consistently across files.

Weak names to avoid by default: `data`, `info`, `thing`, `stuff`, `process`,
`manager`, `helper`, `util`, `temp`, `value`, `obj`. These are acceptable only
when they are genuinely part of the domain or an external protocol.

Function and method names must describe the action performed, not vague
activity.

Prefer `calculate_invoice_total`, `load_workspace_settings`, `enqueue_retry`
over `do_invoice`, `handle_settings`, `process_retry`.

## Methods and Parameters

- A method should do one thing, do it well, and stay at one level of
  abstraction.
- If a method has several sections, it usually wants extraction. If it mixes
  parsing, validation, branching, persistence, formatting, and reporting, it is
  too large.
- Prefer zero, one, or two parameters. Three should trigger scrutiny. More than
  three usually indicates missing structure — group related arguments into a
  value object or domain object.
- Avoid boolean parameters that select behavior.

Prefer `render_public_profile(user)` and `render_private_profile(user)` over
`render_profile(user, include_private_data)`.

## Comments

Comments are not a substitute for clean code.

Good uses: legal notices, warnings about surprising external constraints, short
rationale for a non-obvious tradeoff, intent that cannot be expressed cleanly in
code, and specific actionable TODOs.

Bad comments: restating the code, misleading comments, commented-out code,
journal comments, noise compensating for poor naming, and large explanatory
blocks covering messy code that should be refactored.

If you want to write a long explanatory comment, first ask whether the code
should be simplified instead.

## Formatting and Layout

Formatting must reveal structure.

- Keep related concepts vertically close; separate unrelated concerns with space.
- Use consistent formatting within a file.
- Order code from higher-level ideas to lower-level details when practical.
- Avoid dense walls of code. Readers should grasp a file's shape by scanning it.

## Conditionals

- Prefer intention-revealing predicates.
- Replace nested conditionals with guard clauses when that improves clarity.
- Encapsulate complex conditions in named methods or domain objects.
- Avoid negative conditionals where a positive form is clearer.
- Avoid repeated branching logic across files.

Prefer `return unauthorized_error unless user.can_publish?` over deeply nested
conditionals with implicit else behavior.

## Error Handling

Error handling is part of the design.

- Fail fast on invalid inputs and broken assumptions.
- Prefer explicit handling over silent failure.
- Attach useful, diagnostic context to errors.
- Keep the happy path readable.
- Do not treat validation failures, infrastructure failures, and programmer
  errors as interchangeable unless the design clearly requires it.

Error messages should help a future engineer diagnose the issue.

## Data and Abstractions

- Prefer cohesive domain types over loose bags of fields when behavior belongs
  with the data.
- Avoid anemic abstractions that only move state around, unless they are
  intentionally simple transport objects.
- Do not expose internals unnecessarily. Keep interfaces small and
  intention-revealing.
- Hide volatile implementation details behind stable boundaries when useful.

Data structures should expose data and little behavior. Objects should expose
behavior and hide representation. Do not mix the two carelessly.

## Classes, Modules, and Files

- Each unit should have one clear reason to change. Keep public surfaces small.
- Keep related behavior together and separate policy from infrastructure.
- Avoid files that know too much.

Refactor when you see: a broad type with unrelated methods, a module
coordinating many unrelated systems, repeated conditionals in multiple
locations, feature envy, or vague file and module names.

## Duplication

Duplication includes copied code, repeated business rules, repeated protocol
translation, repeated validation, repeated workflow steps, and repeated literals
with shared meaning.

Remove duplication at the right level of abstraction. Do not invent unstable
abstractions just to merge two similar lines.

## Separation of Concerns

Keep separate: business rules from I/O, orchestration from low-level mechanics,
validation from persistence, parsing from domain decisions, formatting from
computation, and reads from writes when separation improves clarity.

When one unit handles too many concerns, split it.

## Dependencies

- High-level policy should not depend directly on volatile details when a
  boundary is warranted.
- Isolate external systems behind adapters or dedicated integration objects.
- Favor explicit construction and dependency injection when they improve
  testability and clarity.
- Avoid hidden global dependencies and minimize coupling between modules.

Avoid global mutable state. If it is unavoidable, isolate it tightly and make
its usage explicit.

## Test-Driven Development

Tests are design tools, not cleanup work.

- Default to test-driven development.
- For new behavior, bug fixes, and behavior changes, start by expressing the
  expected behavior in a test.
- Write the smallest failing test that proves the requirement.
- Implement only enough code to make the test pass, then refactor once behavior
  is protected.
- Add regression tests for bugs before fixing them whenever practical.

When strict test-first sequencing is blocked by missing harnesses, legacy
constraints, or infrastructure gaps: say so explicitly, add the nearest useful
automated coverage immediately, and leave the area more testable than before.

Tests must be readable, deterministic, focused on behavior, named by expected
outcome, concise in setup, and independent of execution order.

Test domain behavior, edge cases, failure paths, important invariants, and
regressions. Avoid tests that mirror implementation line by line, brittle
snapshots without discipline, huge integration tests compensating for missing
unit boundaries, and vague assertions that do not diagnose failure.

## Refactoring

Refactoring is mandatory when code quality blocks clarity.

- Make small, behavior-preserving improvements continuously.
- Keep tests passing while refactoring.
- Do not leave obvious messes in code you modify. If a full cleanup is too
  large, improve the nearest meaningful boundary.

Typical triggers: poor names, long methods, duplicated logic, mixed abstraction
levels, unnecessary indirection, deep nesting, sprawling conditionals, and
confusing data flow.

## Simplicity

Prefer the simplest design that correctly supports the required behavior.

- Do not speculate about future requirements without evidence.
- Do not build extension points just in case.
- Do not introduce patterns unless they solve a present problem.
- Remove accidental complexity.

Simple does not mean crude. It means there are no unnecessary parts.

## Boundaries and External Systems

Code that talks to databases, APIs, filesystems, queues, or frameworks must
protect the domain from churn.

- Isolate external protocols behind clear boundaries.
- Translate external data into internal representations deliberately.
- Keep framework-specific code from leaking everywhere.
- Write focused tests around boundary behavior.
- Avoid scattering vendor-specific knowledge across the codebase.

## Concurrency and State

If the project uses shared state, background jobs, or concurrent execution:

- minimize shared mutable state
- make ownership and lifecycle explicit
- avoid hidden ordering assumptions
- test invariants where races or retries are possible
- keep concurrent code small and well-contained

## Constants and Literals

- Replace magic numbers and opaque literals with named concepts when meaning is
  not obvious.
- Keep obvious local literals inline when extraction would hurt readability.
- Use named constants for protocol values, thresholds, default limits, and
  status mappings.

## APIs and Interfaces

- Make interfaces small and hard to misuse.
- Favor explicitness over convenience when convenience invites bugs.
- Keep input and output contracts coherent.
- Use names and types to make valid usage obvious.
- Do not expose knobs callers should not control.

## Language and Idiom Guidance

Code should be expressive, not magical, in any language.

- Prefer plain units with clear responsibilities over giant service objects.
- Avoid naming things `FooService`, `FooManager`, or `FooHelper` when a more
  precise domain name exists.
- Use shared modules, mixins, or traits for shared domain meaning or
  well-bounded reuse, not as dumping grounds.
- Use metaprogramming, reflection, and macros sparingly, and only when they make
  code clearer at both the call site and the definition site.
- Keep callback-driven and lifecycle-driven control flow under control. Hidden
  behavior is harder to reason about than explicit calls.
- Favor immutable values where practical.
- Keep ORM, HTTP client, and other infrastructure concerns from swallowing
  domain logic.
- Prefer explicit query methods, command methods, and object boundaries over
  DSL-heavy designs.

## Red Flags

Stop and reconsider when you see: methods with multiple phases of work, boolean
parameters controlling branching, comments explaining confusing code, repeated
`if`/`case`/`switch` logic across files, vague module names, hidden
dependencies, methods that both mutate and answer, long parameter lists, broad
catch-all error handling, sprawling test setup, or utility namespaces collecting
unrelated behavior.

## Decision Heuristics

When multiple designs are possible, prefer the one that:

1. is easiest to explain to another engineer
2. keeps responsibilities separated
3. minimizes hidden state and side effects
4. is easiest to test
5. removes duplication without inventing unstable abstractions
6. preserves flexibility through clarity rather than indirection

## Final Instruction

Treat clean code as a delivery requirement, not a polish pass.

If forced to choose between a fast messy solution and a slightly slower clean
one, choose the clean one unless there is a genuine operational emergency. If a
shortcut is unavoidable, contain it, make it obvious, and leave the code in a
state that can be cleaned up safely.
