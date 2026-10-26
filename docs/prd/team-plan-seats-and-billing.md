# Team plan seats and billing

> Status: Draft. Billing, invitation, and membership rules are agreed. Four
> questions remain open. [Open questions](#open-questions) lists them with the
> shipped behavior, the options, and what each question blocks. Decided
> entries stay in that section as decision records.
>
> Audience: Product, design, and engineering
>
> Date: October 23, 2026. Updated: October 26, 2026.

## Goal

Organizations can buy Team seats, invite people by email, and reuse paid
capacity when members leave. Billing stays predictable as membership changes.

An organization pays for additional seats when it buys them. At renewal, it
keeps and pays for its paid seats. Seats change only when an Admin changes
them: an Admin buys seats at once, and an Admin removes seats for the next
period. The organization can see its current capacity, next renewal date, and
expected bill.

This PRD defines product behavior. It does not prescribe an implementation.

## Scope

The first release covers:

- monthly Team subscriptions,
- seat purchases during a billing period,
- email invitations, with capacity rules and a default lifetime of 7 days,
- removal of shared organization invite links on every plan and deployment,
- member removal and seat reuse,
- paid seats that renew unchanged,
- seat removal by an Admin, effective at renewal.

Deferred: annual billing, cancellation policy, and migration of existing
organizations above the Team maximum. The current Team plan has no included
AI credit. Four questions remain open. They cover access after a failed
renewal payment, seat counts lowered outside SuperPlane, signup without email
delivery, and seat purchases while a cancellation is scheduled. See
[Open questions](#open-questions).

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
| Renewal quantity | Renew the paid seats. Renewal never removes or adds seats automatically. When an Admin confirmed a seat removal, renew the pending quantity. See [D14](#d14-renewal-quantity-and-seat-removal). |
| Unused capacity | Keep unused paid seats. They stay available for reuse until an Admin removes them. |
| Seat removal | Only an Admin removes seats. The removal takes effect at renewal. It makes no current charge, refund, or credit. See [Remove unused seats](#remove-unused-seats). |
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
| Member removal | Remove access immediately. Keep the paid seat available for reuse. Member removal never changes the seat count. |
| Seat reuse | A replacement member can use an existing available seat without another payment. |
| Pending invitations at renewal | Pending invitations stay pending at renewal. Renewal keeps the paid seats, so a pending invitation keeps its seat unless an Admin removed seats. Pending invitations never disappear automatically. |

## Definitions

Seat terms (purchased, occupied, available, invitable, unused, pending
quantity, admission limit, and renewal quantity) apply to per-seat plans.

| Term | Meaning |
| --- | --- |
| Member | A person who belongs to the organization after joining or creating it. A removed person is no longer a member. |
| Service account | A non-human account used for automation. It is not a member for seat purposes. |
| Admin | A member with the Admin role. Admins manage billing and invitations. |
| Purchased seat | Capacity the organization paid for during the current billing period. |
| Paid capacity (P) | The number of purchased seats in the current period. The payment provider reports it. |
| Pending quantity (Q) | The seat count for the next period after an Admin confirmed a seat removal. Without a pending removal, Q equals P. |
| Admission limit (L) | The seat count that limits members and pending invitations now: L = min(P, Q). |
| Occupied seat | A purchased seat used by a current member. The owner occupies one seat. |
| Available seat | A seat below the admission limit without a current member. |
| Pending invitation | An invitation that its recipient has not accepted, that is not canceled, and that has not expired. It does not make the recipient a member. |
| Invitable seat | An available seat that no pending invitation uses. An Admin can send one invitation for each invitable seat. |
| Unused seat | Purchased capacity without a current member. It stays paid until an Admin removes it. |
| Plan maximum | The largest number of seats the organization can purchase on its plan. |
| Renewal | The end of the current subscription period and the start of the next monthly period. |
| Renewal quantity | The seat count of the next period: Q when an Admin confirmed a seat removal, otherwise P. See [D14](#d14-renewal-quantity-and-seat-removal). |

Member counts do not depend on recent activity. A person who remains a member
still occupies a seat even if they do not sign in during the month.

Seat values at any moment:

| Value | Formula |
| --- | --- |
| Admission limit (L) | min(P, Q). Without a pending removal, L = P. |
| Available seats | L − occupied seats |
| Invitable seats | L − occupied seats − pending invitations |
| Purchasable seats | plan maximum − P |
| Renewal quantity | Q. Without a pending removal, Q = P. |

> These values are defined only when the paid capacity is 1 or more. Zero or
> missing paid capacity is an invalid state that needs repair. That state has
> no renewal quantity. See
> [D14](#d14-renewal-quantity-and-seat-removal).

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
- Seat reductions never happen automatically. An Admin removes seats, and the
  removal takes effect at renewal. See [Remove unused seats](#remove-unused-seats).
- A purchase starts from the paid capacity P, not from the pending quantity Q.
  A confirmed purchase supersedes a pending removal. Example: with 10 paid
  seats and a pending quantity of 7, a purchase of 1 seat gives 11 paid seats,
  and the removal ends.
- A purchase needs `now + 10 minutes < renewal`. The boundary is strict.

The charge starts when the seat is purchased, even if nobody joins
immediately. It does not wait for an invitation or acceptance. Unused seats
stay paid and renew until an Admin removes them.

The displayed monthly total for all purchased seats is distinct from the
expected renewal charge. The latter uses the renewal quantity: the paid
seats, or the pending quantity after an Admin removal.

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
- While a plan checkout can still activate, acceptance must not take
  membership above the seat count that the checkout selected. The recipient
  is not admitted, the invitation stays pending, and nothing is charged. The
  recipient sees that the organization is changing its plan and asks an
  Admin. A checkout from before this rule counts too. A verified expiry or
  failure of the checkout ends the restriction.
- Acceptance of a deleted or blocked account admits no one. When an account
  deletion and an acceptance overlap, the result is one of these: no
  membership, or a membership that the deletion removes. A deleted account
  never keeps a membership, a role, or an occupied seat.
- On plans that are not per-seat, acceptance needs no purchased seat. The
  plan's member limit applies, if the plan has one. With billing off, no
  limit applies.

### Remove shared invite links

- Shared organization invite links are removed for every plan and deployment.
- An existing shared link admits no new member. The page explains that the
  organization uses email invitations.
- An existing member who opens an old link that is still enabled reaches the
  organization. This is the only historical-link exception. A disabled link
  admits no one, also not an existing member.
- New organizations do not receive a shared link.
- Shared links no longer allow signup when open registration is disabled.
- Self-hosted administrators receive notice of this change before they upgrade.

### Remove a member and reuse a seat

- Removing a member ends that person's organization access immediately.
- The purchased seat stays paid and available. Member removal never changes
  the seat count.
- Member removal does not produce an automatic prorated refund or billing credit.
- The organization can invite a replacement without another seat purchase.
- Removing and replacing a member does not change the renewal date.
- Member removal must remain possible if a billing service is unavailable.
  Account deletion follows the same rule for the organizations that it does
  not delete.

Reusing a seat changes who occupies the capacity. It does not extend the paid
period or create a new allowance of time.

### Remove unused seats

The rules apply to per-seat plans. Seat removal ships after the rules below
have an implementation. Until then, paid seats renew unchanged.

- Only an Admin removes seats. SuperPlane never removes seats on its own.
- The Admin selects the pending quantity Q for the next period. Confirmation
  requires `max(members, 1) ≤ Q < P`. Members are current, accepted human
  members, owner included.
- Pending invitations do not raise the minimum. They also do not disappear
  automatically. An invitation above the new limit stays pending, and its
  acceptance needs a free seat.
- On confirmation, the admission limit `L = min(P, Q)` applies at once.
  Invitations and acceptance use L.
- The removal takes effect at renewal. It makes no current charge, refund, or
  credit. The paid seats stay usable up to L until renewal.
- The Admin can change the pending quantity or cancel the removal.
- Schedule, change, cancel, and purchase need `now + 10 minutes < renewal`.
  The boundary is strict: at exactly 10 minutes before renewal, the action is
  rejected. Nothing is sent to the payment provider for a rejected action.
- A confirmed purchase uses P and supersedes the removal. See
  [Buy additional seats](#buy-additional-seats).
- While a removal or a purchase has an unresolved provider result, SuperPlane
  uses the most restrictive limit that the result can produce. It blocks a
  second removal or purchase. It shows a renewal estimate only from verified
  provider data, and it labels the value as an estimate.
- SuperPlane can cancel a removal that it scheduled to protect members, for
  example when it cannot verify the removal before the boundary. This
  protective cancellation restores Q = P for the next period. It never charges
  now, and it never sets more seats than P. It is not a repair of a count that
  was lowered outside SuperPlane. That repair is
  [D11](#d11-outside-seat-reduction-below-the-member-count), which stays open.

### Renew the subscription

- Renew the paid seats P. When an Admin confirmed a seat removal before the
  boundary, renew the pending quantity Q.
- Never remove or add seats automatically. Membership changes, pending
  invitations, and the plan maximum do not change the renewal quantity.
- Charge the monthly seat price for that quantity.
- Keep the renewal date.
- Keep pending invitations pending.
- When the paid seat count is zero or missing, send no count. That state needs
  repair. See [D14](#d14-renewal-quantity-and-seat-removal).
- The payment provider stays the source of truth for paid capacity, status,
  and period. SuperPlane reads them again to repair a missed renewal, payment
  failure, or recovery. That repair only reads the provider.

After renewal, adding another person requires a free seat below the admission
limit, or a new seat purchase. This also applies to the recipient of a pending
invitation. Until a seat is free, acceptance admits no one, and the invitation
stays pending.

Acceptance and renewal must produce a consistent membership and billing
result. Concurrent actions must not create more members than the renewed
capacity permits. This invariant is
[D8](#d8-accepting-invitations-near-renewal).

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
- The expected number of seats and charge at renewal: the paid seats, or the
  pending quantity after an Admin removal.
- A pending seat removal, its quantity, and the admission limit that it sets
  at once.
- The immediate cost of a proposed purchase and its remaining paid period.
- Whether a payment succeeded, failed, or is awaiting confirmation.
- Why an invitation cannot be sent and whether a seat purchase can resolve it.
- That paid seats renew unchanged until an Admin removes them.
- For an invitation recipient: why acceptance is not possible now, for
  example no free seat, a failed payment, or a plan checkout in progress.

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
| December 1 | Three members remain. | Renew four seats for $80. The spare seat stays paid and available. |

If an Admin removes one seat before 23:50 on November 30, the pending
quantity is three. The admission limit becomes three at once. The
organization renews three seats for $60 on December 1, with no November
refund or credit. If a replacement joins before December 1, it uses the spare
seat and creates no additional November charge.

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
invitation. It renews 10 seats for $200. The invitation keeps its seat, and
the recipient can accept it after renewal.

If an Admin confirms a pending quantity of 9 before renewal, the minimum is 9
because the pending invitation does not count. The admission limit becomes 9
at once. The invitation stays pending. If the recipient tries to accept,
SuperPlane admits no one and explains that no seat is available. The
organization renews 9 seats for $180.

### Spare capacity

An organization has 10 paid seats and 7 members. Three seats stay paid and
available at every renewal. An Admin can confirm a pending quantity from 7 to
9. A pending quantity of 6 is rejected, because it does not cover the members.

### Purchase while a removal is pending

An organization has 10 paid seats, 7 members, and a pending quantity of 7.
The Admin buys 1 seat. The purchase starts from the 10 paid seats, so the
organization has 11 paid seats. The purchase supersedes the removal, and the
organization renews 11 seats.

### Unresolved operations

A seat purchase from 10 to 11 seats loses its provider response. SuperPlane
keeps the purchase, does not grant the eleventh seat, and blocks another
purchase. A later verified provider read can show 11 paid seats. SuperPlane
then grants the capacity without another charge. The purchase stays
unconfirmed until evidence about that purchase settles it.

### Reach the plan maximum

An organization with 50 purchased seats cannot buy a fifty-first seat.
If one member leaves, the organization can reuse that seat before renewal.
Capacity stays at 50 at renewal unless an Admin removes a seat.

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
| Use an old shared invite link on any plan. | Do not admit a new member. An existing member reaches the organization only through an enabled link. |
| Send invitations on a deployment without email delivery. | The Admin can copy each invitation link. |
| Remove a member during a paid period. | End access immediately and make the seat reusable. Send no seat change. |
| Remove a member while the billing service is unavailable. | Removal succeeds. |
| Delete an account while the billing service is unavailable. | The deletion succeeds for every organization that it does not delete. |
| Replace a removed member before renewal. | Reuse the existing seat without another charge. |
| Renew with unused seats and pending invitations. | Charge for the paid seats, keep the unused seats, and keep pending invitations pending. |
| Remove seats as an Admin. | Accept only `max(members, 1) ≤ Q < P`. Apply `L = min(P, Q)` at once. Renew Q with no current charge, refund, or credit. |
| Remove, change, cancel, or buy at 10 minutes or less before renewal. | Reject the action. Send nothing to the payment provider. |
| Buy a seat while a removal is pending. | Start from P and end the removal. 10 paid seats, a pending quantity of 7, and 1 seat give 11. |
| Accept a pending invitation after renewal with no free seat. | Admit no one. Keep the invitation pending. |
| Accept a pending invitation after renewal and a seat purchase. | Add the recipient with the invited role. No other charge. |
| Accept while a plan checkout for fewer seats can still activate. | Admit no one. Keep the invitation pending. No charge. |
| Accept while the account is being deleted. | Either no membership, or a membership that the deletion removes. Never an active membership for a deleted account. |
| Delete an organization while its checkout can still activate. | Refuse the deletion. A verified expiry or failure allows it. |
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

Task IDs (11, 05, 13b, 17, 19, 26, 27b, 32, 9B1) come from the outside
backlog and appear as written there.

| ID | Title | Blocks |
| --- | --- | --- |
| [D7](#d7-grace-after-a-failed-renewal-payment) | Grace after a failed renewal payment | Launch |
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

**Decided.** 2026-10-26. Retained as an invariant.

**Question.** Does SuperPlane block invitation acceptance in a window before
renewal, or while the stored billing period has expired or cannot be verified?
If it does, how does acceptance reopen?

**Context.** The question came from automatic renewal reductions. Polar
renewed with a seat count that SuperPlane set from the members before the
period ended, so an acceptance after the last update could leave more members
than renewed seats. Renewal now keeps the paid seats. Only an Admin removal
lowers the next quantity, and its admission limit applies at once.

**Decision.**

- The invariant stays: acceptance and renewal must produce a consistent
  membership and billing result. Concurrent actions must not create more
  members than the renewed capacity permits.
- Acceptance uses the admission limit `L = min(P, Q)`. A confirmed removal
  lowers L before renewal, so the renewed quantity covers every member.
- The draft task 13b is superseded. The admission work of 9B1 owns the
  invariant. It keeps the surviving requirements of 13b: the race between
  acceptance and a removal near the boundary, and recovery when the provider
  result of a removal is unknown or the stored period cannot be verified.
- There is no automatic charge in any case.

**Shipped today.**

- `Accept` has no pre-renewal window. It checks the paid capacity under the
  organization lock (`accept_invitation.go`).
- No flow changes the Polar seat count automatically. Member removal,
  acceptance, account deletion, seat purchase settlement, purchase preview,
  billing sync, resume, and subscription webhooks send no seat change.
- `SeatRenewalSyncWorker` reads the Polar subscription every 5 minutes and
  applies it. It repairs a missed renewal, payment failure, or recovery. It
  never writes a seat count (`pkg/workers/seat_renewal_sync_worker.go`).
- Only a seat purchase rejects an unknown or expired billing period
  (`purchase_seats.go`).

**Superseded options.** The earlier options were: keep no window, block a
window before renewal, block while the stored period has expired or cannot be
verified, or make acceptance depend on a next-period seat update. All of them
assumed automatic reductions.

**Unblocks.** 9B1.

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

Option 1 is a repair of a count that was lowered outside SuperPlane. It is
different from the protective cancellation of a removal that SuperPlane
scheduled itself (see [Remove unused seats](#remove-unused-seats)). The
retained renewal policy does not decide this question.

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

### D14. Renewal quantity and seat removal

**Decided.** 2026-10-25. **Replaced.** 2026-10-26.

**Question.** Which seat count does renewal use? Does SuperPlane reduce
unused seats automatically, and how does an Admin remove seats?

**Decision of 2026-10-26.** Two business decisions replace the decision of
2026-10-25:

1. No automatic renewal reductions. Renewal keeps the paid seat quantity.
   SuperPlane never reduces seats automatically.
2. A manual seat removal takes effect at the next renewal, with no mid-period
   refund or credit. On confirmation, the lower admission limit
   `L = min(P, Q)` applies at once. P is the paid capacity, and Q is the
   pending renewal quantity.

These rules follow from the decisions:

- Confirmation requires `max(members, 1) ≤ Q < P`. Pending invitations do not
  raise the minimum and do not disappear automatically.
- A purchase starts from P and supersedes a pending removal. Example: 10 paid
  seats, a pending quantity of 7, and 1 purchased seat give 11 paid seats.
- Schedule, change, cancel, and purchase need `now + 10 minutes < renewal`.
- Renewal never increases the seat count automatically. Only a seat purchase
  adds seats. A change to the plan maximum does not create purchased seats.
- Zero or missing paid seats are an invalid state that needs repair. That
  state has no renewal estimate, and SuperPlane sends no count.
- Legacy cleanup clears only the verified automatic reductions that the
  retired scheduler sent. It keeps schedules that an Admin made.

See [Remove unused seats](#remove-unused-seats) for unresolved operations,
estimates, change and cancel, and protective cancellation.

**Superseded decision of 2026-10-25.** Option 2 for valid paid seats, and the
first choice of option 3 for an invalid count. The renewal quantity was
`min(max(members, 1), current paid seats)`. Members were current, accepted
human members, owner included, and pending invitations never counted. A
scheduler sent that quantity to Polar as a next-period seat update after each
membership change and every 5 minutes. Before it sent the quantity, it also
capped the quantity at the paid seats in the Polar response. The earlier PRD
formula was "occupied seats (minimum 1)". The options considered were: renew
all members (option 1), cap the renewal quantity at the paid seats (option 2,
chosen), and, for an invalid Polar count, send nothing and retry, send the
occupied count without a cap, or block joins (option 3).

**Shipped today.**

- The automatic scheduler is retired. No flow sends a next-period seat
  update. Reductions that the retired scheduler sent stay in Polar until
  legacy cleanup.
- `organization_billing_plans.renewal_seats_scheduled` and
  `renewal_seats_scheduled_at` keep the last count that the retired scheduler
  sent. Nothing writes them now.
- The billing estimate still uses the superseded formula through
  `RenewalSeats` in `pkg/entitlements/seat_renewal.go`. The estimate and the
  copy change with the seat removal presentation. Do not release the retained
  renewal policy before legacy cleanup and that presentation are complete.

**Evidence for the gates.**

- G5: the replaced D14 is recorded here. The renewal quantity, the estimate,
  and the copy show retained seats and Admin removal.
- G9: sandbox evidence shows an ordinary renewal that keeps the paid seats; a
  removal as a next-period update with no charge or credit; change, cancel,
  and same-count clearing; a purchase that clears a pending removal;
  past-due handling; and the 10-minute boundary.
- G10: actual invoices show a purchase that starts from P while a removal is
  pending, and a removal with no current charge, refund, or credit.
- G13: alert policies cover unresolved purchases, independent of the
  reconciliation results; unresolved removals; over-capacity lines that
  describe committed state only; and reconciliation failures. The retired
  signals `billing.seat_renewal_mismatch_fixed`,
  `billing.seat_renewal_blocked_by_purchase`, and the `renewal_impact` field
  no longer exist. Remove them from alert policies.

**Unblocks.** 9B1.

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
