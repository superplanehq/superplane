package autoapprove

import (
	"path"
	"slices"
	"strings"
)

// Tier is the risk level of a change. Only Low is ever eligible for automatic
// approval; Mid and High always require a human, and no configuration can lower
// that. See Decide for the ceiling.
type Tier string

const (
	TierLow  Tier = "low"
	TierMid  Tier = "mid"
	TierHigh Tier = "high"
)

// Category is the class of change, used for reporting and for deciding
// eligibility. It is descriptive; the Tier is what gates automation.
type Category string

const (
	CategoryDocs       Category = "docs"
	CategoryTests      Category = "tests"
	CategoryConfig     Category = "config"
	CategoryDependency Category = "dependency"
	CategoryAppCode    Category = "app_code"
	CategoryMigration  Category = "migration"
	CategoryAuth       Category = "auth"
	CategorySecrets    Category = "secrets"
	CategoryInfra      Category = "infra"
	CategoryUnknown    Category = "unknown"
)

// Classification is the result of inspecting a change. Reason explains the call
// in plain terms so it can be shown to a human and stored in the audit trail.
type Classification struct {
	Category Category
	Tier     Tier
	Inert    bool
	Floors   []Category
	Reason   string
}

// Classify inspects a change and assigns a category and tier. It is deterministic
// and fail-closed: an unreadable or ambiguous change is High, never Low. Danger
// floors are checked first and win over everything. Over-matching a floor is
// acceptable because it only ever sends more work to a human, which is the safe
// direction.
func Classify(c Change) Classification {
	if !c.Known || len(c.Paths) == 0 {
		return Classification{
			Category: CategoryUnknown,
			Tier:     TierHigh,
			Inert:    false,
			Reason:   "change could not be read from the payload; treated as high risk",
		}
	}

	// Normalize every path once so matching is case-insensitive and independent
	// of separators, leading "./", or redundant segments.
	paths := make([]string, len(c.Paths))
	for i, p := range c.Paths {
		paths[i] = normalize(p)
	}

	// Floors first. Any dangerous path forces High and cannot be lowered.
	if floors := detectFloors(paths); len(floors) > 0 {
		return Classification{
			Category: floors[0],
			Tier:     TierHigh,
			Inert:    false,
			Floors:   floors,
			Reason:   "touches " + joinCategories(floors) + "; locked to human review",
		}
	}

	// No floors: classify by what the change is made of.
	switch {
	case allMatch(paths, isDocPath):
		return Classification{Category: CategoryDocs, Tier: TierLow, Inert: true, Reason: "documentation only"}
	case allMatch(paths, isTestPath):
		return Classification{Category: CategoryTests, Tier: TierLow, Inert: false, Reason: "tests only"}
	case anyMatch(paths, isDependencyManifest):
		return Classification{Category: CategoryDependency, Tier: TierMid, Inert: false, Reason: "changes a dependency manifest"}
	case allMatch(paths, isConfigPath):
		return Classification{Category: CategoryConfig, Tier: TierLow, Inert: false, Reason: "non-production configuration only"}
	default:
		return Classification{Category: CategoryAppCode, Tier: TierMid, Inert: false, Reason: "application code"}
	}
}

// normalize lowercases a path and collapses separator and prefix variations so a
// dangerous path cannot dodge a matcher by casing or formatting.
func normalize(p string) string {
	p = strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimPrefix(p, "./")
	return p
}

// detectFloors returns the danger categories a change touches. Input paths are
// already normalized (lowercase, forward slashes).
func detectFloors(paths []string) []Category {
	checks := []struct {
		category Category
		match    func(p, base string) bool
	}{
		{CategoryMigration, isMigrationPath},
		{CategoryAuth, isAuthPath},
		{CategorySecrets, isSecretPath},
		{CategoryInfra, isInfraPath},
	}

	var found []Category
	seen := map[Category]bool{}
	for _, p := range paths {
		base := path.Base(p)
		for _, c := range checks {
			if !seen[c.category] && c.match(p, base) {
				found = append(found, c.category)
				seen[c.category] = true
			}
		}
	}
	return found
}

