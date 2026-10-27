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
| Starter | $20 | 10 | `abc9ef98-20c0-48b5-8204-1448df733737` |
| Pay-as-you-go | Usage-based | 10 | None |

All three plans are public and require a payment card. Team includes every
defined feature. Starter and Pay-as-you-go include only BYOS. The seed
creates Pay-as-you-go with no monthly subscription. Both starter plans allow
10 members. Pay-as-you-go allows five tasks at once; Starter has no task limit.
The seed enables local gating for features other than BYOS to enforce plan access.
Repeated runs refresh these settings and preserve existing prices and product links.
The seed does not assign plans to organizations or change subscriptions.
Production migrations do not run this seed.

## Onboarding plan selection

Workspace onboarding shows Pay-as-you-go and Starter after ticket-source
setup. Pay-as-you-go is selected initially. The choices come from the public
billing catalog. Team is not an onboarding choice. The step is skipped when
billing is off, a paid plan already exists, or starter plans are unavailable.

Pay-as-you-go collects a card in an embedded Polar form, then activates
without a subscription charge. Starter opens subscription checkout directly.
After checkout, onboarding synchronizes billing with Polar before continuing.
Organizations with an existing paid plan skip this step.
There is no admin setting for a default onboarding plan.

In Polar **Settings → Preferences → Embedding**, allow each host that serves
onboarding. For local testing, add `localhost:8000` and your HTTPS tunnel
host to the sandbox organization. Use host names without a URL scheme.
Configure production hosts in the live organization separately. Without this
setting, Polar blocks the card form with its `frame-ancestors` policy.

Every new account receives $10 in signup credit, valid for 365 days.
The grant does not depend on the plan. The grant is limited to one per account.
Selecting a plan does not add a second signup grant. Older accounts whose
onboarding deferred credit receive the grant during plan selection.
Credit alone does not permit runs. An existing trial company keeps its stored
credit balance and trial dates.

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
2. Create a new company with an owner and one active teammate. The company starts on None.
3. Complete workspace onboarding and select Pay-as-you-go. Confirm that card setup activates that plan.
4. Confirm that the organization has one signup credit grant.
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
17. Repeat the flow with a separate two-member company, then abandon checkout before payment.
18. Confirm that this company remains on None and its purchased capacity stays unchanged.

Team checkout accepts Polar discount codes. The request omits
`allow_discount_codes`, so the Polar default applies. Legacy Business checkout
also omits this field. In-app amounts, previews, and estimates show catalog
prices, not discounted prices. Manage the eligible products, redemption limits,
and amounts of a code in Polar.

Each organization has at most one pending subscription checkout. The table
`organization_billing_checkouts` stores its plan, seat count, and Polar status.
A repeated request returns the pending checkout when the plan and seats match.
It fails when they differ. Polar is the only source of an end state:

- Polar reports `expired` or `failed`: the request records the state and
  creates a new checkout.
- Polar reports `succeeded`: the request records the state. It reads the
  subscription from Polar before it creates another checkout.
- A subscription that becomes live resolves the pending checkout only when its
  `checkout_id` names the Polar checkout of the intent. The product or the
  seat count alone never resolves an intent. An intent that no subscription
  names stays pending until the request reads the checkout from Polar.
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

While a checkout can still activate, membership cannot exceed its seat count.
The claim checks that the seats cover the current members, and invitation
acceptance checks that one more member still fits. Both run under the
organization lock. An `ambiguous` or `creating` intent counts as payable.
Before the lock, acceptance reads the pending checkout from Polar, so a
verified `expired` or `failed` status releases the restriction. A rejected
acceptance returns `CHECKOUT_SEAT_LIMIT`, keeps the invitation pending, and
charges nothing. When Polar cannot answer, the stored restriction stays. For
an organization with a Polar customer and no intent rows, acceptance first
looks for a payable checkout from before intents, and it returns
`BILLING_UNVERIFIED` when Polar cannot answer.

Organization deletion and sole-owner account deletion read the pending
checkout from Polar first. A checkout that can still activate stops the
deletion with `ORGANIZATION_CHECKOUT_PENDING`. A paid checkout applies its
subscription before the subscription check. The deletion transaction checks
again under the organization lock that no checkout intent is pending.

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

The API has two routes. Both need `org:update`:

- `POST /api/v1/organizations/{id}/billing/seats/preview` with `additional_seats`.
  It estimates the charge and makes no change.
- `POST /api/v1/organizations/{id}/billing/seats/purchase` with
  `additional_seats`, `expected_purchased_seats`, and `operation_key`. The
  second field must equal the current `purchased_seats`. A stale request gets
  HTTP 409.

