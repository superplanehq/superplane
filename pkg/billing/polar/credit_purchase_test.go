package polar

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateCreditPurchase(t *testing.T) {
	for _, test := range []struct {
		name     string
		credits  int64
		feeBPS   int
		fixedFee int64
		subtotal int64
		invalid  bool
	}{
		{name: "fees disabled", credits: 10000, subtotal: 10000},
		{name: "rounds Starter fee up", credits: 10000, feeBPS: 500, fixedFee: 50, subtotal: 10579},
		{name: "fifty dollar purchase", credits: 5000, feeBPS: 500, fixedFee: 50, subtotal: 5316},
		{name: "fractional percentage", credits: 5000, feeBPS: 525, fixedFee: 50, subtotal: 5330},
		{name: "rounds Early Member fee up", credits: 10000, feeBPS: 400, fixedFee: 40, subtotal: 10459},
		{name: "exact division", credits: 95, feeBPS: 500, subtotal: 100},
		{name: "minimum charge", credits: 1, fixedFee: 49, subtotal: 50},
		{name: "maximum charge", credits: 99999999, subtotal: 99999999},
		{name: "zero credits", credits: 0, invalid: true},
		{name: "negative credits", credits: -1, invalid: true},
		{name: "negative percent", credits: 10000, feeBPS: -1, invalid: true},
		{name: "one hundred percent", credits: 10000, feeBPS: 10000, invalid: true},
		{name: "negative fixed fee", credits: 10000, fixedFee: -1, invalid: true},
		{name: "below minimum charge", credits: 49, invalid: true},
		{name: "fee exceeds maximum charge", credits: 99999999, feeBPS: 1, invalid: true},
		{name: "credits overflow", credits: math.MaxInt64, invalid: true},
		{name: "fixed fee overflow", credits: 10000, fixedFee: math.MaxInt64, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			quote, err := CalculateCreditPurchase(test.credits, test.feeBPS, test.fixedFee)
			if test.invalid {
				require.Error(t, err)
				assert.Nil(t, quote)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.credits, quote.CreditAmountCents)
			assert.Equal(t, test.subtotal, quote.SubtotalCents)
			assert.Equal(t, test.subtotal-test.credits, quote.ProcessingFeeCents)
			assert.Equal(t, test.fixedFee, quote.FixedFeeCents)
			assert.Equal(t, test.feeBPS, quote.PercentageFeeBPS)
			assert.Equal(t, test.subtotal-test.credits-test.fixedFee, quote.PercentageFeeCents)
			assert.Equal(t, "usd", quote.Currency)
			assert.GreaterOrEqual(t, test.subtotal*int64(10000-test.feeBPS), (test.credits+test.fixedFee)*10000)
		})
	}
}
