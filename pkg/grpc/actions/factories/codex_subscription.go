package factories

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/components/runner/codex"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func DescribeFactoryCodexConnection(
	ctx context.Context,
	organizationID string,
	req *pb.DescribeFactoryCodexConnectionRequest,
) (*pb.DescribeFactoryCodexConnectionResponse, error) {
	connection, eligible, callerID, err := loadCodexConnection(ctx, organizationID, req.GetFactoryId(), true)
	if err != nil {
		return nil, err
	}
	return &pb.DescribeFactoryCodexConnectionResponse{
		Connection: serializeCodexConnection(connection, eligible, callerID),
	}, nil
}

func StartFactoryCodexConnection(
	ctx context.Context,
	encryptor crypto.Encryptor,
	organizationID string,
	req *pb.StartFactoryCodexConnectionRequest,
) (*pb.StartFactoryCodexConnectionResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	callerID, err := codexCallerID(ctx)
	if err != nil {
		return nil, err
	}
	if encryptor == nil {
		return nil, grpcerrors.Internal(errors.New("encryptor is required"), "ChatGPT sign-in is not configured")
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return nil, grpcerrors.Internal(errors.New("jwt secret is empty"), "ChatGPT sign-in is not configured")
	}
	baseURL := runner.RunnerSuperplaneBaseURL("")
	if baseURL == "" {
		return nil, grpcerrors.Internal(errors.New("base url is empty"), "ChatGPT sign-in is not configured")
	}
	if req.GetRefreshCatalog() {
		return refreshFactoryCodexCatalog(ctx, encryptor, orgID, callerID, req.GetFactoryId(), secret, baseURL)
	}

	var connection *models.FactoryCodexConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, lockErr := findFactory(tx, orgID, req.GetFactoryId())
		if lockErr != nil {
			return lockErr
		}
		member, memberErr := models.ActiveCodexMember(tx, orgID, callerID)
		if memberErr != nil {
			return memberErr
		}
		if member.ID != callerID {
			return models.ErrCodexConnectionNotEligible
		}
		connection, lockErr = findOrCreateCodexConnection(tx, orgID, factory.ID, member.ID, time.Now())
		if lockErr != nil {
			return lockErr
		}
		if connection.OwnerUserID != member.ID {
			return models.ErrCodexConnectionNotEligible
		}
		taskID := uuid.New()
		now := time.Now()
		if beginErr := connection.BeginLogin(tx, taskID, now); beginErr != nil {
			return beginErr
		}
		token, tokenErr := runner.MintCodexSubscriptionToken(jwt.NewSigner(secret), runner.CodexSubscriptionScope{
			OrganizationID: orgID,
			FactoryID:      factory.ID,
			ConnectionID:   connection.ID,
			TaskID:         taskID,
			Generation:     connection.ReservationGeneration,
		}, models.CodexLoginTimeout)
		if tokenErr != nil {
			return tokenErr
		}
		return createCodexConnectionTask(tx, encryptor, orgID, taskID, token, baseURL, "Sign in to ChatGPT", "login")
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start ChatGPT sign-in")
	}
	_ = messages.PublishFactoryCodexConnectionUpdated(connection.FactoryID.String())
	return &pb.StartFactoryCodexConnectionResponse{
		Connection: serializeCodexConnection(connection, true, callerID),
	}, nil
}

func refreshFactoryCodexCatalog(
	ctx context.Context,
	encryptor crypto.Encryptor,
	organizationID, callerID uuid.UUID,
	factoryRef, secret, baseURL string,
) (*pb.StartFactoryCodexConnectionResponse, error) {
	var connection *models.FactoryCodexConnection
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := findFactory(tx, organizationID, factoryRef)
		if err != nil {
			return err
		}
		member, err := models.ActiveCodexMember(tx, organizationID, callerID)
		if err != nil {
			return err
		}
		if member.ID != callerID {
			return models.ErrCodexConnectionNotEligible
		}
		connection, err = models.LockFactoryCodexConnection(tx, organizationID, factory.ID, callerID)
		if err != nil {
			return err
		}
		if connection.OwnerUserID != member.ID {
			return models.ErrCodexConnectionNotEligible
		}
		if len(connection.AvailableModels) > 0 {
			return nil
		}

		taskID := uuid.New()
		if err := connection.BeginCatalogRefresh(tx, taskID, time.Now()); err != nil {
			return err
		}
		token, err := runner.MintCodexSubscriptionToken(jwt.NewSigner(secret), runner.CodexSubscriptionScope{
			OrganizationID: organizationID,
			FactoryID:      factory.ID,
			ConnectionID:   connection.ID,
			TaskID:         taskID,
			Generation:     connection.ReservationGeneration,
		}, models.CodexLoginTimeout)
		if err != nil {
			return err
		}
		return createCodexConnectionTask(tx, encryptor, organizationID, taskID, token, baseURL, "Refresh ChatGPT models", "refresh")
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to refresh ChatGPT models")
	}
	_ = messages.PublishFactoryCodexConnectionUpdated(connection.FactoryID.String())
	return &pb.StartFactoryCodexConnectionResponse{
		Connection: serializeCodexConnection(connection, true, callerID),
	}, nil
}

