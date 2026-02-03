# Integration Bounties

SuperPlane uses [BountyHub](https://www.bountyhub.dev/) to offer paid bounties on selected integration work. This guide explains how to find, claim, work on, and get paid for bounties, and how review and disputes work.

## Table of Contents

- [Overview](#overview)
- [Finding bounties](#finding-bounties)
- [Claiming a bounty](#claiming-a-bounty)
- [Working on a bounty](#working-on-a-bounty)
- [Review and acceptance](#review-and-acceptance)
- [Disputes](#disputes)

## Overview

We use [BountyHub](https://www.bountyhub.dev/) for bounties (see their site for payout methods, fees, and how it works).

**Only issues with the `bounty` label** are on BountyHub. If an issue does not have that label, there is no bounty attached.

Each bounty issue has a **Bounty Details** section in its description (reward amount and any additional requirements or acceptance criteria). Read it before you start.

You can browse all SuperPlane bounties on [BountyHub](https://www.bountyhub.dev/en/bounties) or find bountied issues in this repo by the `bounty` label.

We may close an issue or withdraw a bounty; work in progress may not be paid if the bounty is no longer active.

## Finding bounties

- **On BountyHub**: Go to [bountyhub.dev/en/bounties](https://www.bountyhub.dev/en/bounties) and search or filter for the SuperPlane repository.
- **On GitHub**: [View all open issues with the `bounty` label](https://github.com/superplanehq/superplane/issues?q=state%3Aopen%20label%3Abounty). Only those issues have an active bounty on BountyHub.

Confirm the bounty exists and is funded on BountyHub before investing significant time. Before you start, sign in to BountyHub with your GitHub account and connect Stripe (or add your PayPal email) so you can receive payouts when your claim is accepted.

## Claiming a bounty

We accept one claim per bounty. When multiple PRs are submitted for the same bounty, we review them and merge the one that best meets the requirements; the bounty is paid to the author of the merged PR.

1. **Work on the issue** – Fork the repository, implement the solution, and open a **pull request** from your fork to the upstream repo (same as any contribution). The GitHub account that opens the PR must be the **same account** you use to sign in on BountyHub; otherwise you cannot claim the bounty.

2. **Submit your claim on BountyHub** – After your PR is open, go to BountyHub, find the bounty for that issue, and use **Submit Claim**. Paste the URL of your pull request. Your claim is then tied to that bounty and PR.

3. **Wait for review** – We review the PR and, when it meets the bounty requirements, we approve the claim on BountyHub. Payout is handled by BountyHub (see [Review and acceptance](#review-and-acceptance)).

You can work on and submit claims for multiple bounties at the same time; each claim is for one bounty/issue.

## Working on a bounty

Bounty work must meet the same standards as any contribution to SuperPlane:

- Follow our [Pull Requests](pull-requests.md) guide (branching, title format, description, sign-off).
- For integration work, follow [Component Implementation](component-implementations.md) and the [Integrations](integrations.md) guide.
- All commits must be [signed off](commit_sign-off.md).
- Code and behavior must meet our [Quality Standards](quality.md).

The [Integrations Board](https://github.com/orgs/superplanehq/projects/2/views/17) shows integration-related work; bountied issues are those with the `bounty` label.

## Review and acceptance

1. **Code review** – We review your PR as we do any contribution: code quality, tests, documentation, and project standards. We may ask for changes; address feedback in the PR.

2. **Functionality and UX review** – Our team verifies that the integration and components work as expected: triggers fire correctly, actions behave as described, and the experience matches our quality bar. We may request changes based on this review.

3. **Acceptance on BountyHub** – Once the PR meets the bounty requirements, we (as bounty creator) approve the claim on BountyHub. BountyHub then pays you according to their process (Stripe or PayPal). We usually merge the PR first, then accept the claim on BountyHub.

We aim to complete review within **one week** of you submitting the claim on BountyHub.

## Disputes

If your claim is rejected and you believe your PR does solve the bounty:

1. **Contact us first** – Reach out to maintainers via [Discord](https://discord.gg/KC78eCNsnw) or in the GitHub issue comments. Often a misunderstanding or missing feedback can be resolved there.

2. **BountyHub dispute process** – If we still disagree, use BountyHub’s process. In your BountyHub dashboard, go to **My Bounties**, find the rejected claim, and click **Open Dispute**. Provide details on why your PR resolves the bounty. BountyHub will open a chat with all parties and manually review the dispute; their decision is binding. You will be notified by email about the outcome. BountyHub’s dispute option may require your PR to be merged; see their documentation for current rules.

For payout methods, fees, and BountyHub’s terms, see [BountyHub](https://www.bountyhub.dev/) and their documentation (e.g. [claiming a bounty](https://www.bountyhub.dev/docs/claim-bounty)).
