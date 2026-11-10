package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	secretpb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func TestPersonalTokenSecurityEvents(t *testing.T) {
	r := support.Setup(t)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-user-id", r.User.String(), "x-account-id", r.Account.ID.String(), "x-organization-id", r.Organization.ID.String(),
	))
	ctx = context.WithValue(ctx, authorization.DomainIdContextKey, r.Organization.ID.String())
	service := NewMeService(r.AuthService, r.Encryptor)
	logs := support.CaptureSecurityLogs(t)
	response, err := service.CreateToken(ctx, &pb.CreateTokenRequest{Name: "security test"})
	require.NoError(t, err)
	require.NotEmpty(t, response.Plaintext)
	events := support.SecurityEvents(t, logs)
	require.Len(t, events, 1)
	assert.Equal(t, "personal_token.create", events[0]["action"])
	assert.Equal(t, "success", events[0]["outcome"])
	assert.Equal(t, r.Account.ID.String(), events[0]["actor_account_id"])
	assert.Equal(t, response.Token.Id, events[0]["target_id"])
	assert.NotContains(t, logs.String(), response.Plaintext)

	logs.Reset()
	_, err = service.RevokeToken(ctx, &pb.RevokeTokenRequest{Id: response.Token.Id})
	require.NoError(t, err)
	events = support.SecurityEvents(t, logs)
	require.Len(t, events, 1)
	assert.Equal(t, "personal_token.revoke", events[0]["action"])
	assert.Equal(t, response.Token.Id, events[0]["target_id"])
	assert.Equal(t, "success", events[0]["outcome"])

	logs.Reset()
	_, err = service.CreateToken(ctx, &pb.CreateTokenRequest{})
	require.Error(t, err)
	events = support.SecurityEvents(t, logs)
	require.Len(t, events, 1)
	assert.NotEqual(t, "success", events[0]["outcome"])
}

func TestSecretLifecycleSecurityEventsExcludeValues(t *testing.T) {
	r := support.Setup(t)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", r.User.String(), "x-account-id", r.Account.ID.String()))
	ctx = context.WithValue(ctx, authorization.DomainTypeContextKey, models.DomainTypeOrganization)
	ctx = context.WithValue(ctx, authorization.DomainIdContextKey, r.Organization.ID.String())
	ctx = context.WithValue(ctx, authorization.OrganizationContextKey, r.Organization.ID.String())
	service := NewSecretService(r.Encryptor, r.AuthService)
	logs := support.CaptureSecurityLogs(t)
	response, err := service.CreateSecret(ctx, &secretpb.CreateSecretRequest{Secret: &secretpb.Secret{
		Metadata: &secretpb.Secret_Metadata{Name: "security-test"},
		Spec:     &secretpb.Secret_Spec{Provider: secretpb.Secret_PROVIDER_LOCAL, Local: &secretpb.Secret_Local{Data: map[string]string{"password": "secret-value-sentinel"}}},
	}})
	require.NoError(t, err)
	events := support.SecurityEvents(t, logs)
	require.Len(t, events, 1)
	assert.Equal(t, "secret.create", events[0]["action"])
	assert.Equal(t, "success", events[0]["outcome"])
	assert.Equal(t, response.Secret.Metadata.Id, events[0]["target_id"])
	assert.Equal(t, r.Account.ID.String(), events[0]["actor_account_id"])
	assert.Equal(t, r.Organization.ID.String(), events[0]["organization_id"])
	assert.NotContains(t, logs.String(), "secret-value-sentinel")
	logs.Reset()
	_, err = service.DeleteSecret(ctx, &secretpb.DeleteSecretRequest{IdOrName: response.Secret.Metadata.Id})
	require.NoError(t, err)
	events = support.SecurityEvents(t, logs)
	require.Len(t, events, 1)
	assert.Equal(t, "secret.delete", events[0]["action"])
	assert.Equal(t, "success", events[0]["outcome"])
	assert.Equal(t, response.Secret.Metadata.Id, events[0]["target_id"])
}