The `operation_key` identifies one purchase in the organization. It is
required and has 1 to 128 printable ASCII characters without spaces. A request
without a valid key gets HTTP 400 and does not reach Polar.

The buy seats dialog makes the key when the user confirms a purchase, not when
it shows a preview. The browser keeps the key in `localStorage` for the
organization, with the request and its billing period. All tabs of the
organization use the same key.

If a storage write fails, the tab keeps unresolved keys in memory. It merges
them with current storage records from other tabs. It excludes keys that
the tab retired, even when storage still contains those keys.

- A new confirmation of the same `additional_seats` and
  `expected_purchased_seats` sends the stored key again. This applies after a
  lost response, an HTTP 5xx error, a page refresh, and a renewal.
- A confirmation of a different request gets a new key. Earlier unresolved
  requests keep their own keys, even if the quantity or seat baseline changes.
- The browser forgets the key when the server answers for that key with a
  known outcome: a `succeeded`, `failed`, or `action_required` purchase, or an
  HTTP 4xx error.
- A lost response, an HTTP 5xx error, or a purchase with the status `pending`,
  `awaiting_confirmation`, or `unknown` keeps the key.
  A billing refresh does not forget the key, because the billing purchase has
  no key.
- The dialog never sends the stored request again by itself. A changed period
  or a stale preview needs a new preview and a new confirmation.

- A repeat with the same key and the same request returns the stored purchase.
  It does not call Polar, also after a renewal.
- A repeat with the same key and a different request gets HTTP 412. It does
  not call Polar.
- The purchase row stores the key, a fingerprint of `additional_seats` and
  `expected_purchased_seats`, the subscription, and the billing period.

Release the server and the web client together. A server with this rule
rejects a purchase from an older cached client until the page reloads. No
purchase path charges without a key.

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
9. Send the same purchase request again with the same `operation_key`. Confirm
   that the response has the same purchase ID and that Polar has no second order.
   Send it with a new key. Confirm HTTP 409 and no second Polar order.
10. Set a sandbox payment method that Polar declines. Buy one seat. Confirm that the
    request fails with the payment failed message and `purchased_seats` does not change.
11. Set a sandbox payment method that needs bank confirmation. Confirm the distinct
    bank confirmation message and that `purchased_seats` does not change.

If Polar does not answer, the purchase has the status `awaiting_confirmation`.
Capacity does not change. A `subscription.updated` webhook, a billing sync, the
reconciliation worker, or the next preview or purchase reads the subscription
from Polar. Only one purchase can be pending or unconfirmed for an
organization at a time.

SuperPlane records an outcome only from evidence about that purchase:

- The response to the purchase request records success, or the rejection
  that Polar returned. A success response must name the purchase
  subscription, the purchase period, and the target seats. Other success
  responses leave the purchase unconfirmed.
- A subscription snapshot records success only when all of these are true:
  it is for the purchase subscription, it is from the billing period in which
  the purchase started, it has exactly the target seats, and Polar modified
  the subscription after SuperPlane recorded the purchase.
- No snapshot records a failure. A snapshot with fewer seats can come from
  another update, for example a legacy renewal seat schedule that Polar
  processed before the purchase and that arrived late. The time since the purchase also
  proves nothing.

A snapshot that proves nothing still updates the capacity. The purchase keeps
its status. A duplicate, late, or reordered event never changes a recorded
outcome.

