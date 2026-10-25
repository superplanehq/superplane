# Configure a monthly Team plan

Installation admins configure per-seat plans in **Billing plans**. The catalog
stores the maximum purchasable seats in `seat_limit`. The default maximum is
50. Valid maxima are integers from 1 through 1,000.

## Configure Polar for each environment

Use the [Polar sandbox dashboard](https://sandbox.polar.sh) for development.
Use the [Polar production dashboard](https://polar.sh) for production.
Products, product IDs, and organization access tokens are separate in each
environment. Sandbox and production tokens are not interchangeable.

1. Create a Polar organization access token with the `products:read` scope.
   Existing billing features need the additional scopes in `.env.example`.
2. Set `POLAR_ACCESS_TOKEN` to that environment's token.
3. Set `POLAR_ENVIRONMENT=sandbox` for development or `production` for production.
4. Restart SuperPlane after configuration changes.

Keep the default API version unless you test another supported version in
sandbox. Remove a local `POLAR_API_BASE_URL` override before production use.
Do not use `POLAR_BUSINESS_PRODUCT_ID` for Team. That setting links the legacy
Business product.

### Gate G1: Allow one subscription per organization

Before rollout, disable **Allow multiple subscriptions** in the Polar organization
settings. Apply this setting in both sandbox and production. Verify both settings
before you enable Team checkout.

SuperPlane also protects the stored subscription identity. A subscription that is
not live cannot replace a different live subscription. A live subscription includes
`active`, `trialing`, `past_due`, and scheduled cancellation with an open period.

## Configure the product and catalog plan

1. Create an active recurring Polar product with a monthly interval and interval count 1.
2. Add exactly one active USD seat-based price.
3. Configure one fixed-rate tier starting at seat 1.
4. Set its maximum to cover the catalog maximum, or leave the Polar tier unlimited.
5. Set `price_per_seat` to the catalog price in cents. Do not add other active charges or variable tiers.
6. Open **Billing plans** as an installation admin and select **Create plan**.
7. Enter the name **Team**, a unique key, category **Paid**, and visibility **Public**.
8. Enable **Per seat**. The empty maximum becomes 50. An existing maximum stays unchanged.
9. Set **Price (USD)** to the same per-seat monthly price configured in Polar.
10. Set **Maximum seats** to the required maximum and enter that environment's Polar product ID.
11. Save the plan.

For example, a $20 monthly seat price requires catalog `price_cents = 2000`
and Polar tier `price_per_seat = 2000`. Polar exposes this price through
[`seat_tiers`](https://github.com/polarsource/polar-js/blob/main/src/models/components/productpriceseatbased.ts),
not the flat-price `price_amount` field.

Public visibility makes the plan available under **Settings → Billing → Payment method**.
Billing managers select a seat count there and start checkout with **Subscribe to Team**.
No migration seeds Team because each environment has its own product ID.

Each linked per-seat save fetches the product, including saves that change
only the maximum. Missing products and product mismatches return HTTP 400.
Unconfigured Polar returns 400. Provider failures return 502. Rejected saves
keep entered editor values and leave the stored plan unchanged.

After linking, the product ID, price, and per-seat setting cannot change.
Create a new plan and product for a new price. The maximum can change within
the valid range and the Polar tier's capacity. Unlinked per-seat plans remain
allowed and still require a valid maximum.

## Local development defaults

`make dev.setup` seeds the sandbox catalog in `superplane_dev` outside CI.
Run `make dev.setup.billing.catalog` to apply these defaults separately.
The target requires the Docker `db` service and the Polar sandbox.

| Plan | Monthly price | Maximum seats | Sandbox product ID |
| --- | --- | --- | --- |
| Team | $50 per seat | 50 | `633e8026-988b-44de-8d65-31829948c975` |
| Starter | $20 | Unlimited | `abc9ef98-20c0-48b5-8204-1448df733737` |
| Pay-as-you-go | Usage-based | Existing setting | None |

All three plans are public and require a payment card. Team includes every
defined feature. Starter and Pay-as-you-go include only BYOS.
The seed enables local gating for features other than BYOS to enforce plan access.
Repeated runs refresh these settings and preserve existing prices, product links,
seat limits, and credit policies.
The seed does not assign plans to organizations or change subscriptions.
Production migrations do not run this seed.

## Credit policy and follow-up work

The initial Team plan has `credit_amount_cents = 0` and
`credit_frequency = none`. The editor hides included-credit controls. New
plans use this policy, and edits preserve stored credit policies. Generic
per-seat plans can store an included-credit policy.

The agreed follow-up rule grants each successfully paid seat's allowance to
the organization ledger, even before a member occupies that seat. Members
share that organization balance. Credit delivery depends on purchased-seat
tracking and is outside this change. Partial-period grants, expiration, and
adjustments require their own specification.

Changing the catalog maximum performs no subscription or ledger writes.
Purchased-capacity behavior belongs to [#8818](https://github.com/superplanehq/superplane/issues/8818).
Additional purchases, invitations, renewal estimates, and credit delivery remain separate work.

## Verify in the sandbox

1. Configure a sandbox product and SuperPlane as described above.
2. Create a public Team plan with a matching monthly seat price and maximum 50.
3. Confirm that the plan appears in Billing plans.
4. Change the maximum within the Polar tier's range and save.
5. Try a mismatched price on a new plan. Confirm that the editor shows an error and retains entered values.
6. Try an archived or missing product. Confirm that SuperPlane rejects the save.
7. Confirm that rejected updates leave the stored maximum unchanged.

## Verify the initial Team purchase in the sandbox

Use separate organizations for successful payment and abandonment. Do not use
production cards or production tokens. The $20 monthly seat price below is
an example. Use your catalog price for the actual estimates.

1. Configure the sandbox product and public Team plan as described above.
2. Enable **New Billing** for a new Trial organization with an owner and one active teammate.
3. Complete workspace onboarding without selecting a plan or entering payment details.
4. Confirm that the organization remains on Trial with the original trial expiration.
5. Open **Settings → Billing → Payment method** as a billing manager.
6. Confirm that Team starts at two seats and shows the effective plan maximum.
7. Select three seats. At $20 per seat, confirm a $60 initial estimate and $60 monthly total.
8. Confirm that the price says **per seat per month** and the note says **Estimates exclude tax.**
9. Select **Subscribe to Team**. Inspect the Polar request with a local HTTP proxy or debugger.
10. Confirm that `seats`, `min_seats`, and `max_seats` all equal 3 in the Polar request.
11. Confirm that hosted checkout fixes the quantity at three seats.
12. Complete payment with a Polar sandbox payment method.
13. Confirm delivery of `subscription.created` or `subscription.active` to the configured webhook.
14. Confirm that the stored organization plan is Team and `purchased_seats` equals 3.
15. Resolve seat usage. Confirm purchased 3, occupied 2, and available 1.
16. Confirm that stored period dates match the webhook's supplied monthly start and end dates.
17. Repeat the flow with a separate two-member Trial organization, then abandon checkout before payment.
18. Confirm that this organization remains on Trial and its purchased capacity and expiration stay unchanged.

Team checkout sends `allow_discount_codes: false`. Legacy Business checkout omits
this field.

Each organization has at most one pending subscription checkout. The table
`organization_billing_checkouts` stores its plan, seat count, and Polar status.
A repeated request returns the pending checkout when the plan and seats match.
It fails when they differ. Polar is the only source of an end state:

- Polar reports `expired` or `failed`: the request records the state and
  creates a new checkout.
- Polar reports `succeeded`: the request records the state. It reads the
  subscription from Polar before it creates another checkout.
- A subscription that becomes live for the plan product resolves the pending
  checkout.
- Polar reports `open` or `confirmed`: the checkout stays pending. A past local
  expiry and a closed browser do not cancel it. `confirmed` does not show that
  the payment succeeded.
- The create call times out or fails with a 5xx error: the checkout is
  `ambiguous`. The next request finds it through the
  `superplane_checkout_intent_id` metadata. When no checkout appears within 10
  minutes, the request releases the intent.

On the first request of an organization without intents, SuperPlane adopts a
payable Polar checkout that was opened before this table existed. The migration
creates no rows.

Checkout creation does not activate a plan or grant capacity. Only subscription
activation stores the purchased quantity. The monthly total describes purchased
capacity. Membership-based renewal estimates belong to
[#8830](https://github.com/superplanehq/superplane/issues/8830).

`occupied_seats` uses billing response field 28. Issue #8830 must reuse that
field and allocate additional fields after field 34. An absent `seat_maximum` means
unlimited. An organization override replaces the target catalog maximum,
including an unlimited override.

## Buy more seats in the sandbox

A billing manager buys more seats during the billing period. Polar charges the
prorated amount at once and does not change the renewal date. SuperPlane adds
capacity only from the Polar subscription snapshot that follows a successful
payment. The $20 monthly seat price below is an example. Use your catalog price.

The API has two routes. Both need `org:update` and **New Billing**:

- `POST /api/v1/organizations/{id}/billing/seats/preview` with `additional_seats`.
  It estimates the charge and makes no change.
- `POST /api/v1/organizations/{id}/billing/seats/purchase` with
  `additional_seats` and `expected_purchased_seats`. The second field must equal
  the current `purchased_seats`. A repeated or stale request gets HTTP 409.

SuperPlane sends `PATCH /subscriptions/{id}` with the absolute seat count and
`proration_behavior` set to `invoice`. It does not use any other value. A
purchase first reads the current subscription from Polar and applies the
snapshot. The absolute count builds on the seats Polar reports, so a purchase
does not remove seats that were added elsewhere.

1. Use an organization with an active Team subscription that is not set to cancel.
2. Read `purchased_seats`, `current_period_end`, and `purchasable_seats` from the billing response.
3. Call the preview route for one seat. Confirm that `prorated_charge_cents` equals
   `seat price x remaining seconds / period seconds`, rounded to the nearest cent.
4. Confirm that `estimate` and `excludes_tax` are `true`.
5. Call the purchase route for one seat. Inspect the Polar request with a local HTTP proxy.
6. Confirm that the body is `{"seats": <purchased + 1>, "proration_behavior": "invoice"}`.
7. In the Polar sandbox dashboard, confirm that Polar created an order for the prorated amount at once.
8. Confirm that `purchased_seats` grew by one, `last_seat_purchase.status` is `succeeded`,
   and `current_period_end` did not change.
9. Send the same purchase request again. Confirm HTTP 409 and no second Polar order.
10. Set a sandbox payment method that Polar declines. Buy one seat. Confirm that the
    request fails with the payment failed message and `purchased_seats` does not change.
11. Set a sandbox payment method that needs bank confirmation. Confirm the distinct
    bank confirmation message and that `purchased_seats` does not change.

If Polar does not answer, the purchase has the status `awaiting_confirmation`.
Capacity does not change. A `subscription.updated` webhook, a billing sync, the
renewal seat schedule, or the next preview or purchase reads the subscription
from Polar and settles the purchase.
Only one purchase can be pending or unconfirmed for an organization at a time.
A purchase that Polar never applied fails five minutes after it started, when the
next preview, purchase, or renewal seat schedule checks Polar. The same rule
applies when Polar reports fewer seats than the target after an outside change.

The table `organization_seat_purchases` is the marker for seat changes that
SuperPlane started. Use `models.FindExpectedSeatCount` to tell them from outside
changes in Polar.

## Renew only occupied seats

Polar renews a subscription with the seat count of the subscription.
SuperPlane schedules the next-period seat count so that Polar renews only
occupied seats. The target comes from `entitlements.ResolveRenewalSeats`:
the number of active human members, owner included, with a minimum of 1,
and never more than the paid seats (`purchased_seats`). Pending invitations,
removed members, and service accounts do not count. SuperPlane never
increases seats automatically.

The target depends only on the paid seats and the members. It does not use
the payment lock, the admission limits, or the catalog maximum. A payment
failure can move the plan row to a fallback catalog (None or Trial) that is
not per seat. The row keeps `purchased_seats`, so the target stays the same.
A seat-based plan without paid seats (`purchased_seats` missing or 0) is
invalid. Schedule returns an error for it and sends nothing. Repair the plan
from Polar.

SuperPlane sends `PATCH /subscriptions/{id}` with
`{"seats": <target>, "proration_behavior": "next_period"}`. Polar does not
change the current period. It charges and credits nothing now, and `seats`
keeps the current count. The subscription shows the scheduled count in
`pending_update.seats`. The next-period count is `pending_update.seats`, or
`seats` when no seat change is pending. SuperPlane never sends a target above
the current `seats`, and never uses `invoice`, `prorate`, or `reset` for this
request.

Example: an organization has 10 seats, and the period ends on December 1.
An admin removes a member on November 10. SuperPlane schedules 9 seats for
December 1. The freed seat stays usable until December 1. When a member joins
before December 1, SuperPlane schedules 10 seats again.

`seatrenewal.Schedule` in `pkg/billing/seatrenewal` does the work. It first
settles or expires an old seat purchase from Polar, so an abandoned purchase
cannot keep its organization unchecked forever. It then does nothing when one
of these conditions is true:

- The plan is not seat-based: the catalog plan is not per seat and the plan has no purchased seats.
- Billing is off.
- An admin assigned the plan.
- The plan has no Polar subscription.
- The subscription is `paused` (`subscription_paused`), or it is `unpaid`, `canceled`, `revoked`, or in a different state that does not renew (`subscription_not_active`).
- The subscription is set to cancel (`subscription_canceling`), also when it is past due.
- The verified Polar snapshot is not `active` or `past_due`, or it is set to cancel (`polar_subscription_not_active`).
- A seat purchase is still inside its confirmation window. The purchase
  replaces any scheduled count when it settles, and SuperPlane schedules the
  count again after that.

### Past-due subscriptions

Polar keeps a `past_due` subscription and continues to renew it while it
retries the payment. Schedule therefore keeps the next-period count of an
`active` or a `past_due` subscription. It sends only the `next_period`
update, so nothing changes in the current period and Polar charges nothing
now.

The scheduled count grants no access. Invitations, invitation acceptance,
and seat purchases stay blocked while the subscription is past due or
paused. The past-due grace period does not change.

When Polar rejects the update, Schedule returns the error and does not
store `renewal_seats_scheduled`. The background retry and the
reconciliation worker log the failure and try again on the next run. When
the payment recovers, the next run keeps the reduced count.

Schedule holds a per-organization PostgreSQL session advisory lock across the
local reads, Polar requests, and record write. API processes and workers use
the same lock. Each schedule reads its target after it acquires this lock, so
an older job cannot overwrite a newer scheduled target.

Each process admits at most two schedules before it reserves database
connections. API schedules and reconciliation workers share this limit.
Other schedules wait without a database connection, and cancellation ends
that wait. This leaves three connections in the default five-connection pool
available for foreground requests, even when Polar is slow.

Schedule reads local state under the organization row lock. It releases the
row lock before it calls Polar. Membership changes do not wait for HTTP.
After each update, it stores the target in
`organization_billing_plans.renewal_seats_scheduled` and the send time in
`renewal_seats_scheduled_at`. Subscription snapshots do not change these
columns. Outside-change detection can compare them with the Polar count.

These columns record only counts that SuperPlane sent. When Polar already
renews with the target but the record is missing or names a different count
(for example, because the record write failed after Polar accepted an
update), Schedule sends the target again and repairs the record. A matching
Polar count alone never repairs it. The send is idempotent: Polar keeps the
current period, and the scheduled count stays the target.

### Live updates

These actions schedule the count after their transaction commits:

- Remove a member (`RemoveUser`).
- Accept an invitation with the status `joined`. This includes a former member that the invitation restores.
- Buy seats (`PurchaseSeats`), after SuperPlane applies the Polar snapshot. An
  `invoice` update replaces the scheduled count, so SuperPlane schedules it again.
- Recover a successful seat purchase through a subscription webhook, billing
  sync, or purchase reconciliation. After the settlement commits, SuperPlane
  schedules the count again. A lost purchase response does not leave the
  reduction dependent on the next worker pass.

After a successful subscription apply or sync, SuperPlane checks the latest
purchase. A succeeded purchase starts a background schedule. Duplicate
snapshots can start another schedule, but an unchanged provider count and
stored record require no update.

The update runs in the background. The action does not wait for Polar and does
not fail because of Polar. The update tries 3 times in about 10 seconds when
Polar rate-limits the request, reports `SubscriptionLocked`, or does not answer.
The worker fixes other failures.

### Reconciliation worker

`SeatRenewalSyncWorker` runs when `START_SEAT_RENEWAL_SYNC_WORKER=yes`. Every 5
minutes it checks each organization on a seat-based Polar subscription that
is not canceled or revoked. Organizations with the earliest `current_period_end`
go first. The worker uses `seatrenewal.Schedule`, so the same rules apply. It
fixes counts after a Polar outage, a lost update, a race between a purchase and
a removal, and membership changes without a live update, such as account
deletion.

When Polar rate-limits a request, the run stops and keeps its position. The
next run continues at the organization where the rate limit stopped it, so a
request budget smaller than the list cannot keep later organizations
unchecked. A run that reaches the end of the list clears the position, and the
next run starts again with the organizations that renew first.

The worker writes these log entries. Each entry has `organization_id`,
`polar_subscription_id`, `renewal_target_seats`, `next_period_seats`, and
`current_seats` when they are known.

- `billing.seat_renewal_mismatch_fixed` (warning): the worker found a different next-period count and scheduled the target.
- `billing.seat_renewal_sync_failed` (error): a fix failed, and the period ends in 24 hours or less.

### Sandbox checks (run before merge)

A person must do these checks in the Polar sandbox before merge. Record each
observed result here. Until then, each result is pending. To watch a renewal
soon, move the period end a few minutes ahead with `PATCH /v1/subscriptions/{id}`
and `{"current_billing_period_end": "<time>"}`.

1. Update back to the current count.
   - Step: on a subscription with 10 seats, schedule 9 seats with `next_period`.
     Then send `{"seats": 10, "proration_behavior": "next_period"}`.
   - Expected result: Polar accepts the update, and `pending_update` is `null`.
     If Polar rejects the update or keeps `pending_update`, find the request that clears a pending seat change.
   - Observed result: pending.
2. Shape of `pending_update`.
   - Step: schedule a reduction. Read the PATCH response, `GET /v1/subscriptions/{id}`, and the `subscription.updated` webhook.
   - Expected result: each one has `pending_update` with `seats` (integer) and `applies_at` (date-time), as `SubscriptionPendingUpdate` parses it.
   - Observed result: pending.
3. Renewal after a scheduled reduction.
   - Step: schedule a reduction from 10 to 9 seats. Move the period end a few minutes ahead and wait for `subscription.cycled`.
   - Expected result: `seats` is 9 in `subscription.cycled`, the renewal invoice charges 9 seats, and Polar issues no credit.
   - Observed result: pending.
4. Scheduled reduction on a past-due subscription (gate G9).
   - Step: make the renewal payment of a subscription with 10 seats fail, so
     that the status is `past_due`. Send
     `{"seats": 9, "proration_behavior": "next_period"}`. Read the PATCH
     response and `GET /v1/subscriptions/{id}`. Then move the period end a
     few minutes ahead and wait for `subscription.cycled`. Move the period
     end: the payment retries can end before the next monthly boundary.
   - Expected result: Polar accepts the update, `pending_update.seats` is 9,
     and Polar creates no invoice and no charge now. The cycle renews with 9
     seats. If Polar rejects the update, record the status and the error
     code. SuperPlane reports the rejection as a failure and tries again, but
     the past-due eligibility rule then needs a follow-up change.
   - Observed result: pending.

## Over capacity

A per-seat organization is over capacity when it has more active members than
`purchased_seats`. Admission is already safe in this state. `JoinLimit()` returns
the purchased seats and `Invitable` is 0, so no one can join or get an
invitation. The Seats card on the billing page tells Admins to buy seats or
remove members. Other users see a request to ask an organization admin.

SuperPlane stores the start of the state in
`organization_billing_plans.over_capacity_since`. It never sends a request to
Polar and never buys seats.

### Causes

- **Checkout race.** Members join on the old plan while a Team checkout is open.
  The activation snapshot then has fewer seats than members. The log cause is
  `activation`.
- **Renewal after a late join.** A member joins before renewal and the renewal
  keeps the older seat count. The log cause is `cycle`.
- **Outside reduction.** Someone lowers the seats in the Polar dashboard. The log
  cause is `update`.
- **Recovery from past due.** The payment recovers while members exceed seats. The
  log cause is `recovery`.

### Log lines

Every line uses the component `billing.seats`. The fields are
`organization_id`, `subscription_id`, `purchased`, `occupied`, and `cause`.

- `billing.seats_over_capacity` is an error. SuperPlane writes it once when the
  state starts. Repeated webhooks while the state lasts write nothing.
- `billing.seats_over_capacity_resolved` is an info line. SuperPlane writes it
  when members and seats balance again. The cause is one of `update`, `cycle`,
  `recovery`, `removal`, or `account_deletion`. The next start alerts again.

A plan that is not an active per-seat plan clears the record without a log
line. Installs with billing off write no line.

With `LOG_FORMAT=json`, find the alerts in Cloud Logging with this query:

```text
jsonPayload.component="billing.seats" AND jsonPayload.message="billing.seats_over_capacity"
```

### Repair

1. Find the organization with the query above.
2. Ask a billing manager of the organization to buy seats in **Settings → Billing**.
   Or remove members in **Settings → Members**.
3. Confirm the `billing.seats_over_capacity_resolved` line.
4. Confirm that the Seats card no longer shows the notice.

Renewal never adds seats automatically. The renewal keeps the seats that Polar
holds. Until someone buys seats or removes members, the organization stays over
capacity and admission stays blocked.

## Resolve subscription identity conflicts

The log `billing.foreign_subscription_snapshot_ignored` means a snapshot arrived
for another subscription that is not live. SuperPlane keeps the live subscription's
billing state, customer, credits, and pending seat purchases. The log includes the
organization ID, both subscription IDs, and their statuses.

The Error log `billing.multiple_live_subscriptions` means Polar confirmed two live
subscriptions for one organization. SuperPlane keeps the stored subscription and
does not apply the incoming snapshot. This condition requires manual operator
resolution.

1. Find the organization and both subscription IDs in the log.
2. Open both subscriptions in the correct Polar environment.
3. Confirm which subscription the organization must retain.
4. Manually end the unintended subscription in Polar.
5. Sync organization billing and confirm its plan, purchased seats, credits, and customer.
6. Confirm that **Allow multiple subscriptions** remains disabled in sandbox and production.

SuperPlane does not cancel duplicate subscriptions or issue refunds automatically.
Handle any required refund separately. If the refresh fails, retry webhook delivery
or sync billing after Polar recovers. SuperPlane preserves local state until it
can verify the stored subscription.
