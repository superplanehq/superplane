# Pull request titles

- [Overview](#overview)
- [Title format rules](#title-format-rules)
- [Allowed types](#allowed-types)
- [Examples](#examples)
- [Breaking changes](#breaking-changes)

## Overview

Our CI enforces [semantic pull request](https://github.com/marketplace/actions/semantic-pull-request)
rules on PR titles. This helps keep with communicating intent, measuring velocity, and publishing new
releases.

- Start the title with a type (feat, fix, chore, docs), followed by a colon and a short description.
- The rest of the title is free-form; we do not enforce subject casing.
- Keep the description concise but descriptive enough to capture the main change.

## Examples

- `feat: Add approvals page filters`
- `feat!: Drop support for github personal access tokens`
- `fix: Handle missing canvas id in logs`
- `chore: Bump Go toolchain version`
- `chore: Add tests for canvas page`
- `docs: `

## Allowed types

- `feat`: New user-facing features or capabilities.
- `fix`: Bug fixes or behavior corrections.
- `chore`: Non user facing changes. Maintenance, dependency bumps, tests, CI, refactoring, etc...
- `docs`: Documentation-only changes.

## Breaking changes

For changes that break existing behavior, mark the title as a breaking change by adding a `!` after the type:

- `feat!: remove deprecated approvals endpoint`
- `fix!: change default canvas layout`

You can optionally add more detail about the breaking behavior in the PR description.
