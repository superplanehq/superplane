# Issue Auto-Resolve for `agent` Label

Keywords: github issue, labeled, agent label, auto resolve, daytona, llm, close issue, status comments

Use this pattern when a user wants issues to be auto-resolved after they are triaged with the `agent` label.

## Decision Checklist

1. Listen to `github.onIssue` with action filter set to `labeled`.
2. Ensure the flow only continues when the added label is `agent`.
3. Post a "started" status update with `github.createIssueComment`.
4. Run the resolution step on Daytona with `daytona.executeCode` (LLM-driven fix plan + implementation).
5. Resolve the issue with `github.updateIssue` (set state to closed).
6. Post a "completed" status update with `github.createIssueComment`.

## Canonical workflow

trigger: `github.onIssue` action: labeled
-> guard/filter: label name equals `agent`
-> `github.createIssueComment` (started message, include execution id)
-> `daytona.executeCode` (run LLM instructions to produce/apply fix)
-> `github.updateIssue` (close issue when resolve step succeeds)
-> `github.createIssueComment` (completed message and short result summary)

## Notes

- Keep both comments concise and operational (what is happening + run identifier).
- If the Daytona step fails, post a failure comment instead of closing the issue.
- Prefer deterministic prompts for the LLM step (include repository, issue id, and expected output format).
