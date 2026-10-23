# Team plan seats and billing

> Status: Draft. Billing, invitation, and membership rules are agreed. Two
> decisions remain open. They are listed at the end.
>
> Audience: Product, design, and engineering
>
> Date: October 23, 2026

## Goal

Organizations can buy Team seats, invite people by email, and reuse paid
capacity when members leave. Billing stays predictable as membership changes.

An organization pays for additional seats when it buys them. At renewal, it
pays only for current members who accepted their invitations. The
organization can see its current capacity, next renewal date, and expected bill.

This PRD defines product behavior. It does not prescribe an implementation.

## Scope

The first release covers:

- monthly Team subscriptions,
- seat purchases during a billing period,
- email invitations, with capacity rules and a fixed lifetime,
- removal of shared organization invite links on every plan and deployment,
- member removal and seat reuse,
- automatic seat reductions at renewal.

Deferred: annual billing, cancellation policy, and migration of existing
organizations above the Team maximum. The current Team plan has no included
AI credit. Two decisions remain open: access after a failed renewal payment,
and acceptance of invitations near renewal.

## Agreed rules

### Billing

| Topic | Rule |
| --- | --- |
| Billing account | One Team subscription per organization. All seat charges are billed to that organization. |
| Pricing | Charge per member seat. The owner also occupies a seat. |
| Billing interval | Monthly for the first release. |
| Team capacity | The Team plan allows a maximum of 50 purchased seats. The maximum is configurable in the plan catalog. |
| Admin seat override | An installation admin can change the plan maximum for one organization. An override never creates purchased capacity. |
| Paid capacity | Every seat must be purchased. The maximum is a hard limit, not an included allowance. |
| Billing managers | Organization Admins manage billing. Owners hold the Admin role. |
| Seat purchases | Buy capacity before inviting additional people. |
| Payment timing | Charge immediately when seats are purchased. |
| Mid-period purchases | Prorate additional seats for the remaining portion of the current billing period. |
| Payment confirmation | Additional seats become available only after successful payment. |
| Renewal date | Keep the billing cycle established by the initial subscription. Seat changes do not restart it. |
| Renewal quantity | Renew only seats occupied by current, accepted members. Pending invitations never count. |
| Unused capacity | Automatically remove unused seats at renewal and exclude them from the new period's charge. |
| Seat changes | Seat count changes only through SuperPlane. Customers do not change seats in the payment provider's portal. |

### Membership and invitations

| Topic | Rule |
| --- | --- |
| Service accounts | Service accounts do not occupy seats. Only human members occupy seats. |
| Invitations | Send invitations to specific email addresses. |
| Inviters | Organization Admins send invitations. Inviters and billing managers are the same group. |
| Invitation capacity | Each pending invitation uses one free purchased seat until it is accepted, canceled, or expires. An Admin can send any number of invitations, but pending invitations cannot exceed free seats. To invite more people, the Admin buys seats first. |
| Invitation role | The inviter selects the role. The default is Operator. |
| Invitation lifetime | An invitation expires 7 days after it is sent. Resend sends a new link and restarts the 7 days. |
| Joining | Sending or accepting an invitation does not create a separate charge or subscription. |
| Shared invite links | Shared organization invite links are removed for every plan and deployment. Email invitations replace them. |
| Deployments without email | An Admin can copy the link of an individual invitation and share it directly. |
| Member removal | Remove access immediately. Keep the paid seat available for reuse until renewal. |
| Seat reuse | A replacement member can use an existing available seat without another payment. |
| Pending invitations at renewal | Pending invitations expire at renewal. |

## Definitions

| Term | Meaning |
| --- | --- |
| Member | A person who belongs to the organization after joining or creating it. A removed person is no longer a member. |
| Service account | A non-human account used for automation. It is not a member for seat purposes. |
| Admin | A member with the Admin role. Admins manage billing and invitations. |
| Purchased seat | Capacity the organization paid for during the current billing period. |
| Occupied seat | A purchased seat used by a current member. The owner occupies one seat. |
| Available seat | A purchased seat without a current member. |
| Pending invitation | An invitation that its recipient has not accepted, that is not canceled, and that has not expired. It does not make the recipient a member. |
| Invitable seat | An available seat that no pending invitation uses. An Admin can send one invitation for each invitable seat. |
| Unused seat | Purchased capacity without a current member at renewal. Seats used by pending invitations are unused for renewal billing. |
| Plan maximum | The largest number of seats the organization can purchase on its plan. |
| Renewal | The end of the current subscription period and the start of the next monthly period. |
| Renewal quantity | The number of current members at renewal, owner included. The minimum is one. |

