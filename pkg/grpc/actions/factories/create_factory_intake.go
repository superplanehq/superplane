package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"gorm.io/gorm"
)

// IntakeDependencies carries what creating an intake canvas needs. An intake
// owns a canvas, so the intake actions have the same dependencies as the canvas
// actions they delegate to.
type IntakeDependencies struct {
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	GitProvider    git.Provider
	WebhookBaseURL string
	UsageService   usage.Service
}

func CreateFactoryIntake(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CreateFactoryIntakeRequest,
) (*pb.CreateFactoryIntakeResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	source, err := parseFactoryIntakeSource(req.GetSource())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		name = intakeDefaultName(source)
	}
	name, err = models.AvailableCanvasName(db, orgID, name)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	confidencePct := DefaultIntakeConfidencePct
	if req.ConfidencePct != nil {
		confidencePct = int(req.GetConfidencePct())
	}

	canvasID, err := createIntakeCanvas(ctx, deps, orgID, factoryID, source, name, confidencePct)
	if err != nil {
		return nil, err
	}

	intake, err := factory.CreateIntake(db, canvasID, source)
	if err != nil {
		// The canvas is live but nothing claims it as an intake, so it would
		// linger as an unexplained factory app. Retire it.
		discardIntakeCanvas(db, orgID, canvasID)
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	intake, err = factory.FindIntake(db, intake.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	spec, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{canvasID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	return &pb.CreateFactoryIntakeResponse{
		Intake: serializeFactoryIntake(intake, spec[canvasID]),
	}, nil
}

// createIntakeCanvas builds the intake graph and commits it as the canvas's
// live version in one step. The graph has to be live from the start: a staged
// graph never receives events.
func createIntakeCanvas(
	ctx context.Context,
	deps IntakeDependencies,
	orgID, factoryID uuid.UUID,
	source, name string,
	confidencePct int,
) (uuid.UUID, error) {
	canvasDoc, err := buildIntakeCanvas(source, name, confidencePct)
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	nodes, edges, err := canvasDoc.Parse(deps.Registry, orgID.String())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to build intake automation")
	}

	response, err := canvases.CreateCanvasWithSeedFiles(
		ctx,
		deps.Registry,
		deps.Encryptor,
		deps.AuthService,
		deps.GitProvider,
		deps.WebhookBaseURL,
		orgID,
		canvasDoc.Metadata.Name,
		canvasDoc.Metadata.Description,
		&factoryID,
		nodes,
		edges,
		deps.UsageService,
		nil,
	)
	if err != nil {
		return uuid.Nil, err
	}

	canvasID, err := uuid.Parse(response.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create factory intake")
	}

	return canvasID, nil
}

func discardIntakeCanvas(db *gorm.DB, orgID, canvasID uuid.UUID) {
	canvas, err := models.FindCanvasInTransaction(db, orgID, canvasID)
	if err != nil {
		log.Errorf("failed to load intake canvas %s for cleanup: %v", canvasID, err)
		return
	}

	if err := canvas.SoftDeleteInTransaction(db); err != nil {
		log.Errorf("failed to discard intake canvas %s: %v", canvasID, err)
	}
}
