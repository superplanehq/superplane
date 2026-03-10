package azure

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockAssertion(token string) func(context.Context) (string, error) {
	return func(_ context.Context) (string, error) {
		return token, nil
	}
}

func failingAssertion(msg string) func(context.Context) (string, error) {
	return func(_ context.Context) (string, error) {
		return "", fmt.Errorf("%s", msg)
	}
}

func TestNewAzureProvider(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("success with valid configuration", func(t *testing.T) {
		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"test-subscription-id",
			mockAssertion("mock-oidc-token-content"),
			logger,
		)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.credential)
		assert.NotNil(t, provider.client)
		assert.Equal(t, "test-subscription-id", provider.GetSubscriptionID())
	})

	t.Run("error when tenant ID is empty", func(t *testing.T) {
		provider, err := NewAzureProvider(
			context.Background(),
			"",
			"test-client-id",
			"test-subscription-id",
			mockAssertion("token"),
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "tenant ID is required")
	})

	t.Run("error when client ID is empty", func(t *testing.T) {
		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"",
			"test-subscription-id",
			mockAssertion("token"),
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "client ID is required")
	})

	t.Run("error when subscription ID is empty", func(t *testing.T) {
		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"",
			mockAssertion("token"),
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "subscription ID is required")
	})

	t.Run("error when assertion callback fails", func(t *testing.T) {
		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"test-subscription-id",
			failingAssertion("OIDC signing failed"),
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "failed to obtain OIDC assertion")
	})
}

func TestAzureProvider_GetCredential(t *testing.T) {
	provider, err := NewAzureProvider(
		context.Background(),
		"test-tenant-id",
		"test-client-id",
		"test-subscription-id",
		mockAssertion("mock-token"),
		logrus.NewEntry(logrus.New()),
	)
	require.NoError(t, err)

	credential := provider.GetCredential()
	assert.NotNil(t, credential)
}

func TestAzureProvider_GetClient(t *testing.T) {
	provider, err := NewAzureProvider(
		context.Background(),
		"test-tenant-id",
		"test-client-id",
		"test-subscription-id",
		mockAssertion("mock-token"),
		logrus.NewEntry(logrus.New()),
	)
	require.NoError(t, err)

	client := provider.getClient()
	assert.NotNil(t, client)
}

func TestAzureProvider_GetSubscriptionID(t *testing.T) {
	expectedSubscriptionID := "test-subscription-123"
	provider, err := NewAzureProvider(
		context.Background(),
		"test-tenant-id",
		"test-client-id",
		expectedSubscriptionID,
		mockAssertion("mock-token"),
		logrus.NewEntry(logrus.New()),
	)
	require.NoError(t, err)

	subscriptionID := provider.GetSubscriptionID()
	assert.Equal(t, expectedSubscriptionID, subscriptionID)
}
