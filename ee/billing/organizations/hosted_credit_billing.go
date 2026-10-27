package organizations

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/grpc/organizationref"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

func ListHostedCreditProducts(
	ctx context.Context,
	orgID string,
	_ *pb.ListHostedCreditProductsRequest,
) (*pb.ListHostedCreditProductsResponse, error) {
	organizationID, err := organizationref.Resolve(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !billingmodels.BillingEnabled() || !polar.Configured() {
		return &pb.ListHostedCreditProductsResponse{}, nil
	}

	plan, err := billingmodels.ResolveOrganizationBillingPlan(database.DB(ctx), organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list hosted credit products")
	}
	if !plan.AllowsCreditPurchase() {
		return &pb.ListHostedCreditProductsResponse{BillingEnabled: true}, nil
	}

	packs, err := availableHostedCreditPacks(ctx)
	if err != nil {
		return nil, polarBillingError(err, "failed to list hosted credit products")
	}

	products := make([]*pb.HostedCreditProduct, 0, len(packs))
	for _, pack := range packs {
		if !strings.EqualFold(pack.Currency, "usd") {
			continue
		}
		products = append(products, &pb.HostedCreditProduct{
			Id:          pack.ID,
			Name:        pack.Name,
			AmountCents: pack.AmountCents,
		})
	}
	return &pb.ListHostedCreditProductsResponse{
		BillingEnabled: true,
		Products:       products,
	}, nil
}

func availableHostedCreditPacks(ctx context.Context) ([]polar.Product, error) {
	tx := database.DB(ctx)
	configured, err := billingmodels.CreditPackCatalogConfigured(tx)
	if err != nil {
		return nil, err
	}
	if !configured {
		return polar.NewClientFromEnv().ListCreditPacks(ctx)
	}
	packs, err := billingmodels.ListCreditPacks(tx)
	if err != nil {
		return nil, err
	}
	products := make([]polar.Product, 0, len(packs))
	for _, pack := range packs {
		if !pack.Enabled || pack.PolarProductID == nil {
			continue
		}
		products = append(products, polar.Product{
			ID: *pack.PolarProductID, Name: pack.Name, AmountCents: pack.AmountCents, Currency: "usd",
		})
	}
	return products, nil
}

func hostedCreditPack(ctx context.Context, productID string) (*polar.Product, error) {
	tx := database.DB(ctx)
	configured, err := billingmodels.CreditPackCatalogConfigured(tx)
	if err != nil {
		return nil, err
	}
	if !configured {
		return polar.NewClientFromEnv().GetCreditPack(ctx, productID)
	}
	pack, err := billingmodels.FindCreditPackByProduct(tx, productID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, polar.ErrNotCreditPack
	}
	if err != nil {
		return nil, err
	}
	if !pack.Enabled {
		return nil, polar.ErrNotCreditPack
	}
	product, err := polar.NewClientFromEnv().GetCheckoutProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	product.Name = pack.Name
	if pack.AmountCents > 0 {
		// Checkout uses an ad-hoc fixed price. Polar catalog prices and custom
		// amount limits do not change the credit amount set by the admin.
		product.AmountCents = pack.AmountCents
		product.CustomPrice = false
		product.MinimumAmountCents = 0
		product.MaximumAmountCents = 0
		return product, nil
	}
	product.AmountCents = 0
	product.CustomPrice = true
	return product, nil
}

func QuoteHostedCreditCheckout(
	ctx context.Context,
	orgID string,
	req *pb.QuoteHostedCreditCheckoutRequest,
) (*pb.QuoteHostedCreditCheckoutResponse, error) {
	_, err := hostedCreditPurchaseOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	quote, err := hostedCreditPurchaseQuote(ctx, req.GetProductId(), req.GetCreditAmountCents())
	if err != nil {
		return nil, err
	}
	return &pb.QuoteHostedCreditCheckoutResponse{
		CreditAmountCents:  quote.CreditAmountCents,
		ProcessingFeeCents: quote.ProcessingFeeCents,
		SubtotalCents:      quote.SubtotalCents,
		Currency:           quote.Currency,
		FixedFeeCents:      quote.FixedFeeCents,
		PercentageFeeBps:   int32(quote.PercentageFeeBPS),
		PercentageFeeCents: quote.PercentageFeeCents,
	}, nil
}

func hostedCreditPurchaseOrganization(ctx context.Context, orgID string) (uuid.UUID, error) {
	organizationID, err := organizationref.Resolve(ctx, orgID)
	if err != nil {
		return uuid.Nil, err
	}
	if !billingmodels.BillingEnabled() || !polar.Configured() {
		return uuid.Nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}
	plan, err := billingmodels.ResolveOrganizationBillingPlan(database.DB(ctx), organizationID)
	if err != nil {
		return uuid.Nil, grpcerrors.Internal(err, "failed to load hosted credit billing")
	}
	if !plan.AllowsCreditPurchase() {
		return uuid.Nil, grpcerrors.FailedPrecondition(nil, "An active paid plan is required to purchase hosted credit.")
	}
	return organizationID, nil
}

func hostedCreditPurchaseQuote(ctx context.Context, productID string, creditCents int64) (*polar.CreditPurchaseQuote, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, grpcerrors.InvalidArgument(nil, "product id is required")
	}
	pack, err := hostedCreditPack(ctx, productID)
	if err != nil {
		if polar.IsNotFound(err) || errors.Is(err, polar.ErrNotCreditPack) {
			return nil, grpcerrors.InvalidArgument(err, "product is not a hosted credit pack")
		}
		return nil, polarBillingError(err, "failed to quote hosted credit")
	}
	if !strings.EqualFold(pack.Currency, "usd") {
		return nil, grpcerrors.InvalidArgument(nil, "Hosted credit purchases require a USD product.")
	}
	if !pack.CustomPrice {
		if creditCents != 0 && creditCents != pack.AmountCents {
			return nil, grpcerrors.FailedPrecondition(nil, "The credit amount changed. Review the updated price.")
		}
		creditCents = pack.AmountCents
	}
	if creditCents <= 0 || creditCents < pack.MinimumAmountCents || (pack.MaximumAmountCents > 0 && creditCents > pack.MaximumAmountCents) {
		return nil, grpcerrors.InvalidArgument(nil, "Enter a credit amount within the product limits.")
	}
	settings, err := billingmodels.GetInstallationLLMSettings(database.DB(ctx))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load credit purchase fees")
	}
	quote, err := polar.CalculateCreditPurchase(creditCents, settings.CreditPurchaseFeeBPS, settings.CreditPurchaseFixedFeeCents)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}
	return quote, nil
}

