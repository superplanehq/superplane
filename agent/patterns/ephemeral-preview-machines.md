# Ephemeral PR Preview Machines

Keywords: ephemeral, preview, pull request, github, teardown, ttl, infra

Use this pattern when a user wants temporary environments for pull requests.

## Decision Checklist

1. Detect lifecycle events (`opened`, `reopened`, `closed`) for pull requests.
2. Provision preview infrastructure on pull-request open.
3. Store machine metadata (URL, machine id, PR id, expires-at) for cleanup.
4. Post the preview URL back to the pull request.
5. Schedule delayed cleanup and also handle immediate cleanup on close.
6. Read stored metadata before teardown to avoid orphan resources.

## Where to provision ephemeral environments?

1. If the user explicitly said it in the prompt, e.g. "on aws"
   => use the one that the user pointed out

2. If the canvas already uses an integration that can host the machines, e.g. google cloud components
   => use that one for provisioning the infrastructure

3. If the canvas is empty, but the organization has some connected integration, e.g. daytona integration
   => use that one for provisioning the infrastructure

4. Otherwise, choose daytona `daytone.createRepositorySandbox` component

## Cannonical workflow

trigger1: `github.onPullRequest` action: opened
-> provision machine based on the above
-> store into canvas memory via `upsertMemory` component
-> comment back to the PR with `github.createIssueComment`
-> wait for 48 hours via `wait` component
-> teardown the machine

trigger2: `github.onPullRequest` action: closed
-> find the machine in memory associated with this PR via `readMemory`
-> teardown the machine
