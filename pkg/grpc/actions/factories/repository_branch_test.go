package factories

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestChooseVerifiedDefaultBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested string
		saved     string
		exists    bool
		verifyErr error
		live      string
		liveErr   error
		want      string
		keepSaved bool
		wantErr   string
	}{
		{name: "empty request keeps the saved branch", requested: " ", saved: "develop", keepSaved: true},
		{name: "existing branch is saved", requested: "master", exists: true, want: "master"},
		{name: "missing branch uses the provider default", requested: "main", live: " master ", want: "master"},
		{name: "failed lookup keeps a saved branch", requested: "main", saved: "develop", verifyErr: errors.New("down"), keepSaved: true},
		{name: "failed lookup with nothing saved is rejected", requested: "main", verifyErr: errors.New("down"), wantErr: "could not verify the default branch"},
		{name: "failed provider default keeps a saved branch", requested: "main", saved: "develop", liveErr: errors.New("down"), keepSaved: true},
		{name: "failed provider default with nothing saved is rejected", requested: "main", liveErr: errors.New("down"), wantErr: "could not verify the default branch"},
		{name: "missing provider default keeps a saved branch", requested: "main", saved: "develop", keepSaved: true},
		{name: "missing provider default with nothing saved is rejected", requested: "main", wantErr: "default branch is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, keepSaved, err := chooseVerifiedDefaultBranch(test.requested, test.saved, test.exists, test.verifyErr, test.live, test.liveErr)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.keepSaved, keepSaved)
		})
	}
}

func TestKeepSavedDefaultBranch(t *testing.T) {
	t.Parallel()

	saved := models.FactoryOnboardingConfig{AppRepository: "acme/api", DefaultBranch: "develop"}
	assert.Equal(t, "master", keepSavedDefaultBranch(saved, "acme/api", "master"))
	assert.Equal(t, "develop", keepSavedDefaultBranch(saved, "acme/api", " "))
	assert.Equal(t, "", keepSavedDefaultBranch(saved, "acme/web", ""))
}

func TestDefaultBranchFromCacheOrProvider(t *testing.T) {
	t.Parallel()

	called := false
	assert.Equal(t, "trunk", defaultBranchFromCacheOrProvider(" trunk ", func() (string, error) {
		called = true
		return "master", nil
	}))
	assert.False(t, called)
	assert.Equal(t, "master", defaultBranchFromCacheOrProvider(" ", func() (string, error) {
		return " master ", nil
	}))
	assert.Equal(t, "", defaultBranchFromCacheOrProvider("", func() (string, error) {
		return "", errors.New("unavailable")
	}))
	assert.Equal(t, "", defaultBranchFromCacheOrProvider("", nil))
}
