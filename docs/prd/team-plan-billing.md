# Team plan seats and billing

> Status: Draft. Core billing rules agreed; remaining decisions are listed below.
>
> Audience: Product, design, and engineering
>
> Date: October 23, 2026

## Goal

Organizations can buy Team seats, invite people by email, and reuse paid
capacity when members leave. Billing stays predictable as membership changes.

An organization pays for additional seats when it buys them. At renewal,
it pays only for current members who accepted their invitations. The
organization can see its current capacity, next renewal date, and expected bill.

This PRD defines product behavior. It does not prescribe an implementation.

## Scope

The first release covers monthly Team subscriptions, seat purchases, email
invitations, member removal, seat reuse, and automatic seat reductions at renewal.

Annual billing, cancellation policy, and migration of existing organizations
above the Team limit are deferred. The current Team plan has no included AI
credit. Failed-renewal access remains an open question.

## Agreed rules

| Topic | Rule |
| --- | --- |
| Billing account | One Team subscription per organization. All seat charges are billed to that organization. |
| Pricing | Charge per member seat. The owner also occupies a seat. |
| Billing interval | Monthly for the first release. |
| Team capacity | The Team plan allows a maximum of 50 purchased seats. The maximum is configurable in the plan catalog. |
| Paid capacity | Every seat must be purchased. The maximum is a hard limit, not an included allowance. |
| Seat purchases | Buy capacity before inviting additional people. |
| Payment timing | Charge immediately when seats are purchased. |
| Mid-period purchases | Prorate additional seats for the remaining portion of the current billing period. |
| Payment confirmation | Additional seats become available only after successful payment. |
| Renewal date | Keep the billing cycle established by the initial subscription. Seat changes do not restart it. |
| Invitations | Send invitations to specific email addresses. Replace reusable shared organization invite links. |
| Joining | Sending or accepting an invitation does not create a separate charge or subscription. |
| Member removal | Remove access immediately. Keep the paid seat available for reuse until renewal. |
| Seat reuse | A replacement member can use an existing available seat without another payment. |
| Renewal quantity | Renew only seats occupied by current, accepted members. Pending invitations do not count. |
| Unused capacity | Automatically remove unused seats at renewal and exclude them from the new period's charge. |

## Definitions

| Term | Meaning |
| --- | --- |
| Member | A person who belongs to the organization after joining or creating it. A removed person is no longer a member. |
| Purchased seat | Capacity the organization paid for during the current billing period. |
| Occupied seat | A purchased seat used by a current member. The owner occupies one seat. |
| Pending invitation | An invitation that its recipient has not accepted. It does not make the recipient a member. |
| Unused seat | Purchased capacity without a current member. Capacity held for a pending invitation is unused for renewal billing. |
| Available seat | Purchased capacity that can be assigned to another person. Invitation reservations may affect availability within the period. |
| Plan maximum | The largest number of seats the organization can purchase on its plan. |
| Renewal | The end of the current subscription period and the start of the next monthly period. |

Member counts do not depend on recent activity. A person who remains a member
still occupies a seat even if they do not sign in during the month.

## Product requirements

### Start a Team subscription

- Show the monthly price per seat and the selected seat count before payment.
- The initial purchase must cover the owner and any existing members.
- The selected count must not exceed the plan maximum.
- Show the initial payment and the recurring monthly total.
- Successful activation establishes the organization's billing cycle.
- All seats follow that cycle, including seats purchased later.

### Buy additional seats

- Only a person authorized to manage billing can purchase seats.
- Show current purchased capacity and how many more seats the plan permits.
- Reject purchases that would exceed the configured maximum.
- Show the additional charge, the period it covers, and the updated monthly total.
- Clearly identify estimates and any tax treatment before confirmation.
- Charge for the remaining portion of the current period immediately.
- Make the additional capacity available after successful payment.
- If payment fails or remains unconfirmed, do not grant additional capacity.
- A repeated submission must not create an unintended duplicate purchase.

The charge starts when the seat is purchased, even if nobody joins immediately.
It does not wait for an invitation or acceptance. Unused seats purchased near
renewal still follow the same automatic reduction rule.

The displayed monthly total for all purchased seats is distinct from the
expected renewal charge. The latter includes only current, accepted members
and can change before renewal.

### Invite people by email

- An authorized inviter sends an invitation to a specific email address.
- The organization must have purchased sufficient capacity before sending it.
- The recipient must accept before becoming a member.
- An invitation must not let another person join under a different email address.
- Email acceptance links belong to individual invitations. They do not provide
  reusable access to the organization.
- Shared organization invite links must no longer admit new members to Team organizations.
- Joining must not purchase seats automatically.
- When capacity is unavailable, a billing administrator must purchase more seats.
- At the plan maximum, the organization must reuse capacity before adding members.
- Repeated acceptance must not consume another seat for an existing member.

Invitation management should show the recipient and status, and support
resending and canceling an invitation. Resending must not require another seat.

**Proposed invitation detail:** Reserve one available paid seat for each pending
invitation during the current period. Canceling or expiring an invitation
releases its reservation for reuse. Reservation and expiry details still need
confirmation; they must not increase renewal quantity.

### Remove a member and reuse a seat

- Removing a member ends that person's organization access immediately.
- The purchased seat remains available through the end of the paid period.
- Member removal does not produce an automatic prorated refund or billing credit.
- The organization can invite a replacement without another seat purchase.
- Removing and replacing a member does not change the renewal date.
- Member removal must remain possible if a billing service is unavailable.

Reusing a seat changes who occupies the capacity. It does not extend the
paid period or create a new allowance of time.

### Renew the subscription