func createCodexConnectionTask(
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	organizationID, taskID uuid.UUID,
	token, baseURL, name, command string,
) error {
	tasks := contexts.NewRunnerTaskContext(tx, encryptor, organizationID)
	_, err := runner.CreateTask(tasks, runner.CreateTaskParams{
		TaskID:      taskID.String(),
		MachineType: runner.MachineTypeE1LargeAMD64,
		Commands: []runner.TaskCommand{{
			Name:    name,
			Command: `node "$SUPERPLANE_TASK_DIR/codex_subscription.js" ` + command,
			Kind:    runner.LiveLogKindSetup,
		}},
		Files: []runner.TaskFile{{
			Path:    "codex_subscription.js",
			Content: codex.SubscriptionScript(),
			Mode:    "0644",
		}},
		Environment: []runner.TaskEnvironmentVariable{
			{Name: runner.EnvCodexSubscriptionToken, Value: token},
			{Name: runner.EnvSuperplaneBaseURL, Value: baseURL},
		},
		ExecutionMode:  runner.ExecutionModeHost,
		TimeoutSeconds: int(models.CodexLoginTimeout / time.Second),
	})
	return err
}

func CancelFactoryCodexConnection(
	ctx context.Context,
	encryptor crypto.Encryptor,
	organizationID string,
	req *pb.CancelFactoryCodexConnectionRequest,
) (*pb.CancelFactoryCodexConnectionResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	callerID, err := codexCallerID(ctx)
	if err != nil {
		return nil, err
	}
	var connection *models.FactoryCodexConnection
	var eligible bool
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, lockErr := findFactory(tx, orgID, req.GetFactoryId())
		if lockErr != nil {
			return lockErr
		}
		eligible, lockErr = codexEligible(tx, orgID, callerID)
		if lockErr != nil {
			return lockErr
		}
		connection, lockErr = models.FindFactoryCodexConnection(tx, orgID, factory.ID, callerID)
		if lockErr != nil {
			return lockErr
		}
		if err := connection.AttachLoginLease(tx); err != nil {
			return err
		}
		if connection.LoginLease == nil {
			return nil
		}
		tasks := contexts.NewRunnerTaskContext(tx, encryptor, orgID)
		if cancelErr := tasks.RequestCancel(connection.LoginLease.TaskID.String()); cancelErr != nil && !errors.Is(cancelErr, models.ErrRunnerTaskNotFound) {
			return cancelErr
		}
		if err := models.ReleaseCodexReservationForTask(tx, connection.LoginLease.TaskID, time.Now()); err != nil {
			return err
		}
		connection, lockErr = models.FindFactoryCodexConnection(tx, orgID, factory.ID, callerID)
		if lockErr != nil {
			return lockErr
		}
		return connection.AttachLoginLease(tx)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to cancel ChatGPT sign-in")
	}
	_ = messages.PublishFactoryCodexConnectionUpdated(connection.FactoryID.String())
	return &pb.CancelFactoryCodexConnectionResponse{
		Connection: serializeCodexConnection(connection, eligible, callerID),
	}, nil
}

func DisconnectFactoryCodexConnection(
	ctx context.Context,
	organizationID string,
	req *pb.DisconnectFactoryCodexConnectionRequest,
) (*pb.DisconnectFactoryCodexConnectionResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	callerID, err := codexCallerID(ctx)
	if err != nil {
		return nil, err
	}
	var connection *models.FactoryCodexConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, lockErr := findFactory(tx, orgID, req.GetFactoryId())
		if lockErr != nil {
			return lockErr
		}
		member, memberErr := models.ActiveCodexMember(tx, orgID, callerID)
		if memberErr != nil {
			return memberErr
		}
		if member.ID != callerID {
			return models.ErrCodexConnectionNotEligible
		}
		connection, lockErr = models.LockFactoryCodexConnection(tx, orgID, factory.ID, callerID)
		if lockErr != nil {
			return lockErr
		}
		if connection.OwnerUserID != member.ID {
			return models.ErrCodexConnectionNotEligible
		}
		return connection.Disconnect(tx, time.Now())
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect ChatGPT")
	}
	_ = messages.PublishFactoryCodexConnectionUpdated(connection.FactoryID.String())
	return &pb.DisconnectFactoryCodexConnectionResponse{
		Connection: serializeCodexConnection(connection, true, callerID),
	}, nil
}

