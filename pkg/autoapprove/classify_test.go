package autoapprove

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		change   Change
		wantCat  Category
		wantTier Tier
		wantInrt bool
	}{
		{
			name:     "unreadable change fails closed to high",
			change:   Change{Known: false},
			wantCat:  CategoryUnknown,
			wantTier: TierHigh,
			wantInrt: false,
		},
		{
			name:     "known but no paths fails closed to high",
			change:   Change{Known: true, Paths: nil},
			wantCat:  CategoryUnknown,
			wantTier: TierHigh,
			wantInrt: false,
		},
		{
			name:     "docs only is inert low",
			change:   Change{Known: true, Paths: []string{"README.md", "docs/guide.md"}},
			wantCat:  CategoryDocs,
			wantTier: TierLow,
			wantInrt: true,
		},
		{
			name:     "a migration hidden among docs is still high",
			change:   Change{Known: true, Paths: []string{"README.md", "db/migrations/0007_add_column.sql"}},
			wantCat:  CategoryMigration,
			wantTier: TierHigh,
			wantInrt: false,
		},
		{
			name:     "auth path forces high",
			change:   Change{Known: true, Paths: []string{"pkg/auth/session.go"}},
			wantCat:  CategoryAuth,
			wantTier: TierHigh,
			wantInrt: false,
		},
		{
			name:     "secrets file forces high",
			change:   Change{Known: true, Paths: []string{"config/.env.production"}},
			wantCat:  CategorySecrets,
			wantTier: TierHigh,
			wantInrt: false,
		},
		{
			name:     "infra file forces high",
			change:   Change{Known: true, Paths: []string{"deploy/Dockerfile"}},
			wantCat:  CategoryInfra,
			wantTier: TierHigh,
			wantInrt: false,
		},
		{
			name:     "tests only is low but not inert",
			change:   Change{Known: true, Paths: []string{"pkg/x/x_test.go"}},
			wantCat:  CategoryTests,
			wantTier: TierLow,
			wantInrt: false,
		},
		{
			name:     "dependency manifest is mid",
			change:   Change{Known: true, Paths: []string{"go.mod", "go.sum"}},
			wantCat:  CategoryDependency,
			wantTier: TierMid,
			wantInrt: false,
		},
		{
			name:     "plain app code is mid",
			change:   Change{Known: true, Paths: []string{"pkg/service/handler.go"}},
			wantCat:  CategoryAppCode,
			wantTier: TierMid,
			wantInrt: false,
		},
		{
			name:     "mixed docs and app code is app code, not inert",
			change:   Change{Known: true, Paths: []string{"README.md", "pkg/service/handler.go"}},
			wantCat:  CategoryAppCode,
			wantTier: TierMid,
			wantInrt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.change)
			if got.Category != tt.wantCat {
				t.Errorf("category = %q, want %q", got.Category, tt.wantCat)
			}
			if got.Tier != tt.wantTier {
				t.Errorf("tier = %q, want %q", got.Tier, tt.wantTier)
			}
			if got.Inert != tt.wantInrt {
				t.Errorf("inert = %v, want %v", got.Inert, tt.wantInrt)
			}
		})
	}
}

// A dangerous change must never be classified below High regardless of how many
// safe files surround it. This is the property the whole ceiling rests on.
func TestClassify_FloorsCannotBeDilutedBySafeFiles(t *testing.T) {
	safe := []string{"README.md", "docs/a.md", "docs/b.md", "x_test.go"}
	dangerous := []string{
		"db/migrations/9001.sql",
		"internal/authorization/policy.go",
		"secrets/token.pem",
		"infra/terraform/main.tf",
	}
	for _, d := range dangerous {
		paths := append(append([]string{}, safe...), d)
		got := Classify(Change{Known: true, Paths: paths})
		if got.Tier != TierHigh {
			t.Errorf("paths with %q classified as %q, want high", d, got.Tier)
		}
		if got.Inert {
			t.Errorf("paths with %q marked inert; dangerous changes are never inert", d)
		}
	}
}

// The bypasses found in review: a dangerous file with a doc-ish extension, a
// private key under docs/, or a danger path in the "wrong" case or with an
// unusual separator must all classify High and never auto-approve.
func TestClassify_DangerousNamesAndCasingCannotAutoApprove(t *testing.T) {
	dangerous := []string{
		"secrets.txt", "passwords.txt", "api_keys.txt", "tokens.rst", ".htpasswd",
		"docs/id_rsa", "docs/id_ed25519", "config/service-account.json.pem",
		"Migrations/0001.sql", "DB/migrate/0001.sql", "migrations\\0001.sql", "MIGRATIONS/x.sql",
		"Dockerfile", "prod.Dockerfile", "infra/main.tf", "service.tfstate",
		".github/workflows/ci.yml", "k8s/deploy.yaml",
		"internal/OAuth/login.go", "pkg/RBAC/policy.go",
	}
	for _, p := range dangerous {
		c := Classify(Change{Known: true, Paths: []string{p}})
		if c.Tier != TierHigh {
			t.Errorf("%q classified %s, want high", p, c.Tier)
		}
		if c.Inert {
			t.Errorf("%q marked inert; dangerous paths are never inert", p)
		}
		if Decide(c, Policy{InertChanges: true}, true, true).AutoApprove {
			t.Errorf("%q was auto-approved; it must require a human", p)
		}
	}
}
