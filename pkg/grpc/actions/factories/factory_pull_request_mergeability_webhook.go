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
)

const (
	factoryMergeabilityWebhookKey      = "factoryMergeability"
	factoryMergeabilityRefreshAttempts = 3
	factoryMergeabilityRefreshDelay    = 200 * time.Millisecond
)

var factoryMergeabilityUnknownDelays = []time.Duration{
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
	20 * time.Second,
	20 * time.Second,
}

var factoryMergeabilityGitHubEvents = []string{"check_run", "check_suite", "status", "pull_request", "push"}

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
		if payload.PullRequest != nil && payload.PullRequest.Merged {
			refreshOpenFactoryPullRequestsOnBase(ctx, db, deps, integration, repository, payload.PullRequest.Base.Ref, payload.PullRequest.Number)
		}
		return
	}
	if isGitHubPushEvent(eventType) {
		branch, ok := gitHubBranchFromRef(payload.Ref)
		if !ok {
			return
		}
		refreshOpenFactoryPullRequestsOnBase(ctx, db, deps, integration, repository, branch, 0)
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
		if err := pollFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest, ""); err != nil {
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
	if err := refreshFactoryPullRequestMergeabilityByID(ctx, deps, organizationID, factoryID, pullRequestID); err != nil {
		log.WithError(err).Warnf("factory mergeability: refresh failed for pull request %s", pullRequestID)
	}
}

func pollFactoryPullRequestMergeability(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	expectedBase string,
) error {
	delays := factoryMergeabilityUnknownDelays
	attempts := len(delays) + 1
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = refreshFactoryPullRequestMergeabilityWithRetry(ctx, db, deps, factory, pullRequest, factoryMergeabilityRefreshOptions{
			ExpectedBase:   expectedBase,
			PersistUnknown: attempt == attempts,
		})
		if err == nil || !errors.Is(err, errFactoryPullRequestMergeabilityUnknown) {
			return err
		}
		if attempt == attempts {
			return err
		}
		if !waitForFactoryMergeabilityUnknown(ctx, delays[attempt-1]) {
			return ctx.Err()
		}
	}
	return err
}

func waitForFactoryMergeabilityUnknown(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func refreshOpenFactoryPullRequestsOnBase(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	integration *models.Integration,
	repository, baseRef string,
	excludeNumber int64,
) {
	if integration == nil {
		return
	}
	if err := ensureGitHubFactoryMergeabilityWebhook(ctx, db, deps.Encryptor, integration, repository); err != nil {
		log.WithError(err).Warnf("factory mergeability: failed to ensure webhook for %s", repository)
	}

	pullRequests, err := models.ListOpenGitHubFactoryPullRequestsInRepository(db, integration.OrganizationID, repository)
	if err != nil {
		log.WithError(err).Warnf("factory mergeability: failed to list open pull requests in %s", repository)
		return
	}

	factoriesByID := map[string]*models.Factory{}
	for i := range pullRequests {
		pullRequest := &pullRequests[i]
		if excludeNumber > 0 && pullRequest.Number == excludeNumber {
			continue
		}
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
		if err := pollFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest, baseRef); err != nil {
			log.WithError(err).Warnf("factory mergeability: failed to refresh pull request %s", pullRequest.ID)
		}
	}
}

func refreshFactoryPullRequestMergeabilityWithRetry(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	opts factoryMergeabilityRefreshOptions,
) error {
	var err error
	for attempt := 1; attempt <= factoryMergeabilityRefreshAttempts; attempt++ {
		err = refreshFactoryPullRequestMergeabilityOptions(ctx, db, deps, factory, pullRequest, opts)
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
	return pollFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest, "")
}

func refreshFactoryPullRequestMergeability(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) error {
	return refreshFactoryPullRequestMergeabilityOptions(ctx, db, deps, factory, pullRequest, immediateFactoryMergeabilityRefreshOptions())
}

func refreshFactoryPullRequestMergeabilityOptions(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	opts factoryMergeabilityRefreshOptions,
) error {
	before := storedFactoryPullRequestMergeability(pullRequest)
	result, err := syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest, opts)
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
	Ref        string `json:"ref"`
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
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
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
	case "ping", "pull_request", "check_run", "check_suite", "status", "push":
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

	webhooks, err := models.ListIntegrationWebhooks(tx, integration.ID)
	if err != nil {
		return fmt.Errorf("list factory mergeability webhooks: %w", err)
	}
	for i := range webhooks {
		hook := &webhooks[i]
		if !isFactoryMergeabilityWebhook(hook.Configuration.Data()) {
			continue
		}
		if factoryMergeabilityWebhookRepository(hook.Configuration.Data()) != repository {
			continue
		}
		return updateFactoryMergeabilityWebhook(tx, hook)
	}

	return createFactoryMergeabilityWebhook(ctx, tx, encryptor, integration.ID, repository)
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

func updateFactoryMergeabilityWebhook(tx *gorm.DB, hook *models.Webhook) error {
	needsEvents := !factoryMergeabilityWebhookHasEvent(hook.Configuration.Data(), "push")
	needsRetry := hook.State == models.WebhookStateFailed
	if !needsEvents && !needsRetry {
		return nil
	}

	updates := map[string]any{
		"state":       models.WebhookStatePending,
		"retry_count": 0,
		"updated_at":  time.Now(),
	}
	if needsEvents {
		updates["configuration"] = datatypes.NewJSONType(any(factoryMergeabilityWebhookConfiguration(
			factoryMergeabilityWebhookRepository(hook.Configuration.Data()),
		)))
	}
	return tx.Model(hook).Updates(updates).Error
}

func factoryMergeabilityWebhookHasEvent(configuration any, event string) bool {
	config, ok := configuration.(map[string]any)
	if !ok {
		return false
	}
	switch events := config["eventTypes"].(type) {
	case []string:
		return slices.Contains(events, event)
	case []any:
		for _, item := range events {
			value, ok := item.(string)
			if ok && value == event {
				return true
			}
		}
	}
	return false
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

func isGitHubPushEvent(eventType string) bool {
	return strings.EqualFold(strings.TrimSpace(eventType), "push")
}

func gitHubBranchFromRef(ref string) (string, bool) {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	branch := strings.TrimPrefix(ref, prefix)
	if branch == "" || strings.Contains(branch, "..") {
		return "", false
	}
	return branch, true
}

func parseGitHubMergeabilityWebhookPayload(eventType string, body []byte) (githubMergeabilityWebhookPayload, bool) {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "pull_request", "check_run", "check_suite", "status", "push":
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