Member counts do not depend on recent activity. A person who remains a member
still occupies a seat even if they do not sign in during the month.

Seat values at any moment:

| Value | Formula |
| --- | --- |
| Available seats | purchased seats − occupied seats |
| Invitable seats | purchased seats − occupied seats − pending invitations |
| Purchasable seats | plan maximum − purchased seats |
| Renewal quantity | occupied seats (minimum 1) |

## Product requirements

### Start a Team subscription

- Only an Admin can start a Team subscription.
- Show the monthly price per seat and the selected seat count before payment.
- The initial purchase must cover the owner and all existing members.
- The selected count must not exceed the plan maximum.
- Show the initial payment and the recurring monthly total.
- Successful activation establishes the organization's billing cycle.
- All seats follow that cycle, including seats purchased later.

### Buy additional seats

- Only an Admin can purchase seats.
- Show current purchased capacity and how many more seats the plan permits.
- Reject purchases that would exceed the plan maximum.
- Show the additional charge, the period it covers, and the updated monthly total.
- Clearly identify estimates and any tax treatment before confirmation.
- Charge for the remaining portion of the current period immediately.
- Make the additional capacity available after successful payment.
- If payment fails or remains unconfirmed, do not grant additional capacity.
- A repeated submission must not create an unintended duplicate purchase.
- Seat reductions do not happen during a period. They happen automatically at renewal.

The charge starts when the seat is purchased, even if nobody joins
immediately. It does not wait for an invitation or acceptance. Unused seats
purchased near renewal still follow the automatic reduction rule.

The displayed monthly total for all purchased seats is distinct from the
expected renewal charge. The latter includes only current, accepted members
and can change before renewal.

### Invite people by email

- An Admin sends an invitation to a specific email address and selects a
  role. The default role is Operator.
- On Team, sending requires at least one invitable seat. Example: with 9 of 10
  seats occupied and no pending invitations, the Admin can send 1 invitation.
  To send a second, the Admin buys 1 more seat first.
- An Admin can send any number of invitations over time. Pending invitations
  can never exceed free purchased seats.
- Sending an invitation never purchases a seat.
- When no invitable seat exists, explain why and offer the seat purchase.
  At the plan maximum, explain that the organization must remove a member or
  cancel an invitation first.
- The invitation email contains the organization name, the inviter, the role,
  the expiry date, and the acceptance link.
- Each acceptance link belongs to one invitation. It does not provide reusable
  access to the organization.
- An Admin can copy the acceptance link of an invitation. Deployments without
  email delivery use this to share invitations.
- An invitation expires 7 days after it is sent.

Invitation management shows the recipient, role, inviter, sent date, expiry
date, and status. It supports resending and canceling an invitation.

- Resending a pending invitation does not use another seat. It sends a new
  link, invalidates the previous link, and restarts the 7 days.
- Resending an expired invitation makes it pending again. It needs an
  invitable seat.
- Canceling an invitation, or its expiry, makes its seat invitable again.

### Accept an invitation

- The recipient must accept before becoming a member.
- Only a person signed in with the invited email address can accept. An
  invitation must not let another person join under a different email address.
- A recipient without an account can sign up through the invitation, even when
  open registration is disabled. This applies only to the invited email address.
- Acceptance adds the recipient with the invited role, without another charge.
- Repeated acceptance must not create another membership, occupied seat, or charge.
- Canceled and expired invitations admit no one.
- Acceptance must never admit a member without purchased capacity. If no seat
  is available, the recipient is not admitted and the invitation stays
  pending. Acceptance never purchases a seat.

### Remove shared invite links

- Shared organization invite links are removed for every plan and deployment.
- An existing shared link admits no new member. The page explains that the
  organization uses email invitations.
- An existing member who opens an old link reaches the organization.
- New organizations do not receive a shared link.
- Shared links no longer allow signup when open registration is disabled.
- Self-hosted administrators receive notice of this change before they upgrade.

### Remove a member and reuse a seat

- Removing a member ends that person's organization access immediately.
- The purchased seat remains available through the end of the paid period.
- Member removal does not produce an automatic prorated refund or billing credit.
- The organization can invite a replacement without another seat purchase.
- Removing and replacing a member does not change the renewal date.
- Member removal must remain possible if a billing service is unavailable.

