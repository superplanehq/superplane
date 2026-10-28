// Package githubenterprise implements the Enterprise GitHub Enterprise
// connection check. It is licensed under the SuperPlane Enterprise license
// in ee/LICENSE.
package githubenterprise

import (
	"github.com/superplanehq/superplane/pkg/enterprise"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
)

// Service checks the installation license before a caller connects an
// organization to a GitHub host other than github.com.
type Service struct {
	entitlements licensing.Entitlements
}

var _ enterprise.GitHubEnterprise = (*Service)(nil)

func NewService(entitlements licensing.Entitlements) *Service {
	return &Service{entitlements: entitlements}
}

func (s *Service) AuthorizeHost(string) error {
	if err := licensing.Require(s.entitlements, licensing.FeatureGitHubEnterprise); err != nil {
		return grpcerrors.PermissionDenied(err, err.Error())
	}

	return nil
}
