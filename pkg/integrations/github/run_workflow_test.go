package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

func TestRunWorkflow__ValidateBranchRestriction__DefaultBehavior_NoEnforcement(t *testing.T) {
	r := &RunWorkflow{}

	// When EnforceBranchRestriction is not set (nil), all branches should be allowed
	// This is opt-in security to avoid breaking existing workflows
	tests := []struct {
		name string
		ref  string
	}{
		{"main branch", "main"},
		{"develop branch", "develop"},
		{"feature branch", "feature/my-feature"},
		{"random branch", "my-random-branch"},
		{"commit SHA", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref: tt.ref,
				// No EnforceBranchRestriction set - should not enforce
			}

			err := r.validateBranchRestriction(spec)
			assert.NoError(t, err, "expected all refs to be allowed when enforcement is not enabled")
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__EnforcedWithDefaults(t *testing.T) {
	r := &RunWorkflow{}

	tests := []struct {
		name        string
		ref         string
		shouldAllow bool
	}{
		// Default allowed branches
		{"main branch", "main", true},
		{"master branch", "master", true},
		{"release branch", "release", true},
		{"production branch", "production", true},
		{"staging branch", "staging", true},
		{"refs/heads/main", "refs/heads/main", true},
		{"refs/heads/master", "refs/heads/master", true},

		// Not in default list
		{"develop branch", "develop", false},
		{"feature branch", "feature/my-feature", false},
		{"random branch", "my-random-branch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref:                      tt.ref,
				EnforceBranchRestriction: boolPtr(true), // Explicitly enable enforcement
			}

			err := r.validateBranchRestriction(spec)
			if tt.shouldAllow {
				assert.NoError(t, err, "expected ref to be allowed: %s", tt.ref)
			} else {
				assert.Error(t, err, "expected ref to be blocked: %s", tt.ref)
				assert.Contains(t, err.Error(), "not in the allowed branches list")
			}
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__CustomAllowedBranches(t *testing.T) {
	r := &RunWorkflow{}

	spec := RunWorkflowSpec{
		Ref:                      "develop",
		AllowedBranches:          []string{"develop", "feature/*"},
		EnforceBranchRestriction: boolPtr(true), // Must enable enforcement
	}

	// develop should be allowed
	err := r.validateBranchRestriction(spec)
	assert.NoError(t, err)

	// feature/x should be allowed via wildcard
	spec.Ref = "feature/my-feature"
	err = r.validateBranchRestriction(spec)
	assert.NoError(t, err)

	// main is NOT in custom list, should be blocked
	spec.Ref = "main"
	err = r.validateBranchRestriction(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in the allowed branches list")
}

func TestRunWorkflow__ValidateBranchRestriction__WildcardPatterns(t *testing.T) {
	r := &RunWorkflow{}

	tests := []struct {
		name           string
		allowedPattern string
		ref            string
		shouldAllow    bool
	}{
		{"release/* matches release/1.0", "release/*", "release/1.0", true},
		{"release/* matches release/v2.0.0", "release/*", "release/v2.0.0", true},
		{"release/* does not match releaseX", "release/*", "releaseX", false},
		{"hotfix/* matches hotfix/urgent-fix", "hotfix/*", "hotfix/urgent-fix", true},
		{"refs/heads/release/* match", "release/*", "refs/heads/release/1.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref:                      tt.ref,
				AllowedBranches:          []string{tt.allowedPattern},
				EnforceBranchRestriction: boolPtr(true), // Must enable enforcement
			}

			err := r.validateBranchRestriction(spec)
			if tt.shouldAllow {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__BlocksCommitSHAs(t *testing.T) {
	r := &RunWorkflow{}

	tests := []struct {
		name        string
		ref         string
		shouldBlock bool
	}{
		{"40-char hex SHA", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", true},
		{"40-char uppercase SHA", "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2", true},
		{"39-char string", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b", false},
		{"41-char string", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c", false},
		{"branch named like SHA but not hex", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6xxxx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref:                      tt.ref,
				AllowedBranches:          []string{tt.ref}, // Add to allowed list to isolate SHA check
				EnforceBranchRestriction: boolPtr(true),    // Must enable enforcement
			}

			err := r.validateBranchRestriction(spec)
			if tt.shouldBlock {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "commit SHA")
			} else {
				// May still fail for other reasons, just check it's not the SHA error
				if err != nil {
					assert.NotContains(t, err.Error(), "commit SHA")
				}
			}
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__BlocksPullRequestRefs(t *testing.T) {
	r := &RunWorkflow{}

	tests := []struct {
		name string
		ref  string
	}{
		{"refs/pull/123/head", "refs/pull/123/head"},
		{"refs/pull/456/merge", "refs/pull/456/merge"},
		{"pull/123/head", "pull/123/head"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref:                      tt.ref,
				AllowedBranches:          []string{tt.ref}, // Add to allowed list to isolate PR check
				EnforceBranchRestriction: boolPtr(true),    // Must enable enforcement
			}

			err := r.validateBranchRestriction(spec)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "pull request references are not allowed")
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__EnforceBranchRestrictionFalse(t *testing.T) {
	r := &RunWorkflow{}

	// When EnforceBranchRestriction is explicitly false, all branches should be allowed
	tests := []struct {
		name string
		ref  string
	}{
		{"random branch", "my-random-branch"},
		{"feature branch", "feature/experimental"},
		{"commit SHA", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"},
		{"PR reference", "refs/pull/123/head"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref:                      tt.ref,
				EnforceBranchRestriction: boolPtr(false),
			}

			err := r.validateBranchRestriction(spec)
			assert.NoError(t, err, "expected all refs to be allowed when EnforceBranchRestriction is false")
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__EnforceBranchRestrictionTrue(t *testing.T) {
	r := &RunWorkflow{}

	// When EnforceBranchRestriction is explicitly true, should use defaults
	spec := RunWorkflowSpec{
		Ref:                      "main",
		EnforceBranchRestriction: boolPtr(true),
	}

	err := r.validateBranchRestriction(spec)
	assert.NoError(t, err, "main should be allowed with defaults")

	// develop is not in defaults
	spec.Ref = "develop"
	err = r.validateBranchRestriction(spec)
	assert.Error(t, err, "develop should not be allowed with defaults")
}

func TestRunWorkflow__ValidateBranchRestriction__EnforceBranchRestrictionNil(t *testing.T) {
	r := &RunWorkflow{}

	// When EnforceBranchRestriction is nil (not set), should NOT enforce (opt-in security)
	// All branches should be allowed
	tests := []struct {
		name string
		ref  string
	}{
		{"main branch", "main"},
		{"develop branch", "develop"},
		{"feature branch", "feature/my-feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := RunWorkflowSpec{
				Ref:                      tt.ref,
				EnforceBranchRestriction: nil,
			}

			err := r.validateBranchRestriction(spec)
			assert.NoError(t, err, "all branches should be allowed when enforcement is nil (opt-in)")
		})
	}
}

func TestRunWorkflow__ValidateBranchRestriction__CustomBranchesWithEnforceTrue(t *testing.T) {
	r := &RunWorkflow{}

	spec := RunWorkflowSpec{
		Ref:                      "develop",
		AllowedBranches:          []string{"develop", "staging"},
		EnforceBranchRestriction: boolPtr(true),
	}

	// develop should be allowed (in custom list)
	err := r.validateBranchRestriction(spec)
	assert.NoError(t, err)

	// main should NOT be allowed (not in custom list)
	spec.Ref = "main"
	err = r.validateBranchRestriction(spec)
	assert.Error(t, err)
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0123456789abcdef", true},
		{"ABCDEF", true},
		{"a1b2c3", true},
		{"ghijkl", false},
		{"a1b2c3g", false},
		{"", true}, // empty string has no non-hex chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isHexString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