func CreateHostedCreditCheckout(
	ctx context.Context,
	orgID string,
	req *pb.CreateHostedCreditCheckoutRequest,
	accountID string,
	baseURL string,
) (*pb.CreateHostedCreditCheckoutResponse, error) {
	organizationID, err := hostedCreditPurchaseOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	productID := strings.TrimSpace(req.GetProductId())
	if productID == "" {
		return nil, grpcerrors.InvalidArgument(nil, "product id is required")
	}

	email := actingUserEmail(ctx, organizationID.String(), accountID)
	if email == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "A user email is required to start checkout.")
	}
	if req.GetCreditAmountCents() <= 0 || req.GetExpectedSubtotalCents() <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "Review the credit amount and subtotal before checkout.")
	}
	quote, err := hostedCreditPurchaseQuote(ctx, productID, req.GetCreditAmountCents())
	if err != nil {
		return nil, err
	}
	if quote.SubtotalCents != req.GetExpectedSubtotalCents() {
		return nil, grpcerrors.FailedPrecondition(nil, "The processing fee changed. Review the updated price.")
	}

	organization, err := models.FindOrganizationByIDInTransaction(database.DB(ctx), organizationID.String())
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	client := polar.NewClientFromEnv()
	customer, err := client.EnsureCustomer(ctx, polar.CreateCustomerInput{
		ExternalID: organizationID.String(),
		Name:       organization.Name,
		OwnerEmail: email,
	})
	if err != nil {
		if polar.IsConflict(err) {
			return nil, grpcerrors.FailedPrecondition(err, "hosted billing customer already exists for another organization")
		}
		return nil, polarBillingError(err, "failed to create hosted credit checkout")
	}
	if err := billingmodels.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customer.ID); err != nil {
		return nil, grpcerrors.Internal(err, "failed to create hosted credit checkout")
	}

	// Polar tax and currency use this IP. Prefer the proxy-set client IP, not a
	// client-supplied body field (CreateHostedCreditCheckoutRequest.customer_ip_address is ignored).
	customerIP := clientIPFromContext(ctx)

	session, err := client.CreateCreditCheckout(ctx, productID, organizationID.String(), hostedCreditCheckoutSuccessURL(baseURL, organizationID), customerIP, quote)
	if err != nil {
		return nil, polarBillingError(err, "failed to create hosted credit checkout")
	}
	if session.CustomerID != "" {
		if err := billingmodels.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, session.CustomerID); err != nil {
			return nil, grpcerrors.Internal(err, "failed to create hosted credit checkout")
		}
	}
	return &pb.CreateHostedCreditCheckoutResponse{CheckoutUrl: session.URL}, nil
}

