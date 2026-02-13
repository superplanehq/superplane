package azure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureProvider(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("success with valid configuration", func(t *testing.T) {
		// Create temporary OIDC token file
		tmpDir := t.TempDir()
		tokenFile := filepath.Join(tmpDir, "token")
		tokenContent := "mock-oidc-token-content"
		err := os.WriteFile(tokenFile, []byte(tokenContent), 0600)
		require.NoError(t, err)

		// Set environment variable
		t.Setenv(AzureFederatedTokenFileEnv, tokenFile)

		// Create provider
		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"test-subscription-id",
			logger,
		)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.credential)
		assert.NotNil(t, provider.computeClient)
		assert.Equal(t, "test-subscription-id", provider.GetSubscriptionID())
	})

	t.Run("error when tenant ID is empty", func(t *testing.T) {
		provider, err := NewAzureProvider(
			context.Background(),
			"",
			"test-client-id",
			"test-subscription-id",
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
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "subscription ID is required")
	})

	t.Run("error when token file environment variable is not set", func(t *testing.T) {
		// Ensure environment variable is not set
		os.Unsetenv(AzureFederatedTokenFileEnv)

		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"test-subscription-id",
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "environment variable")
		assert.Contains(t, err.Error(), "is not set")
	})

	t.Run("error when token file does not exist", func(t *testing.T) {
		// Set environment variable to non-existent file
		t.Setenv(AzureFederatedTokenFileEnv, "/non/existent/path/token")

		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"test-subscription-id",
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("error when token file is empty", func(t *testing.T) {
		// Create empty token file
		tmpDir := t.TempDir()
		tokenFile := filepath.Join(tmpDir, "empty-token")
		err := os.WriteFile(tokenFile, []byte(""), 0600)
		require.NoError(t, err)

		t.Setenv(AzureFederatedTokenFileEnv, tokenFile)

		provider, err := NewAzureProvider(
			context.Background(),
			"test-tenant-id",
			"test-client-id",
			"test-subscription-id",
			logger,
		)

		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

func TestAzureProvider_GetCredential(t *testing.T) {
	// Create temporary OIDC token file
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	err := os.WriteFile(tokenFile, []byte("mock-token"), 0600)
	require.NoError(t, err)

	t.Setenv(AzureFederatedTokenFileEnv, tokenFile)

	provider, err := NewAzureProvider(
		context.Background(),
		"test-tenant-id",
		"test-client-id",
		"test-subscription-id",
		logrus.NewEntry(logrus.New()),
	)
	require.NoError(t, err)

	credential := provider.GetCredential()
	assert.NotNil(t, credential)
}

func TestAzureProvider_GetComputeClient(t *testing.T) {
	// Create temporary OIDC token file
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	err := os.WriteFile(tokenFile, []byte("mock-token"), 0600)
	require.NoError(t, err)

	t.Setenv(AzureFederatedTokenFileEnv, tokenFile)

	provider, err := NewAzureProvider(
		context.Background(),
		"test-tenant-id",
		"test-client-id",
		"test-subscription-id",
		logrus.NewEntry(logrus.New()),
	)
	require.NoError(t, err)

	computeClient := provider.GetComputeClient()
	assert.NotNil(t, computeClient)
}

func TestAzureProvider_GetSubscriptionID(t *testing.T) {
	// Create temporary OIDC token file
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	err := os.WriteFile(tokenFile, []byte("mock-token"), 0600)
	require.NoError(t, err)

	t.Setenv(AzureFederatedTokenFileEnv, tokenFile)

	expectedSubscriptionID := "test-subscription-123"
	provider, err := NewAzureProvider(
		context.Background(),
		"test-tenant-id",
		"test-client-id",
		expectedSubscriptionID,
		logrus.NewEntry(logrus.New()),
	)
	require.NoError(t, err)

	subscriptionID := provider.GetSubscriptionID()
	assert.Equal(t, expectedSubscriptionID, subscriptionID)
}
