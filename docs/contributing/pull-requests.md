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

- Keep PRs focused and reasonably sized
- Respond to review comments promptly
- Don't hesitate to ask questions if something is unclear
