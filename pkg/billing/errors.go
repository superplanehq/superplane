package billing

import (
	"errors"
	"fmt"
)

var (
	//revive:disable-next-line:error-strings
	ErrOrganizationSubscriptionLive = errors.New("Cancel the subscription in Billing before you delete the organization. You can delete it after the paid period ends.")
	//revive:disable-next-line:error-strings
	ErrOrganizationSubscriptionUnverified = errors.New("SuperPlane could not confirm with Polar that the subscription ended. Try again later.")
	//revive:disable-next-line:error-strings
	ErrOrganizationCheckoutPending      = errors.New("Finish the pending checkout or wait until it expires before you delete the organization.")
	ErrBillingPlanChanged               = errors.New("the billing record changed. Refresh and try again")
	ErrHostedCreditEmpty                = errors.New("hosted credit is empty")
	ErrFactoryHostedSpendBudgetNegative = errors.New("hosted spend limit cannot be negative")
	ErrFactoryHostedBudgetEmpty         = errors.New("this workspace has no remaining hosted credit")
	ErrHostedSubscriptionRequired       = errors.New("an active plan is required to run agents")
	ErrHostedPaymentPastDue             = fmt.Errorf("the renewal payment is past due: %w", ErrHostedSubscriptionRequired)
	ErrHostedSubscriptionPaused         = fmt.Errorf("the subscription is paused: %w", ErrHostedSubscriptionRequired)
	ErrPaymentMethodRequired            = errors.New("add a payment method in organization billing before usage")
)