func loadCodexConnection(
	ctx context.Context,
	organizationID, factoryRef string,
	release bool,
) (*models.FactoryCodexConnection, bool, uuid.UUID, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, false, uuid.Nil, err
	}
	callerID, err := codexCallerID(ctx)
	if err != nil {
		return nil, false, uuid.Nil, err
	}
	var connection *models.FactoryCodexConnection
	var eligible bool
	var beforeRelease []models.FactoryCodexConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, lockErr := findFactory(tx, orgID, factoryRef)
		if lockErr != nil {
			return lockErr
		}
		var eligibleErr error
		eligible, eligibleErr = codexEligible(tx, orgID, callerID)
		if eligibleErr != nil {
			return eligibleErr
		}
		connection, lockErr = models.FindFactoryCodexConnection(tx, orgID, factory.ID, callerID)
		if errors.Is(lockErr, models.ErrCodexConnectionNotFound) {
			connection = nil
			return nil
		}
		if lockErr != nil {
			return lockErr
		}
		if release {
			beforeRelease = []models.FactoryCodexConnection{*connection}
			if err := models.ReleaseConfirmedCodexReservations(tx, time.Now(), connection.ID); err != nil {
				return err
			}
			connection, lockErr = models.FindFactoryCodexConnection(tx, orgID, factory.ID, callerID)
			if lockErr != nil {
				return lockErr
			}
		}
		return connection.AttachLoginLease(tx)
	})
	if err != nil {
		return nil, false, uuid.Nil, factoryErrorToStatus(err, "failed to describe ChatGPT subscription")
	}
	messages.PublishCodexConnectionChanges(database.DB(ctx), beforeRelease)
	return connection, eligible, callerID, nil
}

func lockFactory(tx *gorm.DB, organizationID uuid.UUID, ref string) (*models.Factory, error) {
	factory, err := findFactory(tx, organizationID, ref)
	if err != nil {
		return nil, err
	}
	if err := factory.LockForUpdate(tx); err != nil {
		return nil, err
	}
	return factory, nil
}

func findOrCreateCodexConnection(
	tx *gorm.DB,
	organizationID, factoryID, ownerID uuid.UUID,
	now time.Time,
) (*models.FactoryCodexConnection, error) {
	connection, err := models.LockFactoryCodexConnection(tx, organizationID, factoryID, ownerID)
	if err == nil {
		return connection, nil
	}
	if !errors.Is(err, models.ErrCodexConnectionNotFound) {
		return nil, err
	}
	_, err = models.CreateFactoryCodexConnection(tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "factory_id"}, {Name: "owner_user_id"}}, DoNothing: true,
	}), organizationID, factoryID, ownerID, now)
	if err != nil {
		return nil, err
	}
	return models.LockFactoryCodexConnection(tx, organizationID, factoryID, ownerID)
}

func codexEligible(tx *gorm.DB, organizationID, callerID uuid.UUID) (bool, error) {
	_, err := models.ActiveCodexMember(tx, organizationID, callerID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, models.ErrCodexConnectionUnavailable) {
		return false, nil
	}
	return false, err
}

func codexCallerID(ctx context.Context) (uuid.UUID, error) {
	raw, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, grpcerrors.Unauthenticated(errors.New("user is missing"), "Sign in before you connect ChatGPT")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, grpcerrors.Unauthenticated(err, "Sign in before you connect ChatGPT")
	}
	return id, nil
}

func serializeCodexConnection(
	connection *models.FactoryCodexConnection,
	eligible bool,
	callerID uuid.UUID,
) *pb.FactoryCodexConnection {
	view := &pb.FactoryCodexConnection{
		Status:   models.CodexConnectionStatusDisconnected,
		Eligible: eligible,
	}
	if connection == nil {
		return view
	}
	view.Id = connection.ID.String()
	view.RevocationPending = connection.RevocationPendingAt != nil
	view.Status = connection.Status
	if connection.Status == models.CodexConnectionStatusPending && connection.LoginCancellationRequested {
		view.Status = models.CodexConnectionStatusCanceling
	}
	view.AccountLabel = connection.AccountLabel
	view.ReservationHeld = connection.Reserved()
	view.AvailableModels = models.CompactModelIDs(codexModelIDs(connection.AvailableModels))
	if connection.LoginLease != nil {
		view.ActiveTaskId = connection.LoginLease.TaskID.String()
	}
	if eligible && callerID == connection.OwnerUserID && view.Status == models.CodexConnectionStatusPending {
		view.VerificationUrl = connection.VerificationURL
		view.VerificationCode = connection.VerificationCode
	}
	return view
}

func codexModelIDs(available []models.CodexModel) []string {
	ids := make([]string, 0, len(available))
	for _, model := range available {
		ids = append(ids, model.Model)
	}
	return ids
}
