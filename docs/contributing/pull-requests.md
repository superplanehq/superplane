# Pull Requests

## Table of Contents

- [Overview](#overview)
- [Creating a Pull Request](#creating-a-pull-request)
  - [1. Fork and Clone the Repository](#1-fork-and-clone-the-repository)
  - [2. Set Up the Upstream Remote](#2-set-up-the-upstream-remote)
  - [3. Create a Branch](#3-create-a-branch)
  - [4. Make Your Changes](#4-make-your-changes)
  - [5. Commit Your Changes](#5-commit-your-changes)
  - [6. Push to Your Fork](#6-push-to-your-fork)
  - [7. Open the Pull Request](#7-open-the-pull-request)
- [Title Format Rules](#title-format-rules)
  - [Allowed Types](#allowed-types)
  - [Examples](#examples)
  - [Breaking Changes](#breaking-changes)
- [Pull Request Description](#pull-request-description)
  - [Required Information](#required-information)
- [PR Scope](#pr-scope)
  - [What Makes a Well-Scoped PR](#what-makes-a-well-scoped-pr)
  - [When a PR Is Too Large or Too Broad](#when-a-pr-is-too-large-or-too-broad)
  - [Splitting a Large Change](#splitting-a-large-change)
- [Review Process](#review-process)

## Overview

Pull requests are the primary way to contribute code changes to SuperPlane. This guide covers the complete
process of creating a pull request, from setting up your fork to getting your changes merged.

## Creating a Pull Request

### 1. Fork and Clone the Repository

If you haven't already, fork the SuperPlane repository on GitHub and clone your fork to your local machine:

```bash
git clone https://github.com/YOUR_USERNAME/superplane.git
cd superplane
```

### 2. Set Up the Upstream Remote

Add the original repository as an upstream remote to keep your fork synchronized:

```bash
git remote add upstream https://github.com/superplanehq/superplane.git
```

### 3. Create a Branch

Create a new branch for your changes. Use a descriptive name that reflects the work you're doing:

```bash
git checkout -b feat/add-new-feature
```

### 4. Make Your Changes

Make your code changes, following the project's coding standards and guidelines. Remember to:

- Write clear, maintainable code
- Add tests for new functionality
- Update documentation as needed
- Follow the [commit sign-off requirements](commit_sign-off.md)

### 5. Commit Your Changes

Commit your changes with clear, descriptive commit messages. All commits must be signed off:

```bash
git add .
git commit -s -m "feat: Add new feature description"
```

See the [commit sign-off guide](commit_sign-off.md) for more details on signing off commits.

### 6. Push to Your Fork

Push your branch to your fork on GitHub:

```bash
git push origin feat/add-new-feature
```

### 7. Open the Pull Request

1. Navigate to the SuperPlane repository on GitHub
2. You should see a banner suggesting to create a pull request from your recently pushed branch
3. Click "Compare & pull request"
4. Fill in the PR title following the [title format rules](#title-format-rules)
5. Add a detailed description (see [Pull Request Description](#pull-request-description))
6. Submit the pull request

## Title Format Rules

Our CI enforces [semantic pull request](https://github.com/marketplace/actions/semantic-pull-request)
rules on PR titles. This helps keep with communicating intent, measuring velocity, and publishing new
releases.

- Start the title with a type (feat, fix, chore, docs), followed by a colon and a short description.
- The rest of the title is free-form; we do not enforce subject casing.
- Keep the description concise but descriptive enough to capture the main change.

### Allowed Types

- `feat`: New user-facing features or capabilities.
- `fix`: Bug fixes or behavior corrections.
- `chore`: Non user facing changes. Maintenance, dependency bumps, tests, CI, refactoring, etc...
- `docs`: Documentation-only changes.

### Examples

- `feat: Add approvals page filters`
- `feat!: Drop support for github personal access tokens`
- `fix: Handle missing canvas id in logs`
- `chore: Bump Go toolchain version`
- `chore: Add tests for canvas page`
- `docs: Update contributing guide with PR instructions`

### Breaking Changes

For changes that break existing behavior, mark the title as a breaking change by adding a `!` after the type:

- `feat!: remove deprecated approvals endpoint`
- `fix!: change default canvas layout`

You can optionally add more detail about the breaking behavior in the PR description. Be sure to clearly document:

- What behavior is changing
- Why the change is necessary
- Migration steps for users (if applicable)

## Pull Request Description

A good PR description helps reviewers understand your changes quickly. Include:

### Required Information

- **What changed**: A clear summary of what the PR does
- **Why**: The motivation or problem being solved
- **How**: Brief explanation of the approach taken (if not obvious from the code)
- **Related issues**: Link to any related GitHub issues using `Closes #123` or `Fixes #456`
- **Breaking changes**: If applicable, clearly document any breaking changes

## PR Scope

Scope matters more than raw line count: a well-scoped PR has one primary outcome, and a poorly scoped one mixes unrelated changes even when the diff is small. Focused PRs review faster, revert cleanly, and keep history easier to bisect.

### What Makes a Well-Scoped PR

A PR is well-scoped when:

- A reviewer can restate it in **one sentence** without "and also".
- **Every bullet** in the summary supports that same outcome.
- The **title covers all substantive changes**, not just the first one.
- The diff is a **single revert unit**: if something goes wrong, reverting this PR undoes one intent.

Default targets (guidelines, not hard limits):

- About **300 lines changed** is a good aim for most PRs.
- About **15 files** unless the change is mechanical across many call sites.
- **1–3 commits** with a clear story.

Large diffs are fine when the change is mechanical or repetitive, such as a new integration component, an architectural swap, or removing a feature across the stack.

Common patterns that work well:

- **Vertical slice**: migration, API, and UI for one user-facing feature in a single PR.
- **Integration component**: backend implementation, UI mapper, and tests for one (or a symmetric pair
  of) component(s).
- **Foundational stack PR**: proto, plumbing, or infrastructure that later PRs build on. Say explicitly what is intentionally unwired and which PR comes next.
- **Consumer-first removal**: migrate UI/CLI callers first, delete the old API in a follow-up PR.

### When a PR Is Too Large or Too Broad

Split the PR when you have:

- **Unrelated fixes** bundled together (different pages, bugs, or motivations).
- **Refactor and feature** in the same PR, unless the refactor is a small prerequisite and you label it as such.
- **Multiple independent user-facing features** that could ship separately.
- **Config or provider migration plus an unrelated UI tweak** — ship the migration and the UI change in separate PRs unless they are inseparable.
- **Cosmetic polish mixed with behavior changes** — keep polish in the same PR only when it is the same code as your change.

Line count alone does not make a PR too large. A 2,000-line integration component or end-to-end feature removal can be well-scoped. A 50-line PR that fixes two unrelated things is not.

### Splitting a Large Change

When a change does not fit in one PR, split by **concern and dependency**, not by "whatever is left on the branch."

1. **Stack by dependency.** Land prerequisites first: proto and messages, then publishers, then
   consumers, then cleanup. Link each PR to the next in the description.
2. **Migrate consumers before deleting APIs.** Update UI and CLI callers before removing an endpoint.
3. **Split independent concerns onto separate branches.** Each PR should have its own motivation, not just a slice of leftover work.
5. **Label stack PRs.** State what is incomplete, what nothing calls yet, and which PR builds on this one.

Default to independent PRs off `main`. Stack PRs only when the dependency is real.

## Review Process

Once you submit a pull request:

1. **Automated Checks**: CI will run tests and checks, including:

   - DCO verification (all commits must be signed off)
   - Semantic PR title validation
   - Code linting and formatting
   - Test suite execution

2. **Code Review**: Maintainers will review your code for:

   - Code quality and correctness
   - Adherence to project standards
   - Test coverage
   - Documentation updates

3. **Feedback and Iteration**: Be prepared to:

   - Address review comments
   - Make requested changes
   - Update your branch by pushing new commits (they will automatically appear in the PR)

4. **Approval and Merge**: Once approved, a maintainer will merge your PR.

**Tips for a smooth review:**

- Follow the [PR scope guidelines](#pr-scope)
- Respond to review comments promptly
- Don't hesitate to ask questions if something is unclear
