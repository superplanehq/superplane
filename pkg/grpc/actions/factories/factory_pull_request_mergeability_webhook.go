package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	factoryMergeabilityWebhookKey      = "factoryMergeability"
	factoryMergeabilityRefreshAttempts = 3
	factoryMergeabilityRefreshDelay    = 200 * time.Millisecond
)

var factoryMergeabilityGitHubEvents = []string{"check_run", "check_suite", "status", "pull_request"}

func RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
	ctx context.Context,
	deps IntakeDependencies,
	webhook *models.Webhook,
	eventType string,
	body []byte,
) {
	if webhook == nil || webhook.AppInstallationID == nil {
		return
	}

	payload, ok := parseGitHubMergeabilityWebhookPayload(eventType, body)
	if !ok {
		return
	}
	repository := strings.TrimSpace(payload.Repository.FullName)
	configured := factoryMergeabilityWebhookRepository(webhook.Configuration.Data())
	if configured != "" && !strings.EqualFold(configured, repository) {
		return
	}

	db := database.DB(ctx)
	integration, err := models.FindUnscopedIntegrationInTransaction(db, *webhook.AppInstallationID)
	if err != nil {
		log.WithError(err).Warn("factory mergeability: webhook integration not found")
		return
	}

	if isGitHubPullRequestClosedEvent(eventType, payload) {
		closeFactoryWorkOrdersFromGitHubPullRequestClosed(db, integration, payload)
		return
	}

	_, numbers, sha := githubMergeabilityWebhookRefFromPayload(payload)
	pullRequests, err := models.ListOpenGitHubFactoryPullRequestsForWebhook(
		db,
		integration.OrganizationID,
		repository,
		numbers,
		sha,
	)
	if err != nil {
		log.WithError(err).Warn("factory mergeability: failed to list pull requests for GitHub event")
		return
	}
	if len(pullRequests) == 0 {
		return
	}

	factoriesByID := map[string]*models.Factory{}
	for i := range pullRequests {
		pullRequest := &pullRequests[i]
		factory := factoriesByID[pullRequest.FactoryID.String()]
		if factory == nil {
			loaded, findErr := models.FindFactory(db, pullRequest.OrganizationID, pullRequest.FactoryID)
			if findErr != nil {
				log.WithError(findErr).Warnf("factory mergeability: factory %s not found", pullRequest.FactoryID)
				continue
			}
			factory = loaded
			factoriesByID[factory.ID.String()] = factory
		}
		if err := refreshFactoryPullRequestMergeabilityWithRetry(ctx, db, deps, factory, pullRequest); err != nil {
			log.WithError(err).Warnf("factory mergeability: failed to refresh pull request %s", pullRequest.ID)
		}
	}
}

var publishFactoryWorkOrderUpdated = messages.PublishFactoryWorkOrderUpdated

func ScheduleFactoryPullRequestMergeabilityRefresh(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID, factoryID, pullRequestID uuid.UUID,
) {
	if organizationID == uuid.Nil || factoryID == uuid.Nil || pullRequestID == uuid.Nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	go retryFactoryPullRequestMergeabilityRefresh(ctx, deps, organizationID, factoryID, pullRequestID)
}

func retryFactoryPullRequestMergeabilityRefresh(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID, factoryID, pullRequestID uuid.UUID,
) {
	var err error
	for attempt := 1; attempt <= factoryMergeabilityRefreshAttempts; attempt++ {
		err = refreshFactoryPullRequestMergeabilityByID(ctx, deps, organizationID, factoryID, pullRequestID)
		if err == nil {
			return
		}
		log.WithError(err).Warnf(
			"factory mergeability: refresh attempt %d failed for pull request %s",
			attempt,
			pullRequestID,
		)
		if errors.Is(err, errFactoryPullRequestMergeabilityUnavailable) &&
			!errors.Is(err, errFactoryPullRequestMergeabilityTemporary) {
			return
		}
		if attempt < factoryMergeabilityRefreshAttempts && !waitForFactoryMergeabilityRefresh(ctx, attempt) {
			return
		}
	}
}

func refreshFactoryPullRequestMergeabilityWithRetry(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) error {
	var err error
	for attempt := 1; attempt <= factoryMergeabilityRefreshAttempts; attempt++ {
		err = refreshFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
		if err == nil || !errors.Is(err, errFactoryPullRequestMergeabilityTemporary) {
			return err
		}
		log.WithError(err).Warnf(
			"factory mergeability: refresh attempt %d failed for pull request %s",
			attempt,
			pullRequest.ID,
		)
		if attempt < factoryMergeabilityRefreshAttempts && !waitForFactoryMergeabilityRefresh(ctx, attempt) {
			return ctx.Err()
		}
	}
	return err
}

