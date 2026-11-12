package github

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

const gitHubEnterpriseUnavailable = "GitHub Enterprise is not available"

type gitHubMetaResponse struct {
	InstalledVersion string `json:"installed_version"`
}

var (
	gitHubEnterprise             enterprise.GitHubEnterprise
	orgHasGitHubEnterprise       = githubEnterpriseEnabled
	orgHasGitHubEnterpriseServer = githubEnterpriseServerEnabled
	probeGitHubHost              = ProbeGitHubHost
)

// SetGitHubEnterprise installs the Enterprise capability that authorizes
// connections to a GitHub host other than github.com. A nil value is
// Community mode.
func SetGitHubEnterprise(impl enterprise.GitHubEnterprise) {
	gitHubEnterprise = impl
}

type transactionalDB interface {
	DB() *gorm.DB
}

func setupDB(httpCtx core.HTTPContext) *gorm.DB {
	if holder, ok := httpCtx.(transactionalDB); ok {
		if db := holder.DB(); db != nil {
			return db
		}
	}
	return database.Conn()
}

func githubEnterpriseEnabled(tx *gorm.DB, orgID uuid.UUID) bool {
	ok, err := models.OrganizationHasExperimentalFeatures(tx, orgID, features.FeatureGitHubEnterprise)
	return err == nil && ok
}

func githubEnterpriseEnabledForOrg(tx *gorm.DB, orgID string) bool {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return false
	}
	return orgHasGitHubEnterprise(tx, id)
}

func authorizeGitHubEnterpriseHost(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, host common.Host) error {
	if host.IsDotCom() {
		return nil
	}
	if host.IsGHES() {
		if !orgHasGitHubEnterpriseServer(tx, orgID) {
			return grpcerrors.FailedPrecondition(errors.New("GitHub Enterprise Server is not available"), "GitHub Enterprise Server is not available")
		}
	} else if !orgHasGitHubEnterprise(tx, orgID) {
		return grpcerrors.FailedPrecondition(errors.New(gitHubEnterpriseUnavailable), gitHubEnterpriseUnavailable)
	}
	impl := gitHubEnterprise
	if impl == nil {
		return grpcerrors.PermissionDenied(licensing.ErrNotLicensed, licensing.ErrNotLicensed.Error())
	}
	return impl.AuthorizeHost(ctx, orgID, host.WebHostname())
}

func authorizeGitHubEnterpriseOrg(ctx context.Context, tx *gorm.DB, orgID string, host common.Host) error {
	if host.IsDotCom() {
		return nil
	}
	id, err := uuid.Parse(orgID)
	if err != nil {
		if host.IsGHES() {
			return grpcerrors.FailedPrecondition(errors.New("GitHub Enterprise Server is not available"), "GitHub Enterprise Server is not available")
		}
		return grpcerrors.FailedPrecondition(errors.New(gitHubEnterpriseUnavailable), gitHubEnterpriseUnavailable)
	}
	return authorizeGitHubEnterpriseHost(ctx, tx, id, host)
}

func writeGitHubEnterpriseAuthorizationError(w http.ResponseWriter, err error) {
	message := grpcerrors.StatusMessage(err)
	if message == "" {
		message = err.Error()
	}
	status := http.StatusBadRequest
	if grpcerrors.Code(err) == codes.PermissionDenied {
		status = http.StatusForbidden
	}
	http.Error(w, message, status)
}

func gitHubEnterpriseLicensed(ctx context.Context, orgID uuid.UUID) bool {
	impl := gitHubEnterprise
	if impl == nil {
		return false
	}
	return impl.AuthorizeHost(ctx, orgID, "acme.ghe.com") == nil
}

func githubEnterpriseServerEnabled(tx *gorm.DB, orgID uuid.UUID) bool {
	ok, err := models.OrganizationHasExperimentalFeatures(tx, orgID, features.FeatureGitHubEnterpriseServer)
	return err == nil && ok
}

func withGitHubEnterpriseForTest(impl enterprise.GitHubEnterprise) func() {
	previous := gitHubEnterprise
	gitHubEnterprise = impl
	return func() { gitHubEnterprise = previous }
}

// WithGitHubEnterpriseForTest replaces the GitHub Enterprise capability.
func WithGitHubEnterpriseForTest(impl enterprise.GitHubEnterprise) func() {
	return withGitHubEnterpriseForTest(impl)
}

