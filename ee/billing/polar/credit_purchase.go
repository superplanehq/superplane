package polar

import (
	"errors"
	"strconv"
	"strings"
)

const (
	creditPurchaseVersionKey = "superplane_credit_purchase_version"
	creditPurchaseAmountKey  = "superplane_credit_amount_cents"
	creditPurchaseTotalKey   = "superplane_credit_subtotal_cents"
	creditPurchaseVersion    = "1"
	minimumUSDPriceCents     = 50
	maximumUSDPriceCents     = 99999999
	feeBaseBPS               = 10000
)

type CreditPurchaseQuote struct {
	CreditAmountCents  int64
	ProcessingFeeCents int64
	SubtotalCents      int64
	Currency           string
	FixedFeeCents      int64
	PercentageFeeBPS   int
	PercentageFeeCents int64
}

func CalculateCreditPurchase(creditCents int64, feeBPS int, fixedFeeCents int64) (*CreditPurchaseQuote, error) {
	if creditCents <= 0 {
		return nil, errors.New("credit amount must be greater than zero")
	}
	if feeBPS < 0 || feeBPS >= feeBaseBPS {
		return nil, errors.New("processing fee percent must be at least 0 and less than 100")
	}
	if fixedFeeCents < 0 {
		return nil, errors.New("fixed processing fee cannot be negative")
	}
	if creditCents > maximumUSDPriceCents || fixedFeeCents > maximumUSDPriceCents-creditCents {
		return nil, errors.New("credit purchase subtotal cannot exceed $999,999.99")
	}

	// Bound cents before multiplication and round up so the fee covers the base credit.
	numerator := (creditCents + fixedFeeCents) * feeBaseBPS
	denominator := int64(feeBaseBPS - feeBPS)
	subtotal := (numerator + denominator - 1) / denominator
	if subtotal < minimumUSDPriceCents || subtotal > maximumUSDPriceCents {
		return nil, errors.New("credit purchase subtotal must be between $0.50 and $999,999.99")
	}
	return &CreditPurchaseQuote{
		CreditAmountCents:  creditCents,
		ProcessingFeeCents: subtotal - creditCents,
		SubtotalCents:      subtotal,
		Currency:           "usd",
		FixedFeeCents:      fixedFeeCents,
		PercentageFeeBPS:   feeBPS,
		PercentageFeeCents: subtotal - creditCents - fixedFeeCents,
	}, nil
}

func (q *CreditPurchaseQuote) metadata() map[string]string {
	return map[string]string{
		creditPurchaseVersionKey: creditPurchaseVersion,
		creditPurchaseAmountKey:  strconv.FormatInt(q.CreditAmountCents, 10),
		creditPurchaseTotalKey:   strconv.FormatInt(q.SubtotalCents, 10),
	}
}

func creditPurchaseFromOrder(order OrderData) (*CreditPurchaseQuote, error) {
	_, hasVersion := order.Metadata[creditPurchaseVersionKey]
	_, hasAmount := order.Metadata[creditPurchaseAmountKey]
	_, hasTotal := order.Metadata[creditPurchaseTotalKey]
	if !hasVersion && !hasAmount && !hasTotal {
		return nil, nil
	}
	if order.Metadata[creditPurchaseVersionKey] != creditPurchaseVersion {
		return nil, permanentApplyError("credit purchase metadata version is invalid")
	}
	credit, creditErr := parseCreditPurchaseCents(order.Metadata[creditPurchaseAmountKey])
	subtotal, subtotalErr := parseCreditPurchaseCents(order.Metadata[creditPurchaseTotalKey])
	if creditErr != nil || subtotalErr != nil || credit <= 0 || credit > subtotal ||
		subtotal < minimumUSDPriceCents || subtotal > maximumUSDPriceCents {
		return nil, permanentApplyError("credit purchase metadata amounts are invalid")
	}
	if !strings.EqualFold(strings.TrimSpace(order.Currency), "usd") || !creditPurchaseChargeMatches(order, subtotal) {
		return nil, permanentApplyError("credit purchase paid amount or currency does not match the checkout")
	}
	return &CreditPurchaseQuote{
		CreditAmountCents:  credit,
		ProcessingFeeCents: subtotal - credit,
		SubtotalCents:      subtotal,
		Currency:           "usd",
	}, nil
}

// creditPurchaseChargeMatches accepts the quoted subtotal less a consistent
// Polar discount. Orders without subtotal_amount must be undiscounted.
func creditPurchaseChargeMatches(order OrderData, quotedSubtotal int64) bool {
	orderSubtotal := order.NetAmount
	if order.SubtotalAmount != nil {
		orderSubtotal = *order.SubtotalAmount
	}
	return orderSubtotal == quotedSubtotal &&
		order.DiscountAmount >= 0 &&
		order.NetAmount >= 0 &&
		order.NetAmount <= quotedSubtotal &&
		order.NetAmount == orderSubtotal-order.DiscountAmount
}

func parseCreditPurchaseCents(value any) (int64, error) {
	amount, ok := value.(string)
	if !ok {
		return 0, errors.New("credit purchase amount is not a string")
	}
	return strconv.ParseInt(amount, 10, 64)
}
