# Buy seats release gate

Gate G10 requires one manual purchase in the Polar sandbox before release.
Automated tests use simulated Polar responses and do not satisfy this gate.

Status: pending. No manual sandbox purchase is recorded for this change.

1. Use an organization with an active per-seat sandbox subscription.
2. Sign in as an organization admin and open Billing.
3. Select **Buy seats**. Confirm the additional seat count, prorated charge, renewal date, and new monthly total.
4. Select **Confirm purchase** once.
5. If confirmation is pending, close the dialog. Confirm that billing continues to check the payment.
6. Confirm that **Seats added.** appears and purchased capacity increases by the selected count.
7. Verify the charge and updated subscription quantity in the Polar sandbox.

Record the test date, tester, sandbox organization, purchase ID, Polar order ID,
before and after quantities, charge, and final status in the release evidence.
Do not record payment details or access tokens.