After a renewal, the seat count can return to the original count because of a
legacy scheduled reduction, so it proves nothing. A purchase that Polar never applied,
or that crosses a renewal without evidence, stays `awaiting_confirmation` and
blocks new purchases for the organization. SuperPlane does not retry the
charge and does not settle the purchase because time passed. The worker
reports the purchase after one hour. An operator repairs it from Polar
evidence. See
[Repair unresolved seat purchases](#repair-unresolved-seat-purchases).
A purchase from before operation keys has no stored period. Only a snapshot
whose period started before the purchase can settle it.

An unresolved purchase does not block renewal or reconciliation. SuperPlane
sends no automatic seat change that the purchase could race. The purchase
keeps its ID, its operation key, and its unresolved status, and it continues
to block new purchases.

The table `organization_seat_purchases` is the marker for seat changes that
SuperPlane started. Use `models.FindExpectedSeatCount` with the current period
start to tell them from outside changes in Polar.

## Renewal keeps paid seats

Polar renews a subscription with the seat count of the subscription.
SuperPlane does not change that count automatically. A member removal, an
invitation acceptance, an account deletion, a purchase preview, a billing
sync, a resume, and a subscription webhook send no seat change to Polar. Only
an explicit seat purchase changes the count, and it uses the `invoice`
proration. Paid seats that no member occupies stay paid and reusable until an
Admin changes them.

Example: an organization has 10 seats, and the period ends on December 1.
An admin removes a member on November 10. The freed seat stays usable, and
Polar renews 10 seats on December 1.

The retired automatic scheduler sent next-period reductions before this
change. Polar can still hold such a reduction in `pending_update`. SuperPlane
leaves it unchanged. The columns
`organization_billing_plans.renewal_seats_scheduled` and
`renewal_seats_scheduled_at` keep the last count that the retired scheduler
sent. Nothing writes them now. Legacy cleanup uses them to tell verified
automatic reductions from schedules that an Admin made.

Member removal and account deletion never call Polar, so they work while
Polar is unavailable.

### Reconciliation worker

`SeatRenewalSyncWorker` runs when `START_SEAT_RENEWAL_SYNC_WORKER=yes`. Every 5
minutes it checks each organization on a seat-based Polar subscription that
is not canceled or revoked. Organizations with the earliest `current_period_end`
go first. `seatrenewal.Reconcile` in `pkg/billing/seatrenewal` does the work.
It only reads Polar:

1. It settles an old seat purchase when a Polar snapshot proves its success.
   When Polar cannot return the subscription of a purchase on a different
   subscription than the stored plan, for example a replaced subscription, it
   writes `billing.seat_purchase_reconcile_failed` (warning) and continues
   with the current subscription.
2. It reads the stored Polar subscription and applies it. This repairs a
   missed renewal, payment failure, or recovery. Polar is the source of truth
   for the paid seats, the status, and the period. The apply ignores a
   snapshot that is older than the stored plan.

An admin plan and a plan without a Polar subscription need no Polar request.

When Polar rate-limits a request, the run stops and keeps its position. The
next run continues at the organization where the rate limit stopped it, so a
request budget smaller than the list cannot keep later organizations
unchecked. A run that reaches the end of the list clears the position, and the
next run starts again with the organizations that renew first.

The worker writes this log entry. It has `organization_id`,
`polar_subscription_id`, and `current_seats` when they are known.

- `billing.seat_renewal_sync_failed` (error): a reconciliation failed, and the
  period ends in 24 hours or less. Earlier failures are warnings, and the next
  run tries again.

At the start of each run, before it reads the candidate list, the worker also
reports each purchase that stayed `pending` or `unknown` for more than one
hour. The report covers all organizations that are not deleted, also
organizations that the candidate list excludes. A failure in the report does
not stop the reconciliation, and a failure in the reconciliation does not
stop the report. The report changes no purchase.

- `billing.seat_purchase_unresolved` (error): one entry for each unresolved
  purchase on each run. See
  [Repair unresolved seat purchases](#repair-unresolved-seat-purchases).

### Capacity alerts and transactions

The over-capacity lines describe committed state only. A subscription apply
that runs in its own transaction writes its lines after the commit. A caller
that applies a snapshot inside its own transaction, for example the seat
purchase settlement, gets the staged transitions and writes them after its
commit. A rollback, a failed commit, or the end of a nested transaction
writes nothing. A replay of an unchanged state writes nothing.

### Sandbox checks for manual removal and legacy cleanup

SuperPlane does not send these requests now. Manual seat removal and legacy
cleanup will send them. A person must do these checks in the Polar sandbox
before that work ships. Record each observed result here. Until then, each
result is pending. To watch a renewal
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
   - Expected result: each one has `pending_update` with `seats` (integer) and `applies_at` (date-time).
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

- **Activation below the members.** A checkout cannot admit more members than
  its seats while it can still activate, but a checkout from before these
  rules can. The activation snapshot then has fewer seats than members. The
  log cause is `activation`.
- **Renewal with a legacy reduction.** A reduction that the retired automatic
  scheduler left in Polar applies at renewal. The log cause is `cycle`.
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

## Repair unresolved seat purchases

A seat purchase is unresolved when its row in `organization_seat_purchases`
has the status `pending` or `unknown`. The API shows both as
`awaiting_confirmation`. A purchase with a lost response stays unresolved
when no evidence proves its success: for example, Polar did not apply it, or
it crossed a renewal. No subscription snapshot can record a failure.
SuperPlane never settles a purchase because time passed and never repeats a
charge. An operator settles it from Polar evidence.

An unresolved purchase blocks new purchases for the organization until it is
repaired. It does not block renewal or reconciliation.

### Detection and alerting

- **Detection threshold:** 1 hour after the purchase `created_at`.
- **Signal:** `SeatRenewalSyncWorker` writes the Error log
  `billing.seat_purchase_unresolved` on every run (every 5 minutes) while the
  purchase stays unresolved. The worker keeps no state, so the signal
  continues after a restart.
- **Alert owner:** the SuperPlane billing on-call engineer.
- **Routing:** a log-based alert policy with severity Error sends the alert
  to the billing on-call notification channel. Task 16 binds the concrete
  channel to the policy and runs the drill before gate G13 sign-off.
- **Duplicate suppression:** the alert policy opens one incident for each
  `seat_purchase_id`. It uses a notification rate limit, and it closes the
  incident when the line stops.

With `LOG_FORMAT=json`, find the signal in Cloud Logging with this query:

```text
jsonPayload.message="billing.seat_purchase_unresolved"
```

Do not filter only on `jsonPayload.component="billing.seats"`. This line does
not use that component.

Each line has these fields:

| Field | Meaning |
| --- | --- |
| `organization_id` | The organization of the purchase. |
| `seat_purchase_id` | The ID of the purchase row. |
| `operation_key` | The operation key of the purchase. Purchases from before operation keys do not have it. |
| `seat_purchase_status` | `pending` or `unknown`. |
| `polar_subscription_id` | The Polar subscription of the purchase. |
| `current_polar_subscription_id` | The subscription of the stored plan. |
| `current_period_start` | The period start of the stored plan. |
| `purchase_period_start`, `purchase_period_end` | The billing period of the purchase. Purchases from before operation keys do not have them. |
| `age_seconds` | The time since the purchase started. |

### Collect the evidence

Read the purchase row:

```sql
SELECT id, organization_id, polar_subscription_id, from_seats, to_seats,
       status, error_code, operation_key, period_start, period_end,
       created_at, updated_at
FROM organization_seat_purchases
WHERE id = '<seat_purchase_id>';
```

In the correct Polar environment, collect this evidence for the purchase
subscription (`polar_subscription_id`), not for the current subscription:

1. The orders and invoices of the subscription that Polar created from the
   purchase `created_at` through the confirmation window, five minutes later.
2. An order for `to_seats - from_seats` seats with a proration charge, and its
   payment status.
3. The seats of the subscription in the purchase period, from
   `purchase_period_start` to `purchase_period_end`.

Use the evidence to select one outcome:

| Evidence | Outcome |
| --- | --- |
| A paid order for `to_seats - from_seats` seats with a proration charge, and the subscription had `to_seats` seats or more in the purchase period. | `succeeded` with an empty error code. |
| No order or invoice in the window, and the subscription never had `to_seats` seats in the purchase period. | `failed` with the error code `not_applied`. |
| Any other result, for example an unpaid or refunded order, a missing period, or seat counts that do not agree with the orders. | Insufficient evidence. Do not repair. |

### Repair the purchase

Do these steps only when the evidence selects `succeeded` or `failed`.

1. Open an incident ticket. Attach the evidence before you change the row.
2. Run one guarded update for the outcome. The status condition makes sure
   that the update does not overwrite an outcome that SuperPlane recorded
   after you read the row.

   For a purchase that succeeded:

   ```sql
   UPDATE organization_seat_purchases
   SET status = 'succeeded', error_code = '', updated_at = now()
   WHERE id = '<seat_purchase_id>' AND status IN ('pending', 'unknown');
   ```

   For a purchase that Polar did not apply:

   ```sql
   UPDATE organization_seat_purchases
   SET status = 'failed', error_code = 'not_applied', updated_at = now()
   WHERE id = '<seat_purchase_id>' AND status IN ('pending', 'unknown');
   ```

3. Confirm that the update changed exactly one row. If it changed no row, read
   the row again. SuperPlane settled it, so do not run another update.
4. Do not send a request to Polar. The repair does not charge, refund, or
   change seats. Seat capacity changes only from a Polar snapshot.

### Record the audit trail

Record these items in the incident ticket:

- The `seat_purchase_id` and the organization ID.
- The Polar order ID or invoice ID that is the evidence, or a statement that
  no order exists in the window.
- The operator and the time of the update.
- The status and error code before and after the update.
- Links to the Polar orders, invoices, and subscription.

### Verify the repair

1. Sync organization billing. Confirm the plan, the purchased seats, and the
   period from Polar.
2. Confirm that `billing.seat_purchase_unresolved` stops for the
   `seat_purchase_id` after the next worker run.
3. Confirm that a new purchase is allowed: the seat purchase preview for the
   organization succeeds.

### Insufficient evidence

When the evidence does not prove one outcome, leave the status unresolved.
Escalate the incident to the billing on-call owner with the evidence that you
collected. Do not set a status to clear the block. A repair never repeats a
charge, and SuperPlane never retries a charge for an unresolved purchase.

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
