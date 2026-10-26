# Team plan seats and billing

> Status: Draft. Billing, invitation, and membership rules are agreed. Five
> questions remain open. [Open questions](#open-questions) lists them with the
> shipped behavior, the options, and what each question blocks. Decided
> entries stay in that section as decision records.
>
> Audience: Product, design, and engineering
>
> Date: October 23, 2026. Updated: October 25, 2026.

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
- email invitations, with capacity rules and a default lifetime of 7 days,
- removal of shared organization invite links on every plan and deployment,
- member removal and seat reuse,
- automatic seat reductions at renewal.

Deferred: annual billing, cancellation policy, and migration of existing
organizations above the Team maximum. The current Team plan has no included
AI credit. Five questions remain open. They cover access after a failed
renewal payment, acceptance of invitations near renewal, seat counts lowered
outside SuperPlane, signup without email delivery, and seat purchases while a
cancellation is scheduled. See [Open questions](#open-questions).

Purchased seats are required only on per-seat plans, such as Team. Email
invitations and the removal of shared invite links apply to every plan and
deployment.

## Agreed rules

### Billing

The billing rules apply to per-seat plans, such as Team.

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
| Renewal quantity | Renew only seats occupied by current, accepted members. Pending invitations never count. Renewal never adds seats: the quantity is capped at the current paid seats. See [D14](#d14-renewal-quantity-when-over-capacity). |
| Unused capacity | Automatically remove unused seats at renewal and exclude them from the new period's charge. |
| Seat changes | Seat count changes only through SuperPlane. Customers do not change seats in the payment provider's portal. |

### Membership and invitations

| Topic | Rule |
| --- | --- |
| Service accounts | Service accounts do not occupy seats. Only human members occupy seats. |
| Invitations | Send invitations to specific email addresses. |
| Inviters | Organization Admins send invitations. Inviters and billing managers are the same group. |
| Invitation channels | Admins send and manage invitations on the Members page or through the API. SuperPlane has no CLI commands for invitations. |
| Invitation capacity on per-seat plans | Each pending invitation uses one free purchased seat until it is accepted, canceled, or expires. An Admin can send any number of invitations, but pending invitations cannot exceed free seats. To invite more people, the Admin buys seats first. |
| Invitations on other plans | Plans that are not per-seat need no purchased seats. Invitations and acceptance follow the plan's member limit, if the plan has one. With billing off, no seat or member limit applies. |
| Invitation role | The inviter selects the role. The default is Operator. |
| Invitation lifetime | An invitation expires 7 days after it is sent. Installation admins can change this lifetime. Resend sends a new link and restarts the lifetime. |
| Joining | Sending or accepting an invitation does not create a separate charge or subscription. |
| Shared invite links | Shared organization invite links are removed for every plan and deployment. Email invitations replace them. |
| Deployments without email | An Admin can copy the link of an individual invitation and share it directly. Shipped difference: signup through that link still needs an emailed code. See [D15](#d15-signup-without-email-delivery). |
| Member removal | Remove access immediately. Keep the paid seat available for reuse until renewal. |
| Seat reuse | A replacement member can use an existing available seat without another payment. |
| Pending invitations at renewal | Pending invitations stay pending at renewal. They never count toward the renewal quantity, so renewal keeps no seat for them. After renewal, acceptance needs a free seat. |

## Definitions

Seat terms (purchased, occupied, available, invitable, unused, and renewal
quantity) apply to per-seat plans.

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
| Renewal quantity | The number of current members at renewal, owner included. The minimum is one. The maximum is the current paid seats. See [D14](#d14-renewal-quantity-when-over-capacity). |

Member counts do not depend on recent activity. A person who remains a member
still occupies a seat even if they do not sign in during the month.

Seat values at any moment:

| Value | Formula |
| --- | --- |
| Available seats | purchased seats − occupied seats |
| Invitable seats | purchased seats − occupied seats − pending invitations |
| Purchasable seats | plan maximum − purchased seats |
| Renewal quantity | min(max(occupied seats, 1), purchased seats) |

> The renewal quantity is defined only when the purchased seats are 1 or
> more. Zero or missing purchased seats are an invalid state that needs
> repair. That state has no renewal quantity. See
> [D14](#d14-renewal-quantity-when-over-capacity).

## Product requirements

### Start a Team subscription

- Only an Admin can start a Team subscription.
- Show the monthly price per seat and the selected seat count before payment.
- The initial purchase must cover the owner and all existing members.
- The selected count must not exceed the plan maximum.
- The default count also covers pending invitations, up to the plan maximum
  ([D13](#d13-checkout-default-with-pending-invitations)).
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
- On per-seat plans, sending requires at least one invitable seat. Example:
  with 9 of 10 seats occupied and no pending invitations, the Admin can send 1
  invitation. To send a second, the Admin buys 1 more seat first.
- On per-seat plans, an Admin can send any number of invitations over time.
  Pending invitations can never exceed free purchased seats.
- On plans that are not per-seat, sending needs no purchased seat. The plan's
  member limit applies, if the plan has one. With billing off, no limit applies.
- Sending an invitation never purchases a seat.
- On per-seat plans, when no invitable seat exists, explain why and offer the
  seat purchase. At the plan maximum, explain that the organization must
  remove a member or cancel an invitation first.
- The invitation email contains the organization name, the inviter, the role,
  the expiry date, and the acceptance link.
- Each acceptance link belongs to one invitation. It does not provide reusable
  access to the organization.
- An Admin can copy the acceptance link of an invitation. Deployments without
  email delivery use this to share invitations. Signup through that link still
  needs an emailed code. See
  [D15](#d15-signup-without-email-delivery).
- An invitation expires 7 days after it is sent. Installation admins can
  change this lifetime.

Invitation management shows the recipient, role, inviter, sent date, expiry
date, and status. It supports resending and canceling an invitation.

- Resending a pending invitation sends a new link, invalidates the previous
  link, and restarts the lifetime.
- Resending an expired invitation makes it pending again.

On per-seat plans:

- Resending a pending invitation does not use another seat.
- Resending an expired invitation needs an invitable seat.
- Canceling an invitation, or its expiry, makes its seat invitable again.

### Accept an invitation

- The recipient must accept before becoming a member.
- Only a person signed in with the invited email address can accept. An
  invitation must not let another person join under a different email address.
- A recipient without an account can sign up through the invitation, even when
  open registration is disabled. This applies only to the invited email address.

  > Shipped difference: invitation signup always requires a code that
  > SuperPlane sends by email. The invitation link alone is not proof of the
  > mailbox. See [D15](#d15-signup-without-email-delivery).

- Acceptance adds the recipient with the invited role, without another charge.
- Repeated acceptance must not create another membership, occupied seat, or charge.
- Canceled and expired invitations admit no one.
- On per-seat plans, acceptance must never admit a member without purchased
  capacity. If no seat is available, the recipient is not admitted and the
  invitation stays pending. Acceptance never purchases a seat.
- On plans that are not per-seat, acceptance needs no purchased seat. The
  plan's member limit applies, if the plan has one. With billing off, no
  limit applies.

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
- Set the new period's purchased capacity to that member count, at least one
  and at most the current paid seats. Renewal never adds seats.
- When the paid seat count is zero or missing, send no count. That state needs
  repair. See [D14](#d14-renewal-quantity-when-over-capacity).

- Charge the monthly seat price for that quantity.
- Remove all remaining unused capacity automatically.
- Do not require an Admin to request a seat reduction.
- Do not retain unused capacity because an invitation is pending.
- Keep pending invitations pending. Renewal keeps no seat for them.

After renewal, adding another person requires a new seat purchase unless an
existing paid seat becomes available. This also applies to the recipient of a
pending invitation. Until a seat is free, acceptance admits no one, and the
invitation stays pending.

Acceptance and renewal must produce a consistent membership and billing
result. Concurrent actions must not create more members than the paid
capacity permits. How an acceptance just before renewal is handled is an
[open decision](#d8-accepting-invitations-near-renewal).

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
outside SuperPlane, SuperPlane treats the new count as purchased capacity and
alerts the team. A higher count is paid capacity. What happens when a lower
count leaves fewer seats than current members is an
[open question](#d11-outside-seat-reduction-below-the-member-count).

### Admin-assigned plans without a Polar subscription

**Locked decision:** Business Manual (`business_manual`) and custom plans
assigned by an installation admin, such as SuperPlane only, can be active
without a Polar subscription. This is intentional product behavior. Do not
require a subscription to preserve these assignments.

- The Billing page must show the assigned catalog plan as the current plan.
- Opening Billing, running subscription sync, or processing a webhook must
  not replace an admin-assigned plan with None because Polar has no live
  subscription. Old canceled subscriptions and retained subscription IDs
  are history, not authority to remove the manual assignment.
- Preserve the assigned plan, its access, and its credit-purchase permission
  when an inactive subscription snapshot arrives. Apply this rule whether
  the organization retains an old subscription ID or has no subscription ID.
- A live Polar subscription can still replace the admin assignment. Keep
  the existing subscription identity and event-ordering checks. A scheduled
  cancellation with an open paid period still counts as live.

Regression coverage must verify that an old canceled subscription cannot
clear Business Manual or a custom admin-assigned plan. Keep coverage that
verifies activation of a live Polar subscription.

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
- That pending invitations do not count at renewal, and that acceptance after
  renewal needs a free seat.

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
invitation. It renews 9 seats for $180. The invitation stays pending, but
renewal keeps no seat for it. If the recipient tries to accept, SuperPlane
admits no one and explains that no seat is available. To add that person, an
Admin buys a seat. The recipient can then accept the same invitation before it
expires.

### Reach the plan maximum

An organization with 50 purchased seats cannot buy a fifty-first seat.
If one member leaves, the organization can reuse that seat before renewal.
If only 49 members remain at renewal, capacity falls to 49. It can then buy
one additional seat, subject to successful payment.

## Included credit

The current Team plan includes no AI credit. Credit delivery is outside this
release's scope.

For a future plan with included credit, the allowance grows with the number
of successfully paid seats. Each paid seat adds its allowance to the
organization ledger, even before a member occupies it. Delivery depends on
purchased-seat tracking. Members share one organization credit pool. They do
not receive separate personal balances.

Partial-period grants, expiration, and adjustments after seat changes need
their own specification before a plan offers included credit.

## Acceptance scenarios

Scenarios apply to per-seat plans unless the scenario names another plan or
billing state.

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
| Resend a pending invitation. | Send a new link and restart the lifetime. Use no extra seat. |
| Cancel an invitation, or let it expire. | Make its seat invitable again. |
| Open an invitation after its lifetime ends. | Admit no one. |
| Accept an email invitation with paid capacity available. | Add the intended recipient with the invited role and no charge. |
| Accept while signed in with a different email. | Admit no one. |
| Repeat an invitation acceptance. | Do not create another membership, occupied seat, or charge. |
| Use an old shared invite link on any plan. | Do not admit a new member. |
| Send invitations on a deployment without email delivery. | The Admin can copy each invitation link. |
| Remove a member during a paid period. | End access immediately and make the seat reusable. |
| Remove a member while the billing service is unavailable. | Removal succeeds. |
| Replace a removed member before renewal. | Reuse the existing seat without another charge. |
| Renew with unused seats and pending invitations. | Charge only for current, accepted members, reduce capacity to that count, and keep pending invitations pending. |
| Accept a pending invitation after renewal with no free seat. | Admit no one. Keep the invitation pending. |
| Accept a pending invitation after renewal and a seat purchase. | Add the recipient with the invited role. No other charge. |
| Purchase or accept invitations concurrently. | Respect both purchased capacity and the configured maximum. |
| Raise the seat count outside SuperPlane. | Treat the new count as purchased capacity and alert the team. |
| Lower the seat count outside SuperPlane below current members. | Alert the team. The access outcome is an [open question](#d11-outside-seat-reduction-below-the-member-count). Shipped behavior: members keep access, and invitations and acceptance are blocked. |
| Invite and accept on a plan that is not per-seat. | Require no purchased seat. Apply the plan's member limit, if any. |
| Invite and accept with billing off. | Apply no seat or member limit. |

## Open questions

This section is the single list of unresolved decisions. Each entry states
the question, the shipped behavior, the options with their tradeoffs, and what
the question blocks. This section does not choose an option. A decided entry
stays in place as a decision record so that links to it still resolve. The
table lists only unresolved decisions.

Task IDs (11, 05, 13b, 17, 19, 26, 27b, 32) come from the outside backlog and
appear as written there.

| ID | Title | Blocks |
| --- | --- | --- |
| [D7](#d7-grace-after-a-failed-renewal-payment) | Grace after a failed renewal payment | Launch |
| [D8](#d8-accepting-invitations-near-renewal) | Accepting invitations near renewal | 13b |
| [D11](#d11-outside-seat-reduction-below-the-member-count) | Outside seat reduction below the member count | Launch. Also decides whether 17 stays dropped. Affects 11 and 05. |
| [D15](#d15-signup-without-email-delivery) | Signup without email delivery | 26 |
| [09-a](#09-a-seat-purchase-while-a-cancellation-is-scheduled) | Seat purchase while a cancellation is scheduled | No task is named |

### D7. Grace after a failed renewal payment

**Question.** For how long do existing members keep Team access after a
renewal payment fails?

**Context.** Polar starts the new period, then charges. If the charge fails,
the subscription becomes past due. Polar retries after 2, 7, 14, and 21 days,
then revokes the subscription. Polar can keep benefits for 0, 2, 7, 14, or 21
days.

**Shipped today.**

- The installation setting `PastDueGraceDays` sets the grace period. The
  default is 0 days. The maximum is 21 days. An installation admin changes it
  through the admin API (`pkg/models/installation_metadata.go`,
  `pkg/public/admin.go`).
- At 0 days, the paid plan lapses at once. `ApplyPolarSubscription` moves the
  plan to the fallback plan when it stores the past-due snapshot.
  `lapsePaymentBlockedCatalog` does the same for a past-due row that has no
  recorded failure start (`pkg/models/organization_billing_plan.go`).
- Above 0 days, the paid plan stays until the failure start plus the grace
  period. After that, `lapseEndedPastDueGrace` moves the plan to the fallback
  plan.
- While the subscription is past due, seats lock at the occupied count. No
  invitation can be sent, no invitation can be accepted, and no seat can be
  bought (`pkg/entitlements/seats.go`, and `invitations.go`,
  `accept_invitation.go`, and `purchase_seats.go` in
  `pkg/grpc/actions/organizations/`). This applies with and without a grace
  period.

**Options.**

1. Keep the grace period at 0 days. Paid access ends at the first failed
   payment.
   - Tradeoff: The rule is simple, and the organization does not use Team
     without payment.
   - Tradeoff: Members lose Team features after one failed charge, even when a
     Polar retry would succeed.
2. Keep access for the whole Polar retry window, up to 21 days.
   - Tradeoff: Members keep working while the payment contact fixes the
     payment method.
   - Tradeoff: The organization uses Team for up to 21 days without payment.
3. Keep access for a shorter grace period, for example 2, 7, or 14 days.
   - Tradeoff: The exposure is smaller than option 2, and members have time to
     react.
   - Tradeoff: Access can still end before the last Polar retry succeeds.

In every option, no new member can join while the subscription is past due.

**Blocks.** Launch.

### D8. Accepting invitations near renewal

**Question.** Does SuperPlane block invitation acceptance in a window before
renewal, or while the stored billing period has expired or cannot be verified?
If it does, how does acceptance reopen?

**Context.** Polar renews with the seat count set before the period ends. An
acceptance after the last update can leave more members than renewed seats.

**Shipped today.**

- `Accept` has no pre-renewal window. It does not check the stored billing
  period (`accept_invitation.go`).
- After a member joins, `seatrenewal.ScheduleAsync` tries to set the
  next-period seat count in Polar. It makes up to three attempts in about ten
  seconds (`pkg/billing/seatrenewal/async.go`).
- `SeatRenewalSyncWorker` checks and repairs the next-period seat count every
  5 minutes (`pkg/workers/seat_renewal_sync_worker.go`).
- Only a seat purchase rejects an unknown or expired billing period
  (`purchase_seats.go`).

**Options.**

1. Keep the shipped behavior: no window and no period check.
   - Tradeoff: Invitees never wait because of renewal timing.
   - Tradeoff: If the Polar update and the worker both fail before renewal,
     Polar renews with fewer seats than members. This leads to the state in
     [D11](#d11-outside-seat-reduction-below-the-member-count).
2. Block acceptance in a window before renewal. It reopens when the renewal
   snapshot stores the new period.
   - Tradeoff: The window limits the exposure to a late update.
   - Tradeoff: Invitees wait during the window. The length of the window
     needs a value.
3. Block acceptance while the stored period has expired or cannot be verified.
   It reopens when a webhook or billing sync stores a current period.
   - Tradeoff: Acceptance cannot run on stale billing data.
   - Tradeoff: A late webhook or a Polar outage blocks joins that were valid.
4. Make acceptance depend on the next-period seat count update near renewal.
   Acceptance is rejected when the update fails, and it reopens when the
   update succeeds.
   - Tradeoff: The renewed seat count always covers the new member.
   - Tradeoff: A Polar outage near renewal blocks every acceptance.

In every option, there is no automatic charge.

**Blocks.** 13b.

### D11. Outside seat reduction below the member count

**Question.** What happens when the Polar seat count falls below the number of
members?

**Context.** Seat counts must change only through SuperPlane, but the payment
provider's portal or a support action can still lower them. Example: 10 seats
and 10 members, then the count drops to 9.

**Shipped today.**

- SuperPlane alerts the team. `UpdateSeatCapacityState` records
  `OverCapacitySince`, and `pkg/billing/polar/seat_capacity.go` writes the
  `billing.seats_over_capacity` log entry.
- Members keep access.
- Invitations and acceptance are blocked, because the stored seat count is
  lower than the member count.
- Seat purchase stays allowed.

**Options.**

1. Restore the previous count in Polar.
   - Tradeoff: Members and billing stay consistent with what the organization
     bought.
   - Tradeoff: SuperPlane changes a Polar subscription without a customer
     action. The customer or support can see this as an unexpected change.
2. Keep access for all current members. Block new joins and invitations until
   capacity exceeds the member count.
   - Tradeoff: No member loses access.
   - Tradeoff: The organization uses more seats than it pays for until an
     Admin buys seats or removes members.
3. Remove access for one member.
   - Tradeoff: Members never exceed paid capacity.
   - Tradeoff: SuperPlane must choose the member to remove. The removal can
     interrupt work without a customer action.

In every option, SuperPlane alerts the team.

**Blocks.** Launch. The answer also decides whether 17 stays dropped. It
affects 11 and 05.

### D13. Checkout default with pending invitations

**Decided.** 2026-10-25. Option 1, with a cap at the plan maximum.

**Question.** Does the default seat count at Team checkout include pending
invitations?

**Context.** Plans that are not per-seat need no seat for an invitation.
Invitations sent before Team therefore hold no seat. After checkout,
acceptance admits no one until an Admin buys seats.

**Decision.**

- The default seat count is `min(maximum, members + pending)`. With no plan
  maximum, the default is `members + pending`.
- The minimum stays `max(members, 1)`. Pending invitations do not raise the
  minimum. The Admin can remove the seats for pending invitations before
  payment.
- When the plan maximum cannot cover all pending invitations, checkout tells
  the Admin how many pending invitations cannot get a seat.
- When members already exceed the plan maximum, checkout is rejected.
- SuperPlane does not expire pending invitations when Team starts.

**Implementation.** `BillingSeatCheckout.tsx` in
`web_src/src/pages/factories/pages/organizationSettings/` applies the default
and the copy. `validateCheckoutSeats` in `select_billing_plan.go` enforces the
minimum and the maximum on the server. The decision does not change the
checkout lifetime race; 33a and 33b own that work.

**Unblocks.** 19.

### D14. Renewal quantity when over capacity

**Decided.** 2026-10-25. Option 2 for valid paid seats, and the first choice
of option 3 for an invalid count.

**Question.** Which seat count does renewal use when members exceed paid seats,
or when the seat count in Polar is not valid?

**Context.** The earlier PRD formula for renewal quantity was "occupied seats
(minimum 1)". The scheduler capped that value at the paid seats, so the
renewal estimate and the scheduled count could differ. Example: 3 members and
1 paid seat gave an estimate of 3 seats, but the scheduler sent 1.

**Decision.**

- For valid paid seats (1 or more), the renewal quantity is
  `min(max(members, 1), current paid seats)`. Members are current, accepted
  human members, owner included. Pending invitations never count.
- Renewal never increases the seat count automatically. Only a seat purchase
  adds seats. A change to the plan maximum does not create purchased seats.
- When members exceed the paid seats, the organization stays over capacity at
  the next period. The billing page shows the billed seat count and a
  separate over-capacity warning. See
  [D11](#d11-outside-seat-reduction-below-the-member-count).
- Zero or missing stored paid seats are an invalid state that needs repair. That
  state has no renewal estimate, and the scheduler sends no count. The
  scheduler returns an error, and the sync worker tries again.
- The billing estimate and the scheduler use the same quantity from the stored
  plan. Before sending, the scheduler also caps that quantity at the paid seats
  in the Polar response. The estimate and scheduled count match when the stored
  plan and the Polar response report the same valid paid seat count.
- An older Polar response does not replace the newer stored plan, but its paid
  seat count still caps the scheduled count. For example, 9 members and 10
  stored paid seats give an estimate of 9. An older Polar response with 4 paid
  seats caps the scheduled count at 4. The next scheduler run reads Polar again.

**Implementation.** `RenewalSeats` and `ResolveRenewalSeats` in
`pkg/entitlements/seat_renewal.go` define the quantity from the stored plan.
They return `ErrInvalidPaidSeats` for zero or missing paid seats.
`resolvePlanSeats` in `pkg/entitlements/seats.go` uses the rule for the estimate, and
`describe_organization_billing.go` in `pkg/grpc/actions/organizations/` omits
`expected_renewal_seats` and `expected_renewal_cents` when the rule returns no
count. `loadLocalState` in `pkg/billing/seatrenewal/schedule.go` uses the same
rule for the initial scheduler target. `Scheduler.schedule` then applies
`min(*result.Target, *snapshot.Seats)` before sending the count to Polar. This
extra cap prevents renewal from adding seats above the Polar response count.
`BillingSeatsCard.tsx` in
`web_src/src/pages/factories/pages/organizationSettings/` shows the estimate,
the over-capacity explanation, and the unavailable estimate.

**Options considered.**

1. Renew all members. Send the occupied count, even above the paid seats.
   - Tradeoff: Billing matches the people who have access.
   - Tradeoff: The change adds seats without a purchase flow, so the PRD rule
     that seat changes need a SuperPlane purchase and payment confirmation
     does not hold for those seats. It can also exceed the plan maximum.
2. Cap the renewal quantity at the paid seats (chosen).
   - Tradeoff: Renewal never adds seats or charges for seats that the
     organization did not buy.
   - Tradeoff: Members above the cap stay in the over-capacity state at the
     next period. See
     [D11](#d11-outside-seat-reduction-below-the-member-count).
3. Handle an invalid Polar count.
   - Send nothing and retry (chosen). Tradeoff: The renewal can use a stale
     count when the count stays invalid until renewal.
   - Send the occupied count with no cap. Tradeoff: Renewal can add seats
     from bad data.
   - Block joins until the count is valid. Tradeoff: A Polar data problem
     blocks members who did nothing wrong.

**Unblocks.** 27b and 32.

### D15. Signup without email delivery

**Question.** Can an invitation link prove the invited email address in
self-hosted installs that have no email delivery? If yes, how does SuperPlane
detect this mode?

**Shipped today.**

- Invitation signup always requires an emailed code
  (`confirmInvitationEmailProof` in `pkg/authentication/authentication.go`).
  The invitation link is also shown to the Admin, so the link alone is not
  proof of the mailbox.
- No code detects an install without email delivery. The only signal is a nil
  email service (`pkg/services/build.go`). When owner setup is enabled,
  `BuildEmailService` returns an SMTP service from stored settings, so a
  non-nil service does not prove that email works.
- Without delivery, the Admin can copy and share the invitation link. A
  recipient without an account cannot finish signup, because no code reaches
  them.

**Options for the proof.**

1. Always require the emailed code (shipped).
   - Tradeoff: Whoever holds the link cannot create an account for the invited
     address without access to its mailbox.
   - Tradeoff: Self-hosted installs without email cannot onboard new people by
     invitation.
2. Allow invitation-link proof when the install has no email delivery.
   - Tradeoff: Self-hosted installs can onboard people with the shared link.
   - Tradeoff: Anyone who holds the link can create an account for the
     invited address without mailbox proof.

**Options for detection (only if option 2 applies).**

1. Use the nil email service as the signal.
   - Tradeoff: Needs no new setting.
   - Tradeoff: It does not cover the SMTP service that is built from stored
     settings and does not deliver.
2. Add an explicit installation setting that the installer sets.
   - Tradeoff: The signal is clear and not affected by startup order.
   - Tradeoff: Installers must set it, and a wrong value changes the proof
     rule.
3. Test email delivery and use the result.
   - Tradeoff: The signal follows real behavior.
   - Tradeoff: It needs a test send and a rule for transient failures.

**Blocks.** 26.

### 09-a. Seat purchase while a cancellation is scheduled

**Question.** Does buying seats stay blocked while a cancellation is
scheduled at the end of the period? This is shipped behavior. It needs
confirmation.

**Shipped today.**

- `checkSeatPurchase` rejects a purchase when `CancelAtPeriodEnd` is set. The
  message is "Resume the subscription before buying seats."
  (`pkg/grpc/actions/organizations/purchase_seats.go`).
- Renewal sync also skips these plans (`SkipSubscriptionCanceling` in
  `pkg/billing/seatrenewal/schedule.go`).

**Options.**

1. Keep purchases blocked.
   - Tradeoff: The organization never pays for seats that end with the
     subscription.
   - Tradeoff: The Admin must resume the subscription before inviting more
     people than the free seats allow.
2. Allow purchases.
   - Tradeoff: The organization can add members for the rest of the paid
     period without resuming.
   - Tradeoff: The organization pays a prorated charge for seats that end at
     the period end. Polar behavior for seat updates on a canceling
     subscription needs a check.

**Blocks.** No task is named in the issue.

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
