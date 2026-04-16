---
description: Update the current working branch with the latest changes from main, merge, reconcile differences, and resolve conflicts.
---

# Sync working branch with main

Update the current branch with the latest `main` branch changes from `origin`, merge `main` into the current branch, reconcile differences, and resolve merge conflicts.

## Inputs

- Optional: remote name (default `origin`)
- Optional: base branch name (default `main`)

## Process

1. Validate repo state:
   - Confirm this is a git repository.
   - Detect current branch (`git branch --show-current`).
   - If current branch is `main`, stop and ask for a feature branch.
2. Check for local uncommitted changes:
   - If dirty, auto-stash with a descriptive message before merge.
3. Pull latest base branch state:
   - Run `git fetch <remote> <base-branch>`.
   - Inspect incoming changes from `<remote>/<base-branch>` compared to current branch (`git log --oneline --left-right --cherry-pick HEAD...<remote>/<base-branch>`).
4. Merge base into current branch:
   - Run `git merge <remote>/<base-branch>`.
   - If there are no conflicts, continue.
5. If conflicts exist, resolve them fully:
   - List conflicted files with `git status --short`.
   - For each conflict, inspect both sides and reconcile intentionally (do not blanket-use `--ours` or `--theirs`).
   - Preserve current branch intent while incorporating important fixes/changes from `<base-branch>`.
   - Stage resolved files with `git add`.
   - Complete the merge commit.
6. Validate the result:
   - Run relevant tests/checks for changed areas.
   - Ensure working tree is clean and merge is complete.
7. If a stash was created:
   - Re-apply it and resolve any follow-up conflicts.
   - Re-run relevant checks if needed.
8. Output summary:
   - Current branch name.
   - Latest fetched base branch commit.
   - Whether conflicts were found and how many files were reconciled.
   - Final merge commit SHA.

## Constraints

- Do not force-push.
- Do not rewrite history unless explicitly requested.
- Prefer a standard merge commit (not rebase) for this command.
