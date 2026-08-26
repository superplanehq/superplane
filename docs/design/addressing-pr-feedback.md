One thing missing in our onboarding factory is the "addressing comments" part.
After a pull request has been opened for a factory work order, the factory should react to comments on that pull request.

### Existing behaviors in other tools

#### Cursor's /babysit command

- Skill that uses the GH API to check the state of a PR.
- It goes through unresolved conversations and addresses them.
- Once addressing all of them, it stops.
- You have to re-trigger it if more comments are added.

### Mentioning tool/agent on comments

- Cursor and Copilot will address comments when you mention them on the comment.
- Devin also has a mention-only configuration - not default - and only applied to humans.
- Devin, by default, addresses human comments and ignores comments from bots. You have to manually enable reacting to bot comments; recommended way is to allowlist the bots you want to react to.

### GH events

GitHub sends us a couple of webhooks we could use to guide this:
- `pull_request` - opened, closed, assigned, synchronized
- `issue_comment` - name is misleading, we also get events when someone comments on a PR conversation, not a review
- `pull_request_review` - dismissed, submitted
- `pull_request_review_comment`
- `pull_request_review_thread` - resolved, unresolved

### Our components

- `github.onPullRequest` trigger
- `github.onPRReviewComment` trigger: listens to both `pull_request_review` and `pull_request_review_comment` events, which is a bit weird.
- `github.onPRComment` trigger: listens to `issue_comment` events

### Things we know

- We need synchronization. We cannot have more than one agent addressing comments at the same time for the same PR.
- We need to listen to:
  (1) PR conversation comments
  (2) PR review comments
  (3) PR review submission with comments
- The work being done to address the comment - the run - should be visible to the user somehow
- Possibly sensible decision:
  - if user leaves single comments -> we work on them, one by one
  - if user submits a review -> we work on all the review comments at once
  - Question here is if this can be achieved through the automation that drives PR comments itself only somehow
  - We could rely on the concurrency + partition key node configuration, for example, to ensure only a single fixer is running for a PR
- We have this special "Verify" column, where the work order sits, while comments are being addressed

### Things not clear yet

- How do we make the process of resolving comments visible to the user?
- When do we react to a comment? Is mentioning required? What if a comment is not for the factory, but a question for some other user?
  - Easiest way to start is to always require `superplaneagent` to be mentioned

## TODOs

- [ ] Verify the 1 `pull_request_review` + multiple `pull_request_review_comment` events behavior on PR review submittal - no `issue_comment` events are sent
- [ ] Verify how our current `github.onPRReviewComment` works. Will it emit events for comments, and single event for review submission?
