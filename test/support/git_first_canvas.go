package support

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitpkg "github.com/superplanehq/superplane/pkg/git"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const gitFirstWebhookBaseURL = "http://localhost:3000/api/v1"

// CreateCanvasGitFirst seeds a git repository and materializes the initial live version.
// Use this for E2E and integration tests that exercise git-native edit flows.
func CreateCanvasGitFirst(
	t require.TestingT,
	orgID uuid.UUID,
	userID uuid.UUID,
	nodes []models.CanvasNode,
	edges []models.Edge,
) (*models.Canvas, []models.CanvasNode) {
	reg, encryptor, authService, gitProvider := gitFirstDependencies(t)
	user, err := models.FindActiveUserByID(orgID.String(), userID.String())
	require.NoError(t, err)

	inputNodes := make([]models.Node, len(nodes))
	for i, node := range nodes {
		inputNodes[i] = models.Node{
			ID:            node.NodeID,
			Name:          node.Name,
			Type:          node.Type,
			Ref:           node.Ref.Data(),
			Configuration: node.Configuration.Data(),
			Metadata:      node.Metadata.Data(),
			Position:      node.Position.Data(),
			IsCollapsed:   node.IsCollapsed,
		}
	}

	changeManagementEnabled, err := models.IsChangeManagementEnabled(orgID)
	require.NoError(t, err)

	expandedNodes, err := expandBlueprintNodes(t, orgID, inputNodes)
	require.NoError(t, err)

	now := time.Now()
	canvasID := uuid.New()
	canvas := &models.Canvas{
		ID:             canvasID,
		OrganizationID: orgID,
		Name:           RandomName("canvas"),
		Description:    "Test canvas",
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	repoID := gitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: orgID,
		CanvasID:       canvasID,
		Name:           canvas.Name,
	})
	repository := &models.Repository{
		CanvasID:       canvasID,
		OrganizationID: orgID,
		Provider:       gitProvider.Name(),
		RepoID:         repoID,
		Status:         models.RepositoryStatusPending,
	}

	commitSHA, seedErr := materialize.SeedMainRepository(context.Background(), gitProvider, repository, materialize.SeedRepositoryInput{
		Name:                    canvas.Name,
		Description:             canvas.Description,
		Nodes:                   expandedNodes,
		Edges:                   edges,
		ChangeManagementEnabled: changeManagementEnabled,
		ChangeRequestApprovers:  models.DefaultCanvasChangeRequestApprovers(),
		Author: git.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})
	require.NoError(t, seedErr)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(canvas).Error; createErr != nil {
			return createErr
		}

		if repoErr := canvas.CreatePendingRepositoryInTransaction(tx, gitProvider.Name(), repoID); repoErr != nil {
			return repoErr
		}

		if markErr := tx.Model(&models.Repository{}).
			Where("canvas_id = ?", canvasID).
			Updates(map[string]any{
				"status":     models.RepositoryStatusReady,
				"updated_at": time.Now(),
			}).Error; markErr != nil {
			return markErr
		}

		_, matErr := materialize.SyncLiveFromGit(
			context.Background(),
			tx,
			gitProvider,
			reg,
			encryptor,
			authService,
			gitFirstWebhookBaseURL,
			orgID,
			canvasID,
			materialize.SyncLiveFromGitOptions{
				HeadSHA:                   commitSHA,
				SkipChangeManagementCheck: true,
			},
		)
		return matErr
	}))

	updatedCanvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	require.NoError(t, err)

	var createdNodes []models.CanvasNode
	require.NoError(t, database.Conn().
		Where("workflow_id = ?", canvasID).
		Find(&createdNodes).Error)

	return updatedCanvas, createdNodes
}

func gitFirstDependencies(t require.TestingT) (*registry.Registry, crypto.Encryptor, *authorization.AuthService, git.Provider) {
	encryptor := crypto.NewNoOpEncryptor()
	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)

	authService := AuthService(t)

	gitProvider, err := gitpkg.NewProvider()
	if err != nil || strings.TrimSpace(os.Getenv("GIT_STORAGE_PROVIDER")) == "" {
		return reg, encryptor, authService, inmemory.NewProvider()
	}

	return reg, encryptor, authService, gitProvider
}