func isMigrationPath(p, base string) bool {
	if containsSegment(p, "migrations") || strings.HasPrefix(p, "migrations/") ||
		strings.Contains(p, "db/migrate") || strings.Contains(p, "alembic/versions") ||
		strings.Contains(p, "flyway") {
		return true
	}
	// A SQL file living under any migrate-ish directory.
	return strings.HasSuffix(base, ".sql") && (strings.Contains(p, "migrat") || strings.Contains(p, "schema"))
}

func isAuthPath(p, base string) bool {
	if strings.Contains(p, "permission") {
		return true
	}
	for _, kw := range []string{"auth", "authz", "authn", "authentication", "authorization", "rbac", "oauth", "iam"} {
		if containsSegment(p, kw) || strings.Contains(base, kw) {
			return true
		}
	}
	return false
}

func isSecretPath(p, base string) bool {
	if containsSegment(p, "secrets") || containsSegment(p, "credentials") {
		return true
	}
	if strings.HasPrefix(base, ".env") || base == ".htpasswd" || strings.Contains(base, "htpasswd") {
		return true
	}
	if extIn(base, ".pem", ".key", ".ppk", ".p12", ".pfx", ".keystore", ".jks", ".asc", ".gpg") {
		return true
	}
	for _, kw := range []string{"secret", "password", "passwd", "credential", "apikey", "api_key", "token", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "private_key", "privatekey"} {
		if strings.Contains(base, kw) {
			return true
		}
	}
	return false
}

func isInfraPath(p, base string) bool {
	if strings.Contains(base, "dockerfile") {
		return true
	}
	if extIn(base, ".tf", ".tfvars", ".tfstate") {
		return true
	}
	if containsSegment(p, "terraform") || containsSegment(p, "k8s") || containsSegment(p, "kubernetes") || containsSegment(p, "helm") {
		return true
	}
	if strings.Contains(p, ".github/workflows/") || strings.Contains(p, ".gitlab-ci") {
		return true
	}
	return strings.Contains(p, "deploy")
}

func isDocPath(p string) bool {
	if extIn(path.Base(p), ".md", ".mdx", ".rst", ".txt", ".adoc") {
		return true
	}
	return containsSegment(p, "docs") || path.Base(p) == "license"
}

func isTestPath(p string) bool {
	base := path.Base(p)
	return strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.tsx") || strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.tsx") || containsSegment(p, "__tests__") ||
		containsSegment(p, "testdata")
}

func isDependencyManifest(p string) bool {
	switch path.Base(p) {
	case "go.mod", "go.sum", "package.json", "package-lock.json", "yarn.lock",
		"pnpm-lock.yaml", "gemfile", "gemfile.lock", "requirements.txt", "poetry.lock", "cargo.toml", "cargo.lock":
		return true
	}
	return false
}

func isConfigPath(p string) bool {
	return extIn(path.Base(p), ".yaml", ".yml", ".json", ".toml", ".ini")
}

func extIn(base string, exts ...string) bool {
	ext := path.Ext(base)
	return slices.Contains(exts, ext)
}

func allMatch(paths []string, pred func(string) bool) bool {
	for _, p := range paths {
		if !pred(p) {
			return false
		}
	}
	return len(paths) > 0
}

func anyMatch(paths []string, pred func(string) bool) bool {
	for _, p := range paths {
		if pred(p) {
			return true
		}
	}
	return false
}

func containsSegment(p, segment string) bool {
	return slices.Contains(strings.Split(p, "/"), segment)
}

func joinCategories(cats []Category) string {
	parts := make([]string, len(cats))
	for i, c := range cats {
		parts[i] = string(c)
	}
	return strings.Join(parts, ", ")
}
