package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
)

func TestFactoryLLMModelsSeparatesConfiguredAndEffectiveSelections(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = support.SaveProviderModelCatalog(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{"gpt-a", "gpt-b"})
	require.NoError(t, err)
	request := &pb.UpdateFactoryLLMModelsRequest{FactoryId: factory.ID.String(), Provider: "openai", FundingSource: "byok", AllowedModels: []string{"gpt-a"}, InheritParent: proto.Bool(false)}
	_, err = UpdateFactoryLLMModels(t.Context(), r.Organization.ID.String(), request)
	require.NoError(t, err)
	_, err = support.SaveProviderModelCatalog(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{"gpt-b"})
	require.NoError(t, err)
	list := &pb.ListFactoryLLMModelsRequest{FactoryId: factory.ID.String(), Provider: "openai", FundingSource: "byok"}
	response, err := ListFactoryLLMModels(t.Context(), r.Organization.ID.String(), list)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-b"}, codexLineModelIDs(response.Parent))
	assert.Equal(t, []string{"gpt-a"}, codexLineModelIDs(response.ConfiguredModels))
	assert.Empty(t, response.Selected)
	assert.False(t, response.InheritParent)

	request.AllowedModels = nil
	_, err = UpdateFactoryLLMModels(t.Context(), r.Organization.ID.String(), request)
	require.NoError(t, err)
	response, err = ListFactoryLLMModels(t.Context(), r.Organization.ID.String(), list)
	require.NoError(t, err)
	assert.Empty(t, response.Selected)
	assert.False(t, response.InheritParent)

	request.InheritParent = nil
	_, err = UpdateFactoryLLMModels(t.Context(), r.Organization.ID.String(), request)
	require.NoError(t, err)
	response, err = ListFactoryLLMModels(t.Context(), r.Organization.ID.String(), list)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-b"}, codexLineModelIDs(response.Selected))
	assert.True(t, response.InheritParent)
	request.InheritParent = proto.Bool(true)
	request.AllowedModels = []string{"gpt-b"}
	_, err = UpdateFactoryLLMModels(t.Context(), r.Organization.ID.String(), request)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestFactorySubscriptionPolicyCombinesConnectedMembersCatalogs(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other := support.CreateUser(t, r, r.Organization.ID)
	for _, member := range []struct {
		id     uuid.UUID
		models []models.CodexModel
	}{
		{id: r.User, models: []models.CodexModel{{Model: "gpt-native-a", DisplayName: "Readable native model"}, {Model: "gpt-native-shared"}}},
		{id: other.ID, models: []models.CodexModel{{Model: "gpt-native-b"}, {Model: "gpt-native-shared"}}},
	} {
		connection, err := models.CreateCodexConnection(db, r.Organization.ID, member.id, time.Now())
		require.NoError(t, err)
		require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
		require.NoError(t, connection.StoreUsage(db, nil, member.models, time.Now()))
	}
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	otherContext := authentication.SetUserIdInMetadata(t.Context(), other.ID.String())
	request := &pb.UpdateFactoryLLMModelsRequest{FactoryId: factory.ID.String(), Provider: "openai", FundingSource: "subscription", AllowedModels: []string{"gpt-native-b"}, InheritParent: proto.Bool(false)}
	_, err = UpdateFactoryLLMModels(otherContext, r.Organization.ID.String(), request)
	require.NoError(t, err)
	listed, err := ListFactoryLLMModels(ctx, r.Organization.ID.String(), &pb.ListFactoryLLMModelsRequest{FactoryId: factory.ID.String(), Provider: "openai", FundingSource: "subscription"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"gpt-native-a", "gpt-native-b", "gpt-native-shared"}, codexLineModelIDs(listed.Parent))
	assert.Equal(t, []string{"gpt-native-b"}, codexLineModelIDs(listed.ConfiguredModels))
	for _, model := range listed.Parent {
		if model.Id == "gpt-native-a" {
			assert.Equal(t, "Readable native model", model.Name)
		}
	}
	assert.Equal(t, "gpt-native-b", listed.ConfiguredModels[0].Name)
	assert.Equal(t, []string{"gpt-native-b"}, codexLineModelIDs(listed.Selected))

	request.AllowedModels = []string{"gpt-native-b", "gpt-native-a"}
	updated, err := UpdateFactoryLLMModels(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"gpt-native-a", "gpt-native-b"}, codexLineModelIDs(updated.Selected))
	request.AllowedModels = append(request.AllowedModels, "unknown-native-id")
	_, err = UpdateFactoryLLMModels(ctx, r.Organization.ID.String(), request)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	request.AllowedModels = []string{"gpt-native-a"}
	_, err = UpdateFactoryLLMModels(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	otherConnection, err := models.FindCodexConnection(db, r.Organization.ID, other.ID)
	require.NoError(t, err)
	require.NoError(t, otherConnection.Disconnect(db, time.Now()))
	request.AllowedModels = []string{"gpt-native-b"}
	_, err = UpdateFactoryLLMModels(ctx, r.Organization.ID.String(), request)
	require.Error(t, err, "a removed unavailable ID must be validated when added again")
	request.Provider = "anthropic"
	_, err = UpdateFactoryLLMModels(ctx, r.Organization.ID.String(), request)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestFactoryLLMModelsPreservesPersonalAndProviderAccessGates(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	subscription := &pb.ListFactoryLLMModelsRequest{FactoryId: factory.ID.String(), Provider: "openai", FundingSource: "subscription"}
	_, err = ListFactoryLLMModels(ctx, r.Organization.ID.String(), subscription)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	_, err = ListFactoryLLMModels(t.Context(), r.Organization.ID.String(), subscription)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))
	foreignCaller := authentication.SetUserIdInMetadata(t.Context(), uuid.NewString())
	_, err = ListFactoryLLMModels(foreignCaller, r.Organization.ID.String(), subscription)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	withoutConnection, err := ListFactoryLLMModels(ctx, r.Organization.ID.String(), subscription)
	require.NoError(t, err)
	assert.True(t, withoutConnection.InheritParent)
	assert.Empty(t, withoutConnection.Parent)
	_, err = UpdateFactoryLLMModels(ctx, r.Organization.ID.String(), &pb.UpdateFactoryLLMModelsRequest{FactoryId: factory.ID.String(), Provider: "custom", FundingSource: "byok", InheritParent: proto.Bool(false)})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
}