Reusing a seat changes who occupies the capacity. It does not extend the paid
period or create a new allowance of time.

### Renew the subscription

- Determine the number of current, accepted members at renewal.
- Include the owner in that count.
- Exclude removed members, service accounts, and recipients of pending invitations.
- Set the new period's purchased capacity to that member count.
- Charge the monthly seat price for that quantity.
- Remove all remaining unused capacity automatically.
- Do not require an Admin to request a seat reduction.
- Do not retain unused capacity because an invitation is pending.
- Expire all pending invitations at renewal.

After renewal, adding another person requires a new seat purchase unless an
existing paid seat becomes available.

Acceptance and renewal must produce a consistent membership and billing
result. Concurrent actions must not create more members than the paid
capacity permits. How an acceptance just before renewal is handled is an open
decision.

## Plan catalog and Polar responsibilities

| SuperPlane and the plan catalog | Polar |
| --- | --- |
| Define the plan, monthly seat price, and configurable seat maximum. | Collect subscription and seat-purchase payments. |
| Manage invitations, membership, roles, and access. | Calculate proration and issue invoices. |
| Determine the purchased capacity and the quantity required at renewal. | Maintain the subscription billing cycle and bill the required quantity. |
| Explain current capacity, purchases, and expected renewal charges. | Handle applicable taxes and payment-method management. |
| Make all seat count changes. | Do not change seat counts or assign seats outside SuperPlane. |

The catalog offer and the charged subscription must agree on price, interval,
and seat count. A change to the catalog maximum must not be represented as
free purchased capacity.

