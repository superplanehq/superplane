package runner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/jwt"
)

type stubFactoryContext struct {
	link *core.LinkedWorkOrder
	ok   bool
	err  error
}

func (s *stubFactoryContext) CreateWorkOrder(core.WorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}
func (s *stubFactoryContext) UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}
func (s *stubFactoryContext) AddWorkOrderComment(core.AddWorkOrderCommentParams) error { return nil }
func (s *stubFactoryContext) AddWorkOrderArtifact(core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	return nil, nil
}
func (s *stubFactoryContext) LinkedWorkOrder() (*core.LinkedWorkOrder, bool, error) {
	return s.link, s.ok, s.err
}

func TestAppendFactoryRunnerEnvironment(t *testing.T) {
	orderID := uuid.NewString()
	factoryID := uuid.NewString()
	userID := uuid.NewString()
	orgID := uuid.NewString()
	executionID := uuid.New()

	t.Run("no factory context", func(t *testing.T) {
		env := appendFactoryRunnerEnvironment(core.ExecutionContext{}, nil, 60)
		assert.Empty(t, env)
	})

	t.Run("not linked", func(t *testing.T) {
		env := appendFactoryRunnerEnvironment(core.ExecutionContext{
			Factory: &stubFactoryContext{ok: false},
		}, nil, 60)
		assert.Empty(t, env)
	})

	t.Run("injects ids url and execution-bound token", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-session-secret-for-factory-runner")

		env := appendFactoryRunnerEnvironment(core.ExecutionContext{
			ID:             executionID,
			OrganizationID: orgID,
			BaseURL:        "https://app.example.com/",
			Factory: &stubFactoryContext{
				ok: true,
				link: &core.LinkedWorkOrder{
					ID:              orderID,
					FactoryID:       factoryID,
					CreatedByUserID: userID,
				},
			},
		}, nil, 120)

		byName := map[string]string{}
		for _, item := range env {
			byName[item.Name] = item.Value
		}
		assert.Equal(t, "https://app.example.com", byName[envSuperplaneURL])
		assert.Equal(t, factoryID, byName[envSuperplaneFactoryID])
		assert.Equal(t, orderID, byName[envSuperplaneOrderID])
		require.NotEmpty(t, byName[envSuperplaneToken])

		claims, err := jwt.NewSigner("test-session-secret-for-factory-runner").ValidateScopedToken(byName[envSuperplaneToken])
		require.NoError(t, err)
		assert.Equal(t, userID, claims.Subject)
		assert.Equal(t, orgID, claims.OrgID)
		assert.Equal(t, jwt.PurposeFactoryRunner, claims.Purpose)
		assert.Equal(t, executionID.String(), claims.ExecutionID)
		assert.Contains(t, claims.Scopes, "work_orders:read:"+orderID)
		assert.Contains(t, claims.Scopes, "work_orders:update:"+orderID)
	})

	t.Run("overwrites sticky env from prior task or node config", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-session-secret-for-factory-runner")

		existing := []BrokerEnvironmentVariable{
			{Name: envSuperplaneURL, Value: "https://evil.example"},
			{Name: envSuperplaneToken, Value: "stale-token-from-old-task"},
			{Name: envSuperplaneOrderID, Value: uuid.NewString()},
			{Name: envSuperplaneFactoryID, Value: uuid.NewString()},
		}
		env := appendFactoryRunnerEnvironment(core.ExecutionContext{
			ID:             executionID,
			OrganizationID: orgID,
			BaseURL:        "https://app.example.com",
			Factory: &stubFactoryContext{
				ok: true,
				link: &core.LinkedWorkOrder{
					ID:              orderID,
					FactoryID:       factoryID,
					CreatedByUserID: userID,
				},
			},
		}, existing, 60)

		byName := map[string]string{}
		for _, item := range env {
			byName[item.Name] = item.Value
		}
		assert.Equal(t, "https://app.example.com", byName[envSuperplaneURL])
		assert.Equal(t, factoryID, byName[envSuperplaneFactoryID])
		assert.Equal(t, orderID, byName[envSuperplaneOrderID])
		assert.NotEqual(t, "stale-token-from-old-task", byName[envSuperplaneToken])

		claims, err := jwt.NewSigner("test-session-secret-for-factory-runner").ValidateScopedToken(byName[envSuperplaneToken])
		require.NoError(t, err)
		assert.Equal(t, executionID.String(), claims.ExecutionID)
	})

	t.Run("skips token without created by", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-session-secret-for-factory-runner")

		env := appendFactoryRunnerEnvironment(core.ExecutionContext{
			ID:             executionID,
			OrganizationID: orgID,
			BaseURL:        "https://app.example.com",
			Factory: &stubFactoryContext{
				ok: true,
				link: &core.LinkedWorkOrder{
					ID:        orderID,
					FactoryID: factoryID,
				},
			},
		}, nil, 60)

		byName := map[string]string{}
		for _, item := range env {
			byName[item.Name] = item.Value
		}
		assert.Equal(t, factoryID, byName[envSuperplaneFactoryID])
		assert.Equal(t, orderID, byName[envSuperplaneOrderID])
		assert.Empty(t, byName[envSuperplaneToken])
	})

	t.Run("skips inject when organization id invalid", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-session-secret-for-factory-runner")

		env := appendFactoryRunnerEnvironment(core.ExecutionContext{
			ID:             executionID,
			OrganizationID: "not-a-uuid",
			Factory: &stubFactoryContext{
				ok: true,
				link: &core.LinkedWorkOrder{
					ID:              orderID,
					FactoryID:       factoryID,
					CreatedByUserID: userID,
				},
			},
		}, nil, 60)
		assert.Empty(t, env)
	})
}