func withGitHubEnterpriseFlagForTest(fn func(uuid.UUID) bool) func() {
	previous := orgHasGitHubEnterprise
	orgHasGitHubEnterprise = func(_ *gorm.DB, orgID uuid.UUID) bool { return fn(orgID) }
	return func() { orgHasGitHubEnterprise = previous }
}

func withGitHubEnterpriseServerForTest(fn func(uuid.UUID) bool) func() {
	previous := orgHasGitHubEnterpriseServer
	orgHasGitHubEnterpriseServer = func(_ *gorm.DB, orgID uuid.UUID) bool { return fn(orgID) }
	return func() { orgHasGitHubEnterpriseServer = previous }
}

func withProbeGitHubHostForTest(fn func(core.HTTPContext, common.Host, uuid.UUID) error) func() {
	previous := probeGitHubHost
	probeGitHubHost = fn
	return func() { probeGitHubHost = previous }
}

// WithProbeGitHubHostForTest replaces the unauthenticated GET /meta probe.
func WithProbeGitHubHostForTest(fn func(core.HTTPContext, common.Host, uuid.UUID) error) func() {
	return withProbeGitHubHostForTest(fn)
}

func hostFromConfig(configMap map[string]any) (common.Host, error) {
	raw, _ := configMap[common.ConfigHost].(string)
	return common.ParseHost(raw)
}

func HostRequiresPrivateApp(configMap map[string]any) bool {
	host, err := hostFromConfig(configMap)
	if err != nil {
		return false
	}
	return !host.IsDotCom()
}

func hostAllowed(host common.Host) bool {
	if host.IsDotCom() {
		return true
	}
	allowed := config.GitHubEnterpriseAllowedHosts()
	if len(allowed) == 0 {
		return true
	}
	hostname := strings.ToLower(host.WebHostname())
	for _, entry := range allowed {
		if hostnameMatchesAllowlist(hostname, entry) {
			return true
		}
	}
	return false
}

func hostnameMatchesAllowlist(hostname, entry string) bool {
	entry = strings.ToLower(strings.TrimSpace(entry))
	if entry == "" {
		return false
	}
	if strings.HasPrefix(entry, "*.") {
		suffix := strings.TrimPrefix(entry, "*")
		return strings.HasSuffix(hostname, suffix) && hostname != strings.TrimPrefix(suffix, ".")
	}
	return hostname == entry
}

// ProbeGitHubHost calls GET {api}/meta without authentication.
func ProbeGitHubHost(httpCtx core.HTTPContext, host common.Host, orgID uuid.UUID) error {
	req, err := http.NewRequest(http.MethodGet, host.APIURL()+"meta", nil)
	if err != nil {
		return fmt.Errorf("failed to build GitHub meta request: %w", err)
	}

	res, err := httpCtx.Do(req)
	if err != nil {
		return classifyGitHubHostProbeError(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read GitHub meta response: %w", err)
	}

	// ghe.com tenants hide /meta from anonymous requests and answer 404 with GitHub headers.
	if (res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusNotFound) &&
		res.Header.Get("X-GitHub-Enterprise-Version") == "ghe.com" {
		return nil
	}

	switch res.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return fmt.Errorf("this host is not a GitHub API")
	case http.StatusForbidden:
		return fmt.Errorf("GitHub refused the request. Check the IP allow list")
	default:
		return fmt.Errorf("GitHub meta request failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var meta gitHubMetaResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return fmt.Errorf("this host is not a GitHub API")
	}
	if strings.TrimSpace(meta.InstalledVersion) == "" {
		return nil
	}
	if !orgHasGitHubEnterpriseServer(setupDB(httpCtx), orgID) {
		return fmt.Errorf("GitHub Enterprise Server is not available")
	}
	return nil
}

func classifyGitHubHostProbeError(err error) error {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return fmt.Errorf("could not resolve GitHub host")
	}
	var tlsError *tls.CertificateVerificationError
	if errors.As(err, &tlsError) {
		return fmt.Errorf("could not verify the TLS certificate")
	}
	if strings.Contains(strings.ToLower(err.Error()), "x509") || strings.Contains(strings.ToLower(err.Error()), "certificate") {
		return fmt.Errorf("could not verify the TLS certificate")
	}
	return fmt.Errorf("could not reach GitHub host: %w", err)
}