Polar supports immediate prorated subscription changes and changes scheduled
for the next period. These capabilities support the selected billing policy.
See [Polar proration](https://polar.sh/docs/features/subscriptions/proration).

Polar seat-based products also offer seat assignment and seat changes in the
Polar customer portal. SuperPlane does not use Polar seat assignment. The
portal must not let customers change seat counts. If a seat count changes
outside SuperPlane, SuperPlane treats the new count as purchased capacity,
never exceeds it, and alerts the team.

## Information customers need

Billing and member management must make the following information clear:

- Monthly price per seat and the plan maximum.
- Current purchased seats and current members.
- Pending invitations and the seats they use.
- Seats available for new invitations.
- The next renewal date.
- The expected number of seats and charge at renewal, based on current membership.
- The immediate cost of a proposed purchase and its remaining paid period.
- Whether a payment succeeded, failed, or is awaiting confirmation.
- Why an invitation cannot be sent and whether a seat purchase can resolve it.
- That pending invitations do not renew and expire at renewal.

## Examples

All examples assume an illustrative price of $20 per seat per month. Amounts
exclude taxes and discounts; partial-period amounts are approximate.

### Grow during a month and remove a member

The subscription starts on November 1 and renews on December 1.

| Date | Event | Result |
| --- | --- | --- |
| November 1 | Start with the owner and one teammate. | Purchase two seats and pay $40. |
| November 8 | Buy one seat, then invite a third person. | Pay about $15.33 immediately. Capacity increases to three after payment. |
| November 10 | The third person accepts the invitation. | Three seats are occupied. No additional charge. |
| November 15 | Buy another seat and invite a fourth member, who accepts. | Pay about $10.67 immediately. The renewal date remains December 1. |
| November 22 | Remove one member. | Access ends. One of the four paid seats becomes reusable. |
| December 1 | Three members remain. | Renew three seats for $60 and remove the unused seat. |

If a replacement joins before December 1, the organization renews four seats
for $80 instead. The replacement does not create an additional November charge.

### Invitation capacity

An organization has 10 purchased seats and 9 members.

| Step | Result |
| --- | --- |
| The Admin invites one person. | Allowed. Invitable seats drop from 1 to 0. |
| The Admin tries to invite a second person. | Blocked. The page offers a seat purchase. |
| The Admin buys one seat. | Purchased seats are 11. Invitable seats are 1. |
| The Admin invites the second person. | Allowed. Invitable seats are 0. |
| The Admin cancels the first invitation. | Invitable seats are 1. |
| The first invitation would have expired after 7 days. | Same result as canceling. |

### Pending invitations at renewal

An organization has 10 purchased seats, 9 current members, and 1 pending
invitation. It renews 9 seats for $180. The pending invitation expires at
renewal. To add that person, an Admin buys a seat and sends a new invitation.

### Reach the plan maximum

An organization with 50 purchased seats cannot buy a fifty-first seat.
If one member leaves, the organization can reuse that seat before renewal.
If only 49 members remain at renewal, capacity falls to 49. It can then buy
one additional seat, subject to successful payment.

## Included credit

The current Team plan includes no AI credit. Credit delivery is outside this
release's scope.

For a future plan with included credit, the allowance grows with the number
of seats. Members share one organization credit pool. They do not receive
separate personal balances.

Partial-period grants, expiration, and adjustments after seat changes need
their own specification before a plan offers included credit.

## Acceptance scenarios

| Scenario | Required outcome |
| --- | --- |
| Start Team with fewer seats than current members. | Reject before payment. |
| Buy a seat midway through the period. | Charge the partial period immediately and keep the original renewal date. |
| Attempt to buy above the configured maximum. | Reject the purchase without a charge or additional capacity. |
| Purchase payment fails or remains unconfirmed. | Keep the previous capacity. |
| Submit the same purchase twice. | Make one purchase. |
| Invite someone with an invitable seat. | Send the invitation. No charge. |
| Invite someone without an invitable seat. | Require a seat purchase before the invitation can be sent. |
| Invite someone at the plan maximum with no invitable seat. | Explain that a member must be removed or an invitation canceled. |
| Resend a pending invitation. | Send a new link and restart the 7 days. Use no extra seat. |
| Cancel an invitation, or let it expire. | Make its seat invitable again. |
| Open an invitation after 7 days. | Admit no one. |
| Accept an email invitation with paid capacity available. | Add the intended recipient with the invited role and no charge. |
| Accept while signed in with a different email. | Admit no one. |
| Repeat an invitation acceptance. | Do not create another membership, occupied seat, or charge. |
| Use an old shared invite link on any plan. | Do not admit a new member. |
| Send invitations on a deployment without email delivery. | The Admin can copy each invitation link. |
| Remove a member during a paid period. | End access immediately and make the seat reusable. |
| Remove a member while the billing service is unavailable. | Removal succeeds. |
| Replace a removed member before renewal. | Reuse the existing seat without another charge. |
| Renew with unused seats and pending invitations. | Charge only for current, accepted members, reduce capacity to that count, and expire pending invitations. |
| Attempt to accept an invitation after renewal. | Admit no one. The invitation has expired. |
| Purchase or accept invitations concurrently. | Respect both purchased capacity and the configured maximum. |
| Change seat count outside SuperPlane. | Treat the new count as capacity, never exceed it, and alert the team. |

## Open questions

| # | Question | Context | Needed before |
| --- | --- | --- | --- |
| 1 | What access does an organization keep after a failed renewal payment? | Polar starts the new period, then charges. If the charge fails, the subscription becomes past due. Polar retries after 2, 7, 14, and 21 days, then revokes the subscription. Polar can keep benefits for 0, 2, 7, 14, or 21 days. Today SuperPlane removes paid access at the first failure, and the fallback plan has no seat limit. Options: keep member access for the retry window with no new invitations or purchases; keep access for a shorter grace period; or remove paid access at once. In every option, no new member can join. | Launch |
| 2 | How is an invitation acceptance just before renewal handled? | Polar renews with the seat count set before the period ends. An acceptance after the last update can leave more members than renewed seats. Options: update the next-period seat count on every acceptance and reject acceptance when that update fails near renewal; or block all acceptances in a short window before renewal. In every option, there is no automatic charge. | Renewal implementation |

## Deferred work

| Topic | Status |
| --- | --- |
| Annual billing | Deferred. The first release supports monthly billing only. |
| Cancellation policy | Deferred to separate work. |
| Existing organizations above the maximum | Deferred. This PRD does not define a migration policy. |
| Included credit | No current allowance. A future allowance scales with seats and uses a shared organization pool. |
| Removal of stored invite link data | Old shared links stop working in this release. Deleting their stored records is a later cleanup. |

## Related documents

- [Token usage and billing](token-usage-and-billing.md) describes the
  organization wallet and hosted usage program. This PRD specifies Team seats
  and subscriptions.
- [GitHub billing policy](https://docs.github.com/en/billing/concepts/impact-of-plan-changes)
  documents immediate prorated seat additions and reductions at the next renewal.
- [Polar proration](https://polar.sh/docs/features/subscriptions/proration)
  and [Polar failed payments](https://polar.sh/docs/features/subscriptions/failed-payments).
