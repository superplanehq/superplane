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

Public visibility makes the plan available in the customer catalog. This
catalog change does not add Team checkout or purchased-seat tracking.
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
Checkout, purchases, invitations, and credit delivery remain separate work.

## Verify in the sandbox

1. Configure a sandbox product and SuperPlane as described above.
2. Create a public Team plan with a matching monthly seat price and maximum 50.
3. Confirm that the plan appears in Billing plans.
4. Change the maximum within the Polar tier's range and save.
5. Try a mismatched price on a new plan. Confirm that the editor shows an error and retains entered values.
6. Try an archived or missing product. Confirm that SuperPlane rejects the save.
7. Confirm that rejected updates leave the stored maximum unchanged.
