package organizations

import (
	"context"
	"fmt"
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
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}
	if !polar.Configured() {
		return &pb.ListHostedCreditProductsResponse{}, nil
	}

	packs, err := polar.NewClientFromEnv().ListCreditPacks(ctx)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list hosted credit products")
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
	organizationID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}
	if !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}
	productID := strings.TrimSpace(req.GetProductId())
	if productID == "" {
		return nil, grpcerrors.InvalidArgument(nil, "product id is required")
	}

	email := ""
	if strings.TrimSpace(accountID) != "" {
		account, err := models.FindAccountByID(accountID)
		if err == nil && account != nil {
			email = account.Email
		}
	}

	client := polar.NewClientFromEnv()
	customer, err := client.EnsureCustomer(ctx, organizationID.String(), email)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create hosted credit checkout")
	}
	if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customer.ID); err != nil {
		return nil, grpcerrors.Internal(err, "failed to create hosted credit checkout")
	}

	customerIP := strings.TrimSpace(req.GetCustomerIpAddress())
	if customerIP == "" {
		customerIP = clientIPFromContext(ctx)
	}

	successURL := strings.TrimRight(strings.TrimSpace(baseURL), "/") +
		fmt.Sprintf("/%s/organization/llm-spend?credit=added", organizationID.String())
	if strings.TrimSpace(baseURL) == "" {
		successURL = strings.TrimRight(os.Getenv("BASE_URL"), "/") +
			fmt.Sprintf("/%s/organization/llm-spend?credit=added", organizationID.String())
	}

	session, err := client.CreateCheckout(ctx, productID, organizationID.String(), email, successURL, customerIP)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create hosted credit checkout")
	}
	return &pb.CreateHostedCreditCheckoutResponse{CheckoutUrl: session.URL}, nil
}

func CreateBillingPortalSession(
	ctx context.Context,
	orgID string,
	_ *pb.CreateBillingPortalSessionRequest,
) (*pb.CreateBillingPortalSessionResponse, error) {
	organizationID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}
	if !polar.Configured() {
		return nil, grpcerrors.FailedPrecondition(nil, "hosted billing is not configured")
	}

	settings, err := models.FindOrganizationLLMSettings(database.DB(ctx), organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create billing portal session")
	}

	client := polar.NewClientFromEnv()
	customerID := ""
	if settings != nil && settings.PolarCustomerID != nil {
		customerID = strings.TrimSpace(*settings.PolarCustomerID)
	}
	if customerID == "" {
		customer, lookupErr := client.GetCustomerByExternalID(ctx, organizationID.String())
		if lookupErr != nil {
			if polar.IsNotFound(lookupErr) {
				return nil, grpcerrors.FailedPrecondition(lookupErr, "Add hosted credit first.")
			}
			return nil, grpcerrors.Internal(lookupErr, "failed to create billing portal session")
		}
		customerID = customer.ID
		if err := models.SetOrganizationPolarCustomerID(database.DB(ctx), organizationID, customerID); err != nil {
			return nil, grpcerrors.Internal(err, "failed to create billing portal session")
		}
	}

	session, err := client.CreateCustomerSession(ctx, customerID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create billing portal session")
	}
	return &pb.CreateBillingPortalSessionResponse{PortalUrl: session.PortalURL}, nil
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

func clientIPFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{"x-forwarded-for", "grpcgateway-x-forwarded-for", "x-real-ip"} {
		values := md.Get(key)
		if len(values) == 0 {
			continue
		}
		return strings.TrimSpace(strings.Split(values[0], ",")[0])
	}
	return ""
}
