# Billing foundation rollout

Use this procedure for `codex/billing-foundation`. Deploy the admin and
customer branches separately.

Foundation preserves the existing customer billing screens and checkout.
Business subscriptions use the `business_legacy` catalog entry, named Business.
Admin Business assignments use `business_manual`, named Business Manual.
The manual seed copies the existing Business limits, price, and credit policy.
It has no Polar product and does not require a payment method.
Its stored price does not create a charge or subscription.

The migration changes only the catalog reference for existing admin Business
assignments. It preserves subscription history, credit periods, and grants.
The foundation admin screen still labels its existing option **Business**.
That option now assigns Business Manual. The admin branch displays catalog names.

## Before release

1. List the organization IDs for the companies that need verification.
2. Record each organization's expected plan, source, credit balances, trial end,
   current period, and markup override from the admin panel.
3. Record the Polar subscriber's product, subscription ID, status, period, and
   cancellation setting. Confirm that `POLAR_BUSINESS_PRODUCT_ID` matches its product.
4. Keep the current billing and Polar configuration. Confirm that the foundation
   image contains the complete schema and data migration directories.
5. Rehearse the upgrade against a recent database copy in an isolated environment.
   Use the foundation image and the production configuration shape.
6. Compare every listed organization with its recorded state. Check a manual
   assignment, a credit repair, signup, checkout, and a hosted run.
7. Test an actual subscription webhook and a duplicate delivery in a Polar sandbox.
   Confirm that the duplicate does not add included credit again.
8. Prepare a database backup and a foundation-compatible recovery image.
   Verify the backup restore procedure before the production window.

Do not treat a successful migration as proof that each organization's billing
is correct. The organization comparison and subscription checks are release gates.

## Production window

1. Pause incoming work and drain active work before stopping application writers.
2. Stop every old application and worker instance, including scheduled jobs.
   Suspend controllers that could restart them during the window.
3. Take the final backup and record the billing state after writers stop.
4. Apply schema migrations, then data migrations, using the foundation image.
5. Start all application instances on foundation. Confirm that migration versions
   match the image and that both migration tables report `dirty = false`.
6. Check the catalog assignments with the query below. Investigate every returned row.
7. Verify the recorded organizations in the admin panel and their billing screens.
   Compare access, credit buckets, trial dates, periods, and markup overrides.
8. Verify the Polar subscriber's sync, included credit, invoices, and billing portal.
   Confirm that manual companies remain independent of Polar subscription status.
9. Run one internal hosted task and create one internal test organization.
   Check trial credit, checkout, credit history, and cancellation controls.
10. Resume incoming work. Retry or redeliver webhooks held during the window.
    Watch billing errors and check the subscriber again after its next renewal.

The Helm migration job runs before replacement application pods start.
A normal rolling upgrade, or changing only the rollout strategy, does not stop
old writers before that job. Stop them explicitly before the upgrade.

Run this read-only query after migration and before admin catalog assignments:

```sql
SELECT o.id, o.name, b.plan, b.plan_source, p.key AS catalog_key
FROM organizations o
JOIN organization_billing_plans b ON b.organization_id = o.id
LEFT JOIN billing_plans p ON p.id = b.plan_id
WHERE p.id IS NULL
   OR p.key <> CASE
       WHEN b.plan = 'business' AND b.plan_source = 'admin' THEN 'business_manual'
       WHEN b.plan = 'business' THEN 'business_legacy'
       ELSE b.plan
   END
ORDER BY o.name;
```

An organization without a billing row is resolved by the application on access.
Verify all listed organizations in the application, including those absent from
this query's billing join.

## Repair one organization

- **Manual access is missing:** open the organization's Hosted credit section.
  Select Business and save. This assigns Business Manual with paid access.
  Check the balances and run one task for that organization.
- **Credit is too low:** record the missing amount and add it to the current
  SuperPlane grant balance. Enter the resulting total, then select
  **Set SuperPlane grant**. For example, change a $5 grant to $25 to restore $20.
  Add a reason in the note field and verify the credit history.
- **Trial dates are wrong:** select Trial and set the correct future end date.
  Verify the trial balance separately. Use Business for a company that should
  retain manually granted paid access.
- **The Polar subscriber is wrong:** verify the real subscription in Polar and
  the product configuration first. Reload billing to sync it, or redeliver its
  webhook. Do not cancel a valid subscription to unlock manual plan controls.
- **Requests fail across companies:** inspect application and migration errors.
  An admin plan edit cannot repair a missing catalog row or an old application
  writing without `plan_id`. Keep traffic paused and deploy a foundation fix.

Foundation can edit trial, top-up, and SuperPlane grant balances. Editing the
included-credit bucket arrives in the admin branch. Use a SuperPlane grant to
restore missing spendable credit in the meantime. An excess included balance
needs a database repair or the admin branch; do not toggle plans to alter it.

## Cancel the subscription of a deleted organization

SuperPlane blocks the deletion of an organization while its Polar subscription
is live. A subscription is live when its status is `active`, `trialing`,
`past_due`, or `paused`, or when it is set to cancel and the paid period is
still open. Admin-assigned plans and organizations without a Polar subscription
are not blocked.

Account deletion uses the same protection for organizations with no other owner.
The subscription check and deletion share the organization lock and transaction
with subscription updates. Polar HTTP requests run outside these row locks.

Polar can still have a subscription for an organization deleted by an older
release, or a subscription that starts in Polar after deletion commits. Polar then
keeps charging. SuperPlane does not apply the webhooks for that organization.
It writes one error log for each webhook and answers Polar with 202.

1. Search the application logs for `billing.subscription_for_deleted_organization`.
   The entry has the `organization_id`, `polar_subscription_id`, and
   `polar_subscription_status` fields.
2. In Polar, open **Customers** and find the customer with the same external ID
   as the `organization_id`. Open its subscription. Confirm that the ID matches
   `polar_subscription_id`.
3. Cancel or revoke the subscription in Polar. Cancel it to end it at the period
   end. Revoke it to end it now. Refunds are a separate decision. Do not refund
   from this procedure.
4. Wait for the `subscription.canceled` or `subscription.revoked` webhook. The log
   entry appears once more with the new status. Confirm that the status is
   `canceled` or `revoked`.
5. Confirm that Polar shows no later charge for the customer.

A webhook for a deleted organization only writes a log. It does not change data.

To delete a live organization, the Admin cancels the subscription in
**Billing** first. The Admin can delete the organization after the paid period
ends.

## Recovery limits

Use a foundation-compatible image for application recovery after migration.
Do not automatically roll back to the pre-catalog image after a failed upgrade.
Old writes omit the required `plan_id`, and old Helm startup expects older
migration versions. Admin edits cannot remove those incompatibilities.

The empty down migrations do not restore old schema or data. Do not run them
as a recovery procedure. Restoring the old application requires restoring a
matching database backup while all writers remain stopped. A backup restore
discards writes made after that backup; reconcile them before resuming traffic.
