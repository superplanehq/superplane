package organizations

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/metadata"
)

func ListHostedCreditProducts(
	ctx context.Context,
	orgID string,
	_ *pb.ListHostedCreditProductsRequest,
) (*pb.ListHostedCreditProductsResponse, error) {
	if _, err := resolveOrganizationID(ctx, orgID); err != nil {
		return nil, err
	}
	if !polar.Configured() {
		return &pb.ListHostedCreditProductsResponse{}, nil
	}

	packs, err := polar.NewClientFromEnv().ListCreditPacks(ctx)
	if err != nil {
		return nil, polarBillingError(err, "failed to list hosted credit products")
	}

	products := make([]*pb.HostedCreditProduct, 0, len(packs))
	for _, pack := range packs {
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

func CreateHostedCreditCheckout(
	ctx context.Context,
	orgID string,
	req *pb.CreateHostedCreditCheckoutRequest,
	accountID string,
	baseURL string,
) (*pb.CreateHostedCreditCheckoutResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}
	productID := strings.TrimSpace(req.GetProductId())
	if productID == "" {
		return nil, grpcerrors.InvalidArgument(nil, "product id is required")
	}

	email := actingUserEmail(ctx, organizationID.String(), accountID)
	if email == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "A user email is required to start checkout.")
	}

	organization, err := models.FindOrganizationByIDInTransaction(database.DB(ctx), organizationID.String())
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	client := polar.NewClientFromEnv()
	if _, err := client.GetCreditPack(ctx, productID); err != nil {
		if polar.IsNotFound(err) || errors.Is(err, polar.ErrNotCreditPack) {
			return nil, grpcerrors.InvalidArgument(err, "product is not a hosted credit pack")
		}
		return nil, polarBillingError(err, "failed to create hosted credit checkout")
	}

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
	if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customer.ID); err != nil {
		return nil, grpcerrors.Internal(err, "failed to create hosted credit checkout")
	}

	// Polar tax and currency use this IP. Prefer the proxy-set client IP, not a
	// client-supplied body field (CreateHostedCreditCheckoutRequest.customer_ip_address is ignored).
	customerIP := clientIPFromContext(ctx)

	session, err := client.CreateCheckout(ctx, productID, organizationID.String(), hostedCreditCheckoutSuccessURL(baseURL, organizationID), customerIP)
	if err != nil {
		return nil, polarBillingError(err, "failed to create hosted credit checkout")
	}
	if session.CustomerID != "" {
		if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, session.CustomerID); err != nil {
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
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}

	settings, err := models.FindOrganizationLLMSettings(database.DB(ctx), organizationID)
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
		if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, session.CustomerID); err != nil {
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
	if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customer.ID); err != nil {
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
	if !polar.Configured() {
		return false, false
	}
	settings, err := models.FindOrganizationLLMSettings(database.DB(ctx), orgID)
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
	return origin + "/" + organizationID.String() + "/organization/workspace-usage?credit=added&checkout_id={CHECKOUT_ID}"
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
