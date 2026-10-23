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
`proration_behavior` set to `invoice`. It does not use any other value.

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
Capacity does not change. A `subscription.updated` webhook, a billing sync, or the
next preview or purchase reads the subscription from Polar and settles the purchase.
Only one purchase can be pending or unconfirmed for an organization at a time.
A purchase that Polar never applied fails five minutes after it started, when the
next preview or purchase checks Polar.

The table `organization_seat_purchases` is the marker for seat changes that
SuperPlane started. Use `models.FindExpectedSeatCount` to tell them from outside
changes in Polar.