func waitForFactoryMergeabilityRefresh(ctx context.Context, attempt int) bool {
	timer := time.NewTimer(time.Duration(attempt) * factoryMergeabilityRefreshDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func refreshFactoryPullRequestMergeabilityByID(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID, factoryID, pullRequestID uuid.UUID,
) error {
	db := database.DB(ctx)
	factory, err := models.FindFactory(db, organizationID, factoryID)
	if err != nil {
		return fmt.Errorf("factory %s not found: %w", factoryID, err)
	}
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequestID})
	if err != nil {
		return fmt.Errorf("pull request %s not found: %w", pullRequestID, err)
	}
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub {
		return nil
	}
	if err := ensureFactoryMergeabilityWebhookForRepository(ctx, db, deps, factory, pullRequest.Repository); err != nil {
		log.WithError(err).Warnf(
			"factory mergeability: failed to ensure webhook for %s",
			pullRequest.Repository,
		)
	}
	return refreshFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
}

func refreshFactoryPullRequestMergeability(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) error {
	before := storedFactoryPullRequestMergeability(pullRequest)
	result, err := syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
	if err != nil {
		return fmt.Errorf("failed to refresh pull request %s: %w", pullRequest.ID, err)
	}
	if result != nil && result.BlockedReason == pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNAVAILABLE {
		return unavailableFactoryPullRequestMergeabilityError(pullRequest.ID, result.canRetry)
	}
	if storedFactoryPullRequestMergeability(pullRequest) == before {
		return nil
	}
	if err := publishFactoryWorkOrderUpdated(
		factory.ID.String(),
		pullRequest.WorkOrderID.String(),
		factoryevents.EventTypeOrderPullRequestUpdated,
	); err != nil {
		log.WithError(err).Warnf("factory mergeability: failed to publish update for order %s", pullRequest.WorkOrderID)
	}
	return nil
}

func storedFactoryPullRequestMergeability(pullRequest *models.FactoryPullRequest) models.FactoryPullRequestMergeabilitySnapshot {
	if pullRequest == nil {
		return models.FactoryPullRequestMergeabilitySnapshot{}
	}
	return models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable:      pullRequest.Mergeable,
		BlockedReason:  pullRequest.MergeBlockedReason,
		BlockedMessage: pullRequest.MergeBlockedMessage,
		HeadSHA:        pullRequest.MergeableHeadSHA,
		AllowedMethods: pullRequest.MergeableAllowedMethods,
	}
}

type githubWebhookUser struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

type githubMergeabilityWebhookPayload struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Sender      *githubWebhookUser `json:"sender"`
	PullRequest *struct {
		Number   int64              `json:"number"`
		Merged   bool               `json:"merged"`
		MergedAt string             `json:"merged_at"`
		ClosedAt string             `json:"closed_at"`
		MergedBy *githubWebhookUser `json:"merged_by"`
		Head     struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
	CheckRun *struct {
		HeadSHA      string `json:"head_sha"`
		PullRequests []struct {
			Number int64 `json:"number"`
		} `json:"pull_requests"`
	} `json:"check_run"`
	CheckSuite *struct {
		HeadSHA      string `json:"head_sha"`
		PullRequests []struct {
			Number int64 `json:"number"`
		} `json:"pull_requests"`
	} `json:"check_suite"`
	SHA string `json:"sha"`
}

// IsGitHubFactoryMergeabilityEvent reports whether a GitHub webhook can
// refresh factory pull-request mergeability. ping is included so GitHub
// accepts a hook that has no canvas nodes.
func IsGitHubFactoryMergeabilityEvent(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "ping", "pull_request", "check_run", "check_suite", "status":
		return true
	default:
		return false
	}
}

func ensureFactoryMergeabilityWebhook(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
) error {
	if factory == nil {
		return nil
	}
	return ensureFactoryMergeabilityWebhookForRepository(
		ctx,
		tx,
		deps,
		factory,
		factory.OnboardingConfigValue().AppRepository,
	)
}

func ensureFactoryMergeabilityWebhookForRepository(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	repository string,
) error {
	if deps.Encryptor == nil || factory == nil {
		return nil
	}

	repository = strings.TrimSpace(repository)
	integrationID := strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID)
	if repository == "" || integrationID == "" {
		return nil
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil
	}
	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		return nil
	}
	if integration.State != models.IntegrationStateReady {
		return nil
	}

	return ensureGitHubFactoryMergeabilityWebhook(ctx, tx, deps.Encryptor, integration, repository)
}

func ensureGitHubFactoryMergeabilityWebhook(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	integration *models.Integration,
	repository string,
) error {
	if encryptor == nil || integration == nil || integration.AppName != "github" {
		return nil
	}
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return nil
	}

	return tx.Transaction(func(tx *gorm.DB) error {
		var locked models.Integration
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", integration.ID).
			First(&locked).Error; err != nil {
			return fmt.Errorf("lock integration for factory mergeability webhook: %w", err)
		}

		hook, err := findFactoryMergeabilityWebhook(tx, integration.ID, repository)
		if err != nil {
			return err
		}
		if hook != nil {
			return nil
		}
		return createFactoryMergeabilityWebhook(ctx, tx, encryptor, integration.ID, repository)
	})
}

func factoryMergeabilityWebhookConfiguration(repository string) map[string]any {
	return map[string]any{
		"eventTypes":                  slices.Clone(factoryMergeabilityGitHubEvents),
		"repository":                  repository,
		factoryMergeabilityWebhookKey: true,
	}
}

func IsFactoryMergeabilityWebhook(webhook *models.Webhook) bool {
	if webhook == nil {
		return false
	}
	return isFactoryMergeabilityWebhook(webhook.Configuration.Data())
}

func isFactoryMergeabilityWebhook(configuration any) bool {
	config, ok := configuration.(map[string]any)
	if !ok {
		return false
	}
	flag, ok := config[factoryMergeabilityWebhookKey].(bool)
	return ok && flag
}

func factoryMergeabilityWebhookRepository(configuration any) string {
	config, ok := configuration.(map[string]any)
	if !ok {
		return ""
	}
	repository, _ := config["repository"].(string)
	return strings.TrimSpace(repository)
}

func matchesFactoryMergeabilityWebhook(hook *models.Webhook, repository string) bool {
	if hook == nil || !isFactoryMergeabilityWebhook(hook.Configuration.Data()) {
		return false
	}
	return factoryMergeabilityWebhookRepository(hook.Configuration.Data()) == strings.TrimSpace(repository)
}

func findFactoryMergeabilityWebhook(tx *gorm.DB, integrationID uuid.UUID, repository string) (*models.Webhook, error) {
	repository = strings.TrimSpace(repository)
	if integrationID == uuid.Nil || repository == "" {
		return nil, nil
	}

	webhooks, err := models.ListIntegrationWebhooks(tx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("list factory mergeability webhooks: %w", err)
	}
	for i := range webhooks {
		hook := &webhooks[i]
		if matchesFactoryMergeabilityWebhook(hook, repository) {
			return hook, nil
		}
	}
	return nil, nil
}

func findFactoryMergeabilityWebhookForPullRequest(
	tx *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) (*models.Webhook, error) {
	if factory == nil || pullRequest == nil || pullRequest.Provider != models.FactoryPullRequestProviderGitHub {
		return nil, nil
	}

	integrationID := strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID)
	if integrationID == "" {
		return nil, nil
	}
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, nil
	}
	return findFactoryMergeabilityWebhook(tx, id, pullRequest.Repository)
}

const (
	githubWebhookHookLimit       = "cannot have more than 20 hooks"
	webhookSetupHookLimitMessage = "SuperPlane could not register a webhook on this repository. GitHub allows 20 pull request webhooks, and this repository already has 20. Remove an unused webhook, then try again."
	webhookSetupFallbackMessage  = "SuperPlane could not register a webhook on this repository. Try again."
)

func factoryWebhookSetupMessage(stored string) string {
	if strings.Contains(stored, githubWebhookHookLimit) {
		return webhookSetupHookLimitMessage
	}
	return webhookSetupFallbackMessage
}

func webhookSetupInProgress(hook *models.Webhook, creationExpected bool) bool {
	if hook == nil {
		return creationExpected
	}
	return hook.State == models.WebhookStatePending || hook.State == models.WebhookStateProvisioning
}

