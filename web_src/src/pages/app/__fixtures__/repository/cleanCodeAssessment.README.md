Clean Code Score

A SuperPlane app that grades every pull request 0-100 against Robert C. Martin's
Clean Code rubric and the measurable metrics Uncle Bob explicitly calls out
(test coverage, dependency structure, cyclomatic complexity, module sizes,
mutation testing). The score and a short justification with concrete examples
land as a single, self-updating PR comment.

The work is done by a Cursor Cloud Agent that actually checks out the PR and
runs metric tooling, rather than an LLM guessing from a diff — there is no
GitHub action in SuperPlane today that returns a PR diff, so checking the code
out is the only honest way to compute the metrics Uncle Bob asks for.

What gets installed







File



Purpose





canvas.yaml



The workflow graph (trigger → analyze → score → comment → store).





console.yaml



The operational dashboard (avg score, score by PR, latest assessments).





params.json



Install-time inputs (target repository).





README.md



This file.



Prerequisites

You need two SuperPlane integrations connected to your organization before installing:





GitHub (GitHub App preferred) with permissions for issues:write,

pull_requests:write, contents:read, and the pull_request event subscribed.



Cursor with a Cloud Agent API key configured. The agent runs the analysis

in an isolated cloud environment — be aware it executes PR code, so apply
 the same trust posture you would to any PR-triggered CI.

Take note of the integration names you used (e.g. acme-github, acme-cursor);
you'll plug them into the YAML below.

Installation



Option A — Install from a GitHub repo (recommended)





Push this folder to a public GitHub repo at the repo root, e.g.

your-org/superplane-clean-code-score.



In SuperPlane, open Apps → Install from GitHub and point at that repo.



The install wizard reads params.json, asks for the target repository, and

wires the GitHub + Cursor integrations automatically.



Open the new canvas's Console to confirm the dashboard rendered.



Option B — Import the YAMLs directly





Create a new canvas in SuperPlane.



Open the Files tab, paste canvas.yaml, and replace the placeholders:





<github-integration-name> → the name of your GitHub integration.



<cursor-integration-name> → the name of your Cursor integration.



{{ install_params.repository }} → the owner/repo you want to monitor
(this placeholder is only resolved by the install wizard from Option A).



Open Console view, click Import, and paste console.yaml.



Data model







Where



What



Lifetime





Canvas memory namespace prScores



One row per PR, keyed by prNumber. Holds score, report (full markdown), commentId, commentUrl, headSha, prTitle, author, prUrl, repo, updatedAt. upsertMemory overwrites in place on every PR update, so there is always exactly one current assessment per PR.



Persists across runs; survives canvas restarts.





Workflow runs (automatic)



Every PR event produces a workflow_runs row + node executions, so historical scores remain queryable via the Console runs data source.



Persists as part of the run history.





Cursor execution KV



The Cursor agent ID is stored as an execution KV for webhook correlation; managed by the component itself.



Per-execution.

Scores are stored as numeric strings (SuperPlane resolves template values to
strings); Console number / chart widgets coerce them, and the table's grade
column casts via int(score).

How the workflow flows

The pipeline has two entry points that both fan into the same scoring chain:





On Pull Request — github.onPullRequest fires on opened, reopened,
synchronize, ready_for_review.



Manual Trigger — start ("Manual Run") that takes a PR number + head SHA
and emits a payload shaped like a GitHub pull_request webhook, so the rest
of the pipeline runs identically. See Manual testing.

Both feed into:





Existing PR State — readMemory looks up the prior assessment for this PR.



Has New Commits? (skip-if-unchanged guard) — only present when a prior

assessment exists; compares stored headSha to the incoming PR head. When
 equal, the run terminates at Analysis Skipped.



Analyze PR — cursor.launchAgent in PR mode (autoCreatePr: false,

useCursorBot: false) checks out the PR and runs the Clean Code prompt.



Agent Succeeded? — if on the terminal status; failures hit Analysis Failed.



Get Report — cursor.getLastMessage returns the full markdown report.

From here, three parallel branches:





Publish Commit Status — clean-code/score success/failure on the PR.



Add Grade Label — clean-code:A/B/C/D based on the score.



Existing Assessment? — readMemory decides post vs update:





notFound → Post Assessment (github.createIssueComment) →
Save New Assessment (upsertMemory with the new comment ID).



found → Update Assessment (github.updateIssueComment reusing the
stored comment ID) → Update Saved Assessment (upsertMemory keeps
commentId/commentUrl, refreshes everything else).

The single, self-updating comment is the result of reusing the stored
commentId — exactly the pattern documented on github.updateIssueComment.

Manual testing

The Manual Trigger node lets you fire the entire pipeline against any open
PR without waiting for a webhook. Two ways to invoke it:





From the Console — open the canvas's Console view; the Manual Trigger
card in the Pipeline panel has a Run button that opens the parameter
form.



From the canvas — click the Manual Trigger node, then Run → score-pr.

Parameters:







Parameter



Type



Required



Notes





prNumber



number



yes



e.g. 1234





headSha



string



yes



The full 40-character commit SHA of the PR head. Grab it from the GitHub PR page (Files tab → commit dropdown) or git rev-parse HEAD on the PR branch. The commit-status step rejects anything that isn't ^[a-f0-9]{40}$.





prTitle



string



no



Defaults to "Manual run"; only used for display in the assessments table.





prAuthor



string



no



Defaults to "manual"; only used for display in the assessments table.

The manual trigger synthesizes a payload with the same shape as
github.onPullRequest, so it shares every downstream node (skip-if-unchanged
guard, agent, comment write/update, commit status, label, memory upsert). A
manual run on a PR that already has an assessment will update the existing
comment in place, just like a real webhook event would.

The target repository is fixed at install time (via
{{ install_params.repository }}); you don't pass it per-run.

Optional enhancements wired into the YAML

These are toggled on by default in the included canvas.yaml; remove the nodes
and edges to opt out.





Skip unchanged commits — a Has New Commits? if node compares the
stored headSha to the incoming PR head.sha before calling the agent.
Avoids burning Cursor cost on label/title edits.



Merge gate (commit status) — github.publishCommitStatus posts
clean-code/score as success when score >= 70, otherwise failure, so
low scores show red on the PR checks.



Grade label — github.addIssueLabel applies clean-code:A / B / C / D
based on score bands, for quick triage in the PR list.



Score trend over time — a runs-based chart panel in console.yaml plots
historical scores using the automatic workflow_runs history (no extra storage).



Verification

After install, the fastest smoke test uses the manual trigger so you don't
have to wait for a webhook:





Pick any open PR on the monitored repo and copy its number + head SHA.



In Console, click Run on the Manual Trigger card (or use the node

directly in the canvas), fill in prNumber and headSha, and run.



Watch the canvas run end-to-end (a few minutes — Cursor agent polling

dominates).



Confirm:





one comment appears on the PR;



clean-code/score shows up as a check on the PR;



a clean-code:A/B/C/D label is applied;



the Console KPIs populate and a row appears in Latest assessments.



Run the manual trigger a second time with the same headSha — the

Has New Commits? guard should short-circuit at Analysis Skipped.



Change headSha (or push a new commit) and re-run — the existing PR

comment should be edited in place (not duplicated).



Cost & safety notes





The Cursor Cloud Agent runs PR code in its isolated environment. Treat it
with the same caution as any PR-triggered CI; do not enable on
fork-permissive public repos without a review.



Pin the Cursor model (e.g. via a Cursor integration default model) for
deterministic scoring across runs.



For an extra cost cut, restrict the trigger to non-draft PRs by adding a
draft != true filter step before Analyze PR.

