There are no GitHub Actions workflow files in this repo - just the bot checks. Let me check the current check status once more:

- **DCO**: ✅ pass (fixed by adding `Signed-off-by` trailers to all commits)
- **policy-bot: main**: pending (requires human reviewer approval — not fixable programmatically)
- **OSS Guard**: ✅ pass

The only actionable check failure was **DCO**. Here's what I did:

1. **Identified the issue**: All 8 commits on the branch were missing `Signed-off-by` trailers required by the DCO check.
2. **Fixed it**: Ran `git rebase --signoff main` to add `Signed-off-by: Fabro <noreply@fabro.sh>` to all commits.
3. **Created the PR**: Opened PR [#6145](https://github.com/superplanehq/superplane/pull/6145) with title `fix: Remove "Pin App" button from homepage, keep only star`.
4. **Force-pushed**: Updated the branch with the rebased commits.

The `policy-bot: main` check (0/1 rules approved) is pending because it requires a human reviewer to approve the PR — that's expected for a draft PR and is not a CI failure to fix.