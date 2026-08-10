package runner

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	envSuperplaneURL       = "SUPERPLANE_URL"
	envSuperplaneToken     = "SUPERPLANE_TOKEN"
	envSuperplaneFactoryID = "SUPERPLANE_FACTORY_ID"
	envSuperplaneOrderID   = "SUPERPLANE_ORDER_ID"

	factoryRunnerTokenSkew = 5 * time.Minute
)

// factoryRunnerMaxTTL matches the longest allowed runner wall-clock plus skew.
// Execution-active checks remain the primary kill switch; TTL is a backstop.
var factoryRunnerMaxTTL = time.Duration(maxExecutionTimeoutSecondsRequest)*time.Second + factoryRunnerTokenSkew

// appendFactoryRunnerEnvironment injects CLI auth + work-order identity when
// the current canvas run is linked to a factory work order and the task runs
// in Host mode (fleet user-data installs the CLI). Docker mode is skipped for
// now — use setup commands if a custom image needs the CLI.
//
// For factory-linked Host runs, factory/order IDs and SUPERPLANE_TOKEN are
// always overwritten from the linked order + a freshly minted JWT so node
// config or a previous task cannot sticky-reuse credentials. The JWT is bound
// to ctx.ID (node execution) and claims.OrgID so it dies when the execution
// finishes and cannot authenticate against another org.
func appendFactoryRunnerEnvironment(
	ctx core.ExecutionContext,
	env []BrokerEnvironmentVariable,
	timeoutSeconds int,
	executionMode string,
) []BrokerEnvironmentVariable {
	if normalizeExecutionMode(executionMode) != ExecutionModeHost {
		return env
	}
	if ctx.Factory == nil {
		return env
	}

	link, ok, err := ctx.Factory.LinkedWorkOrder()
	if err != nil {
		warnFactoryRunnerEnv(ctx, err, "resolve linked work order")
		return env
	}
	if !ok || link == nil {
		return env
	}

	orgID := strings.TrimSpace(ctx.OrganizationID)
	if _, err := uuid.Parse(orgID); err != nil {
		warnFactoryRunnerEnv(ctx, err, "organization id for factory runner token")
		return env
	}
	if ctx.ID == uuid.Nil {
		warnFactoryRunnerEnv(ctx, errExecutionIDRequired, "node execution id for factory runner token")
		return env
	}

	// Always pin identity to the linked work order (no sticky override).
	env = upsertEnv(env, envSuperplaneFactoryID, link.FactoryID)
	env = upsertEnv(env, envSuperplaneOrderID, link.ID)
	// Drop any prior token before mint so a failed mint cannot leave a
	// stale credential paired with the newly pinned order IDs.
	env = stripEnv(env, envSuperplaneToken)

	baseURL := strings.TrimRight(strings.TrimSpace(ctx.BaseURL), "/")
	if baseURL == "" {
		baseURL = config.BaseURL()
	}
	if baseURL != "" {
		env = upsertEnv(env, envSuperplaneURL, baseURL)
	}

	if link.CreatedByUserID == "" {
		warnFactoryRunnerEnv(ctx, errCreatedByRequired, "mint scoped token")
		return env
	}

	token, err := mintFactoryRunnerToken(orgID, link.CreatedByUserID, link.ID, ctx.ID.String(), timeoutSeconds)
	if err != nil {
		warnFactoryRunnerEnv(ctx, err, "mint scoped token")
		return env
	}

	return upsertEnv(env, envSuperplaneToken, token)
}

func mintFactoryRunnerToken(orgID, userID, orderID, executionID string, timeoutSeconds int) (string, error) {
	secret, err := config.JWTSecret()
	if err != nil {
		return "", errJWTSecretMissing
	}
	if _, err := uuid.Parse(orgID); err != nil {
		return "", fmtInvalidOrgID(err)
	}
	if _, err := uuid.Parse(userID); err != nil {
		return "", fmtInvalidUserID(err)
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return "", fmtInvalidOrderID(err)
	}
	if _, err := uuid.Parse(executionID); err != nil {
		return "", fmtInvalidExecutionID(err)
	}

	ttl := time.Duration(timeoutSeconds) * time.Second
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds) * time.Second
	}
	ttl += factoryRunnerTokenSkew
	if ttl > factoryRunnerMaxTTL {
		ttl = factoryRunnerMaxTTL
	}

	signer := jwt.NewSigner(secret)
	return signer.GenerateScopedToken(jwt.ScopedTokenClaims{
		Subject:     userID,
		OrgID:       orgID,
		Purpose:     jwt.PurposeFactoryRunner,
		ExecutionID: executionID,
		Scopes: []string{
			"work_orders:read:" + orderID,
			"work_orders:update:" + orderID,
		},
	}, ttl)
}

var (
	errJWTSecretMissing    = errors.New("JWT_SECRET is not configured")
	errExecutionIDRequired = errors.New("execution id is required")
	errCreatedByRequired   = errors.New("work order has no created-by user")
)

func fmtInvalidOrgID(err error) error {
	return errors.Join(errors.New("invalid organization id"), err)
}

func fmtInvalidUserID(err error) error {
	return errors.Join(errors.New("invalid user id"), err)
}

func fmtInvalidOrderID(err error) error {
	return errors.Join(errors.New("invalid order id"), err)
}

func fmtInvalidExecutionID(err error) error {
	return errors.Join(errors.New("invalid execution id"), err)
}

func warnFactoryRunnerEnv(ctx core.ExecutionContext, err error, msg string) {
	if ctx.Logger != nil {
		ctx.Logger.WithError(err).Warn("factory runner env: " + msg)
		return
	}
	log.WithError(err).Warn("factory runner env: " + msg)
}

func upsertEnv(env []BrokerEnvironmentVariable, name, value string) []BrokerEnvironmentVariable {
	if strings.TrimSpace(value) == "" {
		return env
	}
	for i, item := range env {
		if item.Name == name {
			env[i].Value = value
			return env
		}
	}
	return append(env, BrokerEnvironmentVariable{Name: name, Value: value})
}

func stripEnv(env []BrokerEnvironmentVariable, name string) []BrokerEnvironmentVariable {
	if len(env) == 0 {
		return env
	}
	out := make([]BrokerEnvironmentVariable, 0, len(env))
	for _, item := range env {
		if item.Name == name {
			continue
		}
		out = append(out, item)
	}
	return out
}