func CreateBillingPortalSession(
	ctx context.Context,
	orgID string,
	_ *pb.CreateBillingPortalSessionRequest,
) (*pb.CreateBillingPortalSessionResponse, error) {
	organizationID, err := organizationref.Resolve(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !billingmodels.BillingEnabled() || !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}

	settings, err := billingmodels.FindOrganizationLLMSettings(database.DB(ctx), organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create billing portal session")
	}

	client := polar.NewClientFromEnv()
	session, err := openPolarPortal(ctx, client, polar.CustomerSessionRequest{
		ExternalCustomerID: organizationID.String(),
		ExternalMemberID:   organizationID.String(),
	})
	if err != nil && polar.IsNotFound(err) {
		customerID := ""
		if settings != nil && settings.PolarCustomerID != nil {
			customerID = strings.TrimSpace(*settings.PolarCustomerID)
		}
		if customerID == "" {
			resolved, lookupErr := lookupPolarCustomer(ctx, client, organizationID)
			if lookupErr != nil {
				return nil, lookupErr
			}
			customerID = resolved
		}
		session, err = openPolarPortal(ctx, client, polar.CustomerSessionRequest{
			CustomerID:       customerID,
			ExternalMemberID: organizationID.String(),
		})
	}
	if err != nil && polar.IsNotFound(err) {
		resolved, lookupErr := lookupPolarCustomer(ctx, client, organizationID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		session, err = openPolarPortal(ctx, client, polar.CustomerSessionRequest{
			CustomerID:       resolved,
			ExternalMemberID: organizationID.String(),
		})
	}
	if err != nil {
		return nil, polarBillingError(err, "failed to create billing portal session")
	}
	if session.CustomerID != "" {
		if err := billingmodels.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, session.CustomerID); err != nil {
			return nil, grpcerrors.Internal(err, "failed to create billing portal session")
		}
	}
	return &pb.CreateBillingPortalSessionResponse{PortalUrl: session.PortalURL}, nil
}

func openPolarPortal(ctx context.Context, client *polar.Client, req polar.CustomerSessionRequest) (*polar.CustomerSession, error) {
	session, err := client.CreateCustomerSession(ctx, req)
	if !polar.IsTeamMemberRequired(err) {
		return session, err
	}

	memberID, memberErr := client.GetOwnerMember(ctx, req.ExternalCustomerID, req.CustomerID)
	if memberErr != nil {
		return nil, memberErr
	}
	req.MemberID = memberID
	req.ExternalMemberID = ""
	return client.CreateCustomerSession(ctx, req)
}

func lookupPolarCustomer(ctx context.Context, client *polar.Client, organizationID uuid.UUID) (string, error) {
	customer, err := client.GetCustomerByExternalID(ctx, organizationID.String())
	if err != nil {
		if polar.IsNotFound(err) {
			return "", grpcerrors.FailedPrecondition(err, "Add hosted credit first.")
		}
		return "", polarBillingError(err, "failed to create billing portal session")
	}
	if err := billingmodels.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customer.ID); err != nil {
		return "", grpcerrors.Internal(err, "failed to create billing portal session")
	}
	return customer.ID, nil
}

func polarBillingError(err error, fallback string) error {
	if polar.IsRateLimited(err) {
		return grpcerrors.ResourceExhausted(err, "Hosted billing is busy. Try again shortly.")
	}
	if polar.IsUnauthorized(err) {
		return grpcerrors.FailedPrecondition(err, "Hosted billing token is missing a required Polar scope.")
	}
	return grpcerrors.Internal(err, fallback)
}

func actingUserEmail(ctx context.Context, orgID, accountID string) string {
	if userID := metadataValue(ctx, "x-user-id"); userID != "" {
		user, err := models.FindActiveUserByIDInTransaction(database.DB(ctx), orgID, userID)
		if err == nil && user != nil {
			if email := strings.TrimSpace(user.GetEmail()); email != "" {
				return email
			}
		}
	}
	if strings.TrimSpace(accountID) == "" {
		return ""
	}
	account, err := models.FindAccountByID(accountID)
	if err != nil || account == nil {
		return ""
	}
	return strings.TrimSpace(account.Email)
}

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return firstMetadataValue(md, key)
}

func billingState(ctx context.Context, orgID uuid.UUID) (enabled bool, hasCustomer bool) {
	if !billingmodels.BillingEnabled() || !polar.Configured() {
		return false, false
	}
	settings, err := billingmodels.FindOrganizationLLMSettings(database.DB(ctx), orgID)
	if err != nil {
		return true, false
	}
	if settings != nil && settings.PolarCustomerID != nil && strings.TrimSpace(*settings.PolarCustomerID) != "" {
		return true, true
	}
	return true, false
}

func hostedCreditCheckoutSuccessURL(baseURL string, organizationID uuid.UUID) string {
	origin := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if origin == "" {
		origin = strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/")
	}
	return origin + "/" + organizationID.String() + "/organization/billing?credit=added&checkout_id={CHECKOUT_ID}"
}

func clientIPFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{
		"cf-connecting-ip",
		"grpcgateway-cf-connecting-ip",
		"true-client-ip",
		"grpcgateway-true-client-ip",
		"x-real-ip",
		"grpcgateway-x-real-ip",
	} {
		if ip := firstMetadataValue(md, key); ip != "" {
			return ip
		}
	}
	for _, key := range []string{"x-forwarded-for", "grpcgateway-x-forwarded-for"} {
		if ip := rightMostForwardedIP(md.Get(key)); ip != "" {
			return ip
		}
	}
	return ""
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func rightMostForwardedIP(values []string) string {
	if len(values) == 0 {
		return ""
	}
	parts := strings.Split(values[0], ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if ip := strings.TrimSpace(parts[i]); ip != "" {
			return ip
		}
	}
	return ""
}