func factoryMergeabilityWebhookExpected(tx *gorm.DB, factory *models.Factory, repository string) bool {
	if factory == nil {
		return false
	}
	repository = strings.TrimSpace(repository)
	integrationID := strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID)
	if repository == "" || integrationID == "" {
		return false
	}
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return false
	}
	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		return false
	}
	return integration.State == models.IntegrationStateReady && integration.AppName == "github"
}

func failedFactoryMergeabilityResult(pullRequest *models.FactoryPullRequest, hook *models.Webhook) *factoryPullRequestMergeability {
	result := &factoryPullRequestMergeability{PullRequest: pullRequest}
	message := webhookSetupFallbackMessage
	if hook != nil {
		message = factoryWebhookSetupMessage(hook.LastError)
	}
	return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_WEBHOOK_FAILED, message)
}

func createFactoryMergeabilityWebhook(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	integrationID uuid.UUID,
	repository string,
) error {
	webhookID := uuid.New()
	_, encryptedKey, err := crypto.NewRandomKey(ctx, encryptor, webhookID.String())
	if err != nil {
		return fmt.Errorf("generate factory mergeability webhook secret: %w", err)
	}

	now := time.Now()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            encryptedKey,
		Configuration:     datatypes.NewJSONType(any(factoryMergeabilityWebhookConfiguration(repository))),
		AppInstallationID: &integrationID,
		CreatedAt:         &now,
	}
	if err := tx.Create(&webhook).Error; err != nil {
		return fmt.Errorf("create factory mergeability webhook: %w", err)
	}
	return nil
}

func VerifyGitHubFactoryMergeabilitySignature(
	ctx context.Context,
	encryptor crypto.Encryptor,
	webhook *models.Webhook,
	headers http.Header,
	body []byte,
) (int, error) {
	if encryptor == nil || webhook == nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature := strings.TrimPrefix(headers.Get("X-Hub-Signature-256"), "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := encryptor.Decrypt(ctx, webhook.Secret, []byte(webhook.ID.String()))
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}
	if err := crypto.VerifySignature(secret, body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}
	return http.StatusOK, nil
}

func isGitHubPullRequestClosedEvent(eventType string, payload githubMergeabilityWebhookPayload) bool {
	return strings.EqualFold(strings.TrimSpace(eventType), "pull_request") &&
		strings.EqualFold(strings.TrimSpace(payload.Action), "closed") &&
		payload.PullRequest != nil
}

func parseGitHubMergeabilityWebhookPayload(eventType string, body []byte) (githubMergeabilityWebhookPayload, bool) {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "pull_request", "check_run", "check_suite", "status":
	default:
		return githubMergeabilityWebhookPayload{}, false
	}

	var payload githubMergeabilityWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return githubMergeabilityWebhookPayload{}, false
	}
	if strings.TrimSpace(payload.Repository.FullName) == "" {
		return githubMergeabilityWebhookPayload{}, false
	}
	return payload, true
}

func githubMergeabilityWebhookRef(eventType string, body []byte) (repository string, numbers []int64, sha string) {
	payload, ok := parseGitHubMergeabilityWebhookPayload(eventType, body)
	if !ok {
		return "", nil, ""
	}
	return githubMergeabilityWebhookRefFromPayload(payload)
}

func githubMergeabilityWebhookRefFromPayload(payload githubMergeabilityWebhookPayload) (repository string, numbers []int64, sha string) {
	repository = strings.TrimSpace(payload.Repository.FullName)
	if repository == "" {
		return "", nil, ""
	}

	seen := map[int64]struct{}{}
	addNumber := func(number int64) {
		if number <= 0 {
			return
		}
		if _, ok := seen[number]; ok {
			return
		}
		seen[number] = struct{}{}
		numbers = append(numbers, number)
	}

	if payload.PullRequest != nil {
		addNumber(payload.PullRequest.Number)
		sha = strings.TrimSpace(payload.PullRequest.Head.SHA)
	}
	if payload.CheckRun != nil {
		if sha == "" {
			sha = strings.TrimSpace(payload.CheckRun.HeadSHA)
		}
		for _, pullRequest := range payload.CheckRun.PullRequests {
			addNumber(pullRequest.Number)
		}
	}
	if payload.CheckSuite != nil {
		if sha == "" {
			sha = strings.TrimSpace(payload.CheckSuite.HeadSHA)
		}
		for _, pullRequest := range payload.CheckSuite.PullRequests {
			addNumber(pullRequest.Number)
		}
	}
	if sha == "" {
		sha = strings.TrimSpace(payload.SHA)
	}
	return repository, numbers, sha
}
