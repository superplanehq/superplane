# GA4 conversion tracking

## Scope

The app loads the website GTM container, `GTM-TKMMDB5T`, on `app.superplane.com`.
The loader runs once at app startup and remains available across all SPA routes.
Local, preview, and self-hosted installations do not load GTM or push these events.
PostHog remains independent. The app does not install a second Google Analytics loader.
There is no noscript iframe, consistent with the website integration.

## Events

- `sign_up`: After the authenticated account loads following confirmed registration.
  Password and magic-code flows use the existing success confirmation.
  OAuth uses the backend `auth_signup_result=created` redirect signal.
  A signup attempt, existing-account login, or visit to `/welcome` alone does not qualify.
- `purchase`: After a checkout return matches a paid Polar order retrieved by the backend.
  Both Business subscriptions and hosted credit purchases include `checkout_id` in their return URL.
  The backend queries orders for the authenticated organization.
  The return URL, plan status, and checkout button cannot independently confirm payment.

Purchase events use the standard `ecommerce` object. It contains `transaction_id`,
`currency`, `value`, `tax`, and `items`. The value excludes tax and reflects discounts.
Amounts from Polar are converted from cents to currency units. The order ID is the transaction ID.
The app clears the previous ecommerce object before each purchase event.
Account names and email addresses are not included in these conversion payloads.
Impersonation sessions do not produce conversion events.

Local storage and an in-memory set suppress repeated events for the same account or order.
GA4 also receives the stable transaction ID for purchase deduplication.
If storage is unavailable, in-memory deduplication lasts until the page reloads.
Clearing storage or using another browser can remove client-side deduplication.

The billing page retries order retrieval up to 30 times, with two seconds between attempts.
A later page reload can retry an unresolved payment. A user who never returns from checkout
cannot produce a browser data-layer event. Subscription renewals do not trigger this checkout event.
Ad blockers and tag consent settings can prevent delivery to GA4.

## GTM setup after deployment

1. Configure a Google tag for the app hostname in the shared container.
2. Use the existing measurement ID, `G-3FR4MP9N8W`, if the app shares the website property.
3. Preserve the website settings that disable Google Signals and advertising personalization.
4. Restrict this base tag to `app.superplane.com` to avoid duplicate website initialization.
5. Add custom-event triggers named `sign_up` and `purchase`.
6. Connect GA4 event tags to these triggers. Enable ecommerce data from the data layer for `purchase`.
7. Configure SPA page views once, through the Google tag or GTM history triggers.
8. Review consent behavior and publish the container through the existing approval process.

The agency owns the GTM triggers and GA4 configuration. Repository changes do not grant GTM access
or publish container changes. No GTM permissions change is required by this code.

## Verification

1. Run `make dev.up` and `make dev.setup` in a Docker environment.
2. Run `make pb.gen` after changing the protobuf fields. Do not commit generated files.
3. Run `make check.proto.field.numbers`.
4. Run `make check.test.ui FILES="src/lib/googleTagManager.spec.ts src/lib/signupAnalytics.spec.ts src/contexts/AccountProvider.gtm.spec.tsx src/hooks/useGooglePurchaseTracking.spec.tsx"`.
5. Run `make test PKG_TEST_PACKAGES="./pkg/billing/polar ./pkg/grpc/actions/organizations"`.
6. Run the required frontend and backend formatting, lint, and build checks.
7. On production, use GTM Preview to confirm one container request and the conversion payloads.
8. Register through each supported method. Confirm one `sign_up` after success.
9. Confirm failed signup and existing-account login produce no `sign_up`.
10. Complete an approved test checkout. Confirm one `purchase` with the paid order ID and correct amounts.
11. Refresh the return page. Confirm no duplicate conversion.
12. Confirm canceled checkout and an unmatched checkout ID produce no `purchase`.
13. Confirm the corresponding events arrive in GA4 DebugView.

References: [Google data layer](https://developers.google.com/tag-platform/tag-manager/datalayer)
and [Polar orders](https://polar.sh/docs/api-reference/orders/list).
