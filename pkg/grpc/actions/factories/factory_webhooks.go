package factories

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	webhooktrigger "github.com/superplanehq/superplane/pkg/triggers/webhook"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const factoryWebhookNameMaxLength = 100

var factoryWebhookHeaderPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,100}$`)

var errFactoryWebhookInUse = errors.New("webhook is in use")

func ListFactoryWebhooks(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.ListFactoryWebhooksRequest,
) (*pb.ListFactoryWebhooksResponse, error) {
	db := database.DB(ctx)
	orgID, factory, err := factoryForWebhooks(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to list webhooks")
	}
	webhooks, err := models.ListFactoryWebhooks(db, orgID, factory.ID)
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to list webhooks")
	}
	ids := make([]uuid.UUID, 0, len(webhooks))
	for _, webhook := range webhooks {
		ids = append(ids, webhook.ID)
	}
	counts, err := models.CountActiveWebhookNodes(db, ids)
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to list webhooks")
	}
	out := make([]*pb.FactoryWebhook, 0, len(webhooks))
	for i := range webhooks {
		out = append(out, serializeFactoryWebhook(deps, &webhooks[i], counts[webhooks[i].ID]))
	}
	return &pb.ListFactoryWebhooksResponse{Webhooks: out}, nil
}

func CreateFactoryWebhook(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CreateFactoryWebhookRequest,
) (*pb.CreateFactoryWebhookResponse, error) {
	db := database.DB(ctx)
	orgID, factory, err := factoryForWebhooks(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to create webhook")
	}
	name, config, err := parseFactoryWebhookSettings(req.GetName(), req.GetAuthentication(), req.GetHeaderName())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to create webhook")
	}

	id := uuid.New()
	secret, encrypted, err := crypto.NewRandomKey(ctx, encryption.ForDB(db, deps.Encryptor), id.String())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to create webhook")
	}
	webhook := models.NewFactoryWebhook(id, orgID, factory.ID, name, config, encrypted)
	if err := webhook.Create(db); err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to create webhook")
	}
	return &pb.CreateFactoryWebhookResponse{
		Webhook: serializeFactoryWebhook(deps, &webhook, 0),
		Secret:  secret,
	}, nil
}

func UpdateFactoryWebhook(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryWebhookRequest,
) (*pb.UpdateFactoryWebhookResponse, error) {
	db := database.DB(ctx)
	orgID, factory, err := factoryForWebhooks(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to update webhook")
	}
	name, config, err := parseFactoryWebhookSettings(req.GetName(), req.GetAuthentication(), req.GetHeaderName())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to update webhook")
	}
	webhook, err := findFactoryWebhook(db, orgID, factory.ID, req.GetWebhookId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to update webhook")
	}
	if err := webhook.UpdateSettings(db, name, config); err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to update webhook")
	}
	counts, err := models.CountActiveWebhookNodes(db, []uuid.UUID{webhook.ID})
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to update webhook")
	}
	return &pb.UpdateFactoryWebhookResponse{Webhook: serializeFactoryWebhook(deps, webhook, counts[webhook.ID])}, nil
}

func DeleteFactoryWebhook(
	ctx context.Context,
	organizationID string,
	req *pb.DeleteFactoryWebhookRequest,
) (*pb.DeleteFactoryWebhookResponse, error) {
	db := database.DB(ctx)
	orgID, factory, err := factoryForWebhooks(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to delete webhook")
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		webhook, err := findFactoryWebhook(tx, orgID, factory.ID, req.GetWebhookId())
		if err != nil {
			return err
		}
		counts, err := models.CountActiveWebhookNodes(tx, []uuid.UUID{webhook.ID})
		if err != nil {
			return err
		}
		if counts[webhook.ID] > 0 {
			return errFactoryWebhookInUse
		}
		return tx.Delete(webhook).Error
	})
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to delete webhook")
	}
	return &pb.DeleteFactoryWebhookResponse{}, nil
}

func RegenerateFactoryWebhookSecret(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.RegenerateFactoryWebhookSecretRequest,
) (*pb.RegenerateFactoryWebhookSecretResponse, error) {
	db := database.DB(ctx)
	orgID, factory, err := factoryForWebhooks(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to regenerate webhook secret")
	}
	webhook, err := findFactoryWebhook(db, orgID, factory.ID, req.GetWebhookId())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to regenerate webhook secret")
	}
	secret, encrypted, err := crypto.NewRandomKey(ctx, encryption.ForDB(db, deps.Encryptor), webhook.ID.String())
	if err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to regenerate webhook secret")
	}
	if err := webhook.ReplaceSecret(db, encrypted); err != nil {
		return nil, factoryWebhookErrorToStatus(err, "failed to regenerate webhook secret")
	}
	return &pb.RegenerateFactoryWebhookSecretResponse{Secret: secret}, nil
}

func factoryForWebhooks(db *gorm.DB, organizationID, factoryRef string) (uuid.UUID, *models.Factory, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return uuid.Nil, nil, err
	}
	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return uuid.Nil, nil, err
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryCustomAutomations) {
		return uuid.Nil, nil, errCustomAutomationsDisabled
	}
	factory, err := findFactory(db, orgID, factoryRef)
	if err != nil {
		return uuid.Nil, nil, err
	}
	return orgID, factory, nil
}

func findFactoryWebhook(tx *gorm.DB, orgID, factoryID uuid.UUID, raw string) (*models.Webhook, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, invalidArgument("invalid webhook id")
	}
	webhook, err := models.FindFactoryWebhook(tx, orgID, factoryID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.NotFound(err, "Webhook not found.")
	}
	return webhook, err
}

func parseFactoryWebhookSettings(
	rawName string,
	authentication pb.FactoryWebhook_Authentication,
	rawHeader string,
) (string, models.WebhookAuthenticationConfig, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", models.WebhookAuthenticationConfig{}, invalidArgument("Enter a webhook name.")
	}
	if utf8.RuneCountInString(name) > factoryWebhookNameMaxLength {
		return "", models.WebhookAuthenticationConfig{}, invalidArgument("Use a webhook name of 100 characters or fewer.")
	}

	mode := factoryWebhookAuthenticationFromProto(authentication)
	if mode == "" {
		return "", models.WebhookAuthenticationConfig{}, invalidArgument("Select an authentication method.")
	}
	header := strings.TrimSpace(rawHeader)
	switch mode {
	case models.WebhookAuthenticationBearer:
		header = ""
	case models.WebhookAuthenticationSignature:
		if header == "" {
			header = webhooktrigger.DefaultSignatureHeader
		}
	case models.WebhookAuthenticationHeaderToken:
		if header == "" {
			header = webhooktrigger.DefaultHeaderTokenName
		}
	}
	if header != "" && !factoryWebhookHeaderPattern.MatchString(header) {
		return "", models.WebhookAuthenticationConfig{}, invalidArgument("Use only letters, numbers, and hyphens in the header name.")
	}
	return name, models.WebhookAuthenticationConfig{Authentication: mode, HeaderName: header}, nil
}

func factoryWebhookAuthenticationFromProto(value pb.FactoryWebhook_Authentication) string {
	switch value {
	case pb.FactoryWebhook_AUTHENTICATION_BEARER:
		return models.WebhookAuthenticationBearer
	case pb.FactoryWebhook_AUTHENTICATION_SIGNATURE:
		return models.WebhookAuthenticationSignature
	case pb.FactoryWebhook_AUTHENTICATION_HEADER_TOKEN:
		return models.WebhookAuthenticationHeaderToken
	}
	return ""
}

func factoryWebhookAuthenticationToProto(value string) pb.FactoryWebhook_Authentication {
	switch value {
	case models.WebhookAuthenticationBearer:
		return pb.FactoryWebhook_AUTHENTICATION_BEARER
	case models.WebhookAuthenticationSignature:
		return pb.FactoryWebhook_AUTHENTICATION_SIGNATURE
	case models.WebhookAuthenticationHeaderToken:
		return pb.FactoryWebhook_AUTHENTICATION_HEADER_TOKEN
	}
	return pb.FactoryWebhook_AUTHENTICATION_UNSPECIFIED
}

func serializeFactoryWebhook(deps IntakeDependencies, webhook *models.Webhook, automationCount int) *pb.FactoryWebhook {
	config := webhook.AuthenticationConfig()
	out := &pb.FactoryWebhook{
		Id:              webhook.ID.String(),
		Name:            webhook.DisplayName(),
		Url:             contexts.WebhookCallbackURL(deps.WebhookBaseURL, webhook.ID.String()),
		Authentication:  factoryWebhookAuthenticationToProto(config.Authentication),
		HeaderName:      config.HeaderName,
		AutomationCount: int32(automationCount),
	}
	if webhook.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*webhook.CreatedAt)
	}
	return out
}

func factoryWebhookErrorToStatus(err error, internalMessage string) error {
	switch {
	case errors.Is(err, models.ErrWebhookNameAlreadyExists):
		return grpcerrors.AlreadyExists(err, "A webhook with this name already exists in this workspace.")
	case errors.Is(err, errFactoryWebhookInUse):
		return grpcerrors.FailedPrecondition(err, "Remove this webhook from its automations before you delete it.")
	}
	return factoryErrorToStatus(err, internalMessage)
}
