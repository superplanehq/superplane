**Software Factory** turns a GitHub issue into a reviewed pull request: plan, implement, open a draft PR, babysit CI, then hand off for human review.

___

> [!SECTION:setup] Quick start
>
> 1. Describe the work in **Create a task** and submit.
> 2. Or add the `factory` label to an existing issue.
> 3. Or assign the issue to `superplaneagent`.
> 4. Or mention `@superplaneagent` on an issue.
>
> Each path fires the factory and shows up on the **PR pipeline** board.

> [!SECTION:runbook] How the factory works
>
> 1. [Create Branch](node:runner-create-branch-2) and [Open Draft PR](node:github-create-pr-2) set up the workstream.
> 2. [Write Plan](node:implementation-implementation-2-td4042) drafts the approach.
> 3. [Implementation](node:runner-implement) applies the change.
> 4. [Run Semaphore CI](node:semaphore-runworkflow-semaphore-runworkflow-v7enk8) checks the PR; [CI Loop](node:ci-loop) retries repairs up to five times.
> 5. [Mark PR Ready](node:github-markpullrequestreadyforreview-github-markpullrequestreadyforreview-wie92w) moves the card to **Human review**.

> [!SECTION:run] Follow-ups
>
> Mention `@superplaneagent` on a PR conversation or review comment.
> The request is applied on the existing branch and CI is babysat again via
> [On mention on PR](node:on-pr-comment-trigger).

> [!SECTION:troubleshoot] Board columns
>
> | Column | Meaning |
> | ------ | ------- |
> | In progress | Draft PR open; plan, implement, or CI still running |
> | Human review | Ready for a person to review |
> | Failed | Run failed or was cancelled |
> | Done | Run passed |
