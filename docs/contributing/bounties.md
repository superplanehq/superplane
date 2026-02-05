# Integration Bounties

SuperPlane uses [BountyHub](https://www.bountyhub.dev/) to offer paid bounties on selected integration work. This guide explains how to find, claim, work on, and get paid for bounties, and how review and disputes work.

## Table of Contents

- [Overview](#overview)
- [Finding bounties](#finding-bounties)
- [Claiming a bounty](#claiming-a-bounty)
- [Working on a bounty](#working-on-a-bounty)
- [How to approach bounty issues](#how-to-approach-bounty-issues)
- [Review and acceptance](#review-and-acceptance)
- [Disputes](#disputes)

## Overview

We use [BountyHub](https://www.bountyhub.dev/) for bounties (see their site for payout methods, fees, and how it works).

**Only issues with the `bounty` label** are on BountyHub. If an issue does not have that label, there is no bounty attached.

Each bounty issue has a **Bounty Details** section in its description (reward amount and any additional requirements or acceptance criteria). Read it before you start.

We may close an issue or withdraw a bounty; work in progress may not be paid if the bounty is no longer active.

## Finding bounties

Browse open bounties at [superplane.com/bounties](https://superplane.com/bounties/); each listing links to the corresponding GitHub issue. Confirm the bounty exists and is funded on BountyHub before investing significant time. Before you start, sign in to BountyHub with your GitHub account and connect Stripe (or add your PayPal email) so you can receive payouts when your claim is accepted.

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

## How to approach bounty issues

Human judgment and polish must drive the work. We welcome and encourage using AI tools to implement bounties, however—at the end of the day, code quality and the UX are what matter. Your contribution should meet our standards (see the guides linked above).

Issue descriptions and specs define the requirements, but we expect you to go beyond the literal spec:

- **Understand the tool** you’re integrating: its API, behavior, and typical use cases.
- **Think about the context** of the SuperPlane app and the user experience—how the integration fits into workflows and the UI.
- **Take freedom to explore and propose solutions**: you’re not limited to the spec as written. Suggest improvements, better UX, or alternative approaches when they make sense, and explain your choices in the PR.

## Review and acceptance

**Required for review:** Pull requests for integration and component bounties **must** include a **video** showing the working integration and components. PRs that do not include such a video will be **automatically closed** and will not be reviewed for bounty acceptance.

**Video requirement:** The video must show the component actually working as intended—for example, triggers firing, actions executing, and the correct user flow. A short, focused screen recording is sufficient.

Once your PR includes the required video, we review as follows:

1. **Functionality and UX review** – Our team verifies that the integration and components work as expected: triggers fire correctly, actions behave as described, and the experience matches our quality bar. We may request changes based on this review.

2. **Code review** – We review your PR as we do any contribution: code quality, tests, documentation, and project standards. We may ask for changes; address feedback in the PR.

3. **Acceptance on BountyHub** – Once the PR meets the bounty requirements, we (as bounty creator) approve the claim on BountyHub. BountyHub then pays you according to their process (Stripe or PayPal). We usually merge the PR first, then accept the claim on BountyHub.

We aim to complete review within **one week** of you submitting the claim on BountyHub.

## Disputes

We do our best to guide you and provide feedback in pull requests so your work can meet the bounty requirements. We only pay out bounties for **merged** PRs, and we never reject claims for PRs we merge. If you believe your claim was rejected despite your PR being merged—an error may have occurred—please reach out via [Discord](https://discord.gg/KC78eCNsnw). As an alternative, BountyHub has a [dispute process](https://www.bountyhub.dev/) for such cases; see their documentation for current rules.

For payout methods, fees, and BountyHub’s terms, see [BountyHub](https://www.bountyhub.dev/) and their documentation (e.g. [claiming a bounty](https://www.bountyhub.dev/docs/claim-bounty)).