- Determine the number of current, accepted members at renewal.
- Include the owner in that count.
- Exclude removed members and recipients of pending invitations.
- Set the new period's purchased capacity to that member count.
- Charge the monthly seat price for that quantity.
- Remove all remaining unused capacity automatically.
- Do not require an administrator to request a seat reduction.
- Do not retain unused capacity because an invitation is pending.

After renewal, adding another person requires a new seat purchase unless an
existing paid seat becomes available. A pending invitation cannot bypass this
requirement. The invitation's exact state after renewal remains open.

Acceptance and renewal must produce a consistent membership and billing result.
Concurrent actions must not create more members than the paid capacity permits.

These rules define renewal quantity. Access after a failed renewal payment
remains an open decision.

## Plan catalog and Polar responsibilities

| SuperPlane and the plan catalog | Polar |
| --- | --- |
| Define the plan, monthly seat price, and configurable seat maximum. | Collect subscription and seat-purchase payments. |
| Manage invitations, membership, and access. | Calculate proration and issue invoices. |
| Determine the purchased capacity and the quantity required at renewal. | Maintain the subscription billing cycle and bill the required quantity. |
| Explain current capacity, purchases, and expected renewal charges. | Handle applicable taxes and payment-method management. |

The catalog offer and the charged subscription must agree on price, interval,
and seat count. A change to the catalog maximum must not be represented as
free purchased capacity.

Polar supports immediate prorated subscription changes and changes scheduled
for the next period. These capabilities support the selected billing policy.
See [Polar proration](https://polar.sh/docs/features/subscriptions/proration).

## Information customers need

Billing and member management must make the following information clear:

- Monthly price per seat and the plan maximum.
- Current purchased seats and current members.
- Pending invitations and any capacity reserved for them.
- Capacity available for new invitations.
- The next renewal date.
- The expected number of seats and charge at renewal, based on current membership.
- The immediate cost of a proposed purchase and its remaining paid period.
- Whether a payment succeeded, failed, or is awaiting confirmation.
- Why an invitation cannot proceed and whether a seat purchase can resolve it.

Customers must understand that pending invitations do not renew paid seats.
The interface must also explain what happens to those invitations at renewal
once that behavior is decided.

## Examples

### Grow during a month and remove a member

Assume an illustrative price of $20 per seat per month. The subscription starts
on November 1 and renews on December 1. Amounts exclude taxes and discounts;
partial-period amounts are approximate.

| Date | Event | Result |
| --- | --- | --- |
| November 1 | Start with the owner and one teammate. | Purchase two seats and pay $40. |
| November 8 | Buy one seat before inviting a third person. | Pay about $15.33 immediately. Capacity increases to three after payment. |
| November 10 | The third person accepts the invitation. | Three seats are occupied. No additional charge. |
| November 15 | Buy another seat and add a fourth member. | Pay about $10.67 immediately. The renewal date remains December 1. |
| November 22 | Remove one member. | Access ends. One of the four paid seats becomes reusable. |
| December 1 | Three members remain. | Renew three seats for $60 and remove the unused seat. |

If a replacement joins before December 1, the organization renews four seats
for $80 instead. The replacement does not create an additional November charge.

### Pending invitations at renewal

An organization has five purchased seats, three current members, and two
pending invitations. It renews three seats. The pending invitations cannot
preserve the other two seats or cause charges.

Those recipients need purchased capacity to join after renewal. Whether the
existing invitations remain pending or must be replaced is still open.

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
| Buy a seat midway through the period. | Charge the partial period immediately and keep the original renewal date. |
| Attempt to buy above the configured maximum. | Reject the purchase without a charge or additional capacity. |
| Purchase payment fails or remains unconfirmed. | Keep the previous capacity. |
| Invite someone without available purchased capacity. | Require a seat purchase before the invitation can proceed. |
| Use an old shared invite link for a Team organization. | Do not admit a new member. |
| Accept an email invitation with paid capacity available. | Add the intended recipient without another charge. |
| Repeat an invitation acceptance. | Do not create another membership, occupied seat, or charge. |
| Remove a member during a paid period. | End access immediately and make the seat reusable. |
| Replace a removed member before renewal. | Reuse the existing seat without another charge. |
| Renew with unused seats and pending invitations. | Charge only for current, accepted members and reduce capacity to that count. |
| Attempt to accept an invitation after its capacity disappears at renewal. | Do not admit a member without purchased capacity or create an automatic charge. |
| Purchase or accept invitations concurrently. | Respect both purchased capacity and the configured maximum. |

## Open questions and deferred work

| Topic | Status |
| --- | --- |
| Failed renewal payment | Access behavior remains open. No grace-period feature is included in this scope. Resolve access behavior before launch. |
| Pending invitations after renewal | Only accepted members renew. Decide whether outstanding invitations expire or remain pending until capacity is purchased again. |
| Invitation reservations and expiry | Confirm the proposed reservation behavior, invitation lifetime, and how an expired invitation is renewed. |
| Invitation rollout outside Team | Confirm how replacement of shared invitations applies to other plans and deployments. |
| Included credit | No current allowance. Future allowance scales with seats and uses a shared organization pool. |
| Existing organizations above the maximum | Deferred. This PRD does not define a migration policy. |
| Cancellation policy | Deferred to separate work. |
| Annual billing | Deferred. The first release supports monthly billing only. |

The pending-invitation decisions must preserve the agreed rule: an unaccepted
invitation never increases the renewal quantity.

## Related documents

- [Token usage and billing](token-usage-and-billing.md) describes the organization
  wallet and hosted usage program. This PRD specifies Team seats and subscriptions.
- [GitHub billing policy](https://docs.github.com/en/billing/concepts/impact-of-plan-changes)
  documents immediate prorated seat additions and reductions at the next renewal.
