package models

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryPullRequestProviderGitHub    = "github"
	FactoryPullRequestProviderBitbucket = "bitbucket"

	FactoryPullRequestStateOpen   = "open"
	FactoryPullRequestStateDraft  = "draft"
	FactoryPullRequestStateClosed = "closed"
	FactoryPullRequestStateMerged = "merged"
)

const (
	factoryPullRequestProviderRepoNumberConstraint   = "idx_factory_pull_requests_factory_provider_repo_number"
	factoryPullRequestURLConstraint                  = "idx_factory_pull_requests_factory_url"
	factoryPullRequestProviderExternalConstraint     = "idx_factory_pull_requests_factory_provider_external"
	factoryPullRequestRunUniqueConstraint            = "idx_factory_pull_request_runs_run_unique"
	factoryPullRequestActivityActiveUniqueConstraint = "idx_factory_pull_request_runs_active_handler_revision"
	factoryPullRequestRevisionSHAUniqueConstraint    = "idx_factory_pull_request_revisions_pull_request_sha"

	FactoryPullRequestAccessConcurrent = "concurrent"
	FactoryPullRequestAccessWaiting    = "waiting"
	FactoryPullRequestAccessExclusive  = "exclusive"
	FactoryPullRequestAccessReleased   = "released"

	FactoryPullRequestActivityStateActive       = "active"
	FactoryPullRequestActivityStateFinished     = "finished"
	FactoryPullRequestActivityStateLimitReached = "limit_reached"
)

var (
	ErrFactoryPullRequestNotFound             = errors.New("factory pull request not found")
	ErrFactoryPullRequestInvalid              = errors.New("invalid factory pull request")
	ErrFactoryPullRequestAlreadyExists        = errors.New("factory pull request already exists")
	ErrFactoryPullRequestRunAlreadyLinked     = errors.New("run is already linked to a different pull request")
	ErrFactoryPullRequestLookupIncomplete     = errors.New("pull request lookup is incomplete")
	ErrFactoryPullRequestActivityNotFound     = errors.New("factory pull request activity not found")
	ErrFactoryPullRequestActivityDuplicate    = errors.New("factory pull request activity already exists for this handler and revision")
	ErrFactoryPullRequestActivityLimitReached = errors.New("factory pull request activity reached the attempt limit")
)

var factoryPullRequestProviders = []string{
	FactoryPullRequestProviderGitHub,
	FactoryPullRequestProviderBitbucket,
}

var factoryPullRequestStates = map[string]bool{
	FactoryPullRequestStateOpen:   true,
	FactoryPullRequestStateDraft:  true,
	FactoryPullRequestStateClosed: true,
	FactoryPullRequestStateMerged: true,
}

var (
	githubPullRequestURLPattern = regexp.MustCompile(
		`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/]+)/pull/(\d+)(?:[/?#].*)?$`,
	)
	bitbucketPullRequestURLPattern = regexp.MustCompile(
		`(?i)^https?://(?:www\.)?bitbucket\.org/([^/]+)/([^/]+)/pull-requests/(\d+)(?:[/?#].*)?$`,
	)
)

type FactoryPullRequest struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	FactoryID           uuid.UUID
	WorkOrderID         uuid.UUID
	Provider            string
	ExternalID          *string
	Repository          string
	Number              int64
	URL                 string
	Title               string
	State               string
	MergedAt            *time.Time
	ClosedAt            *time.Time
	CurrentRevisionID   *uuid.UUID
	ActiveMutationRunID *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type FactoryPullRequestRun struct {
	PullRequestID     uuid.UUID `gorm:"primaryKey"`
	RunID             uuid.UUID `gorm:"primaryKey"`
	FeedbackHandlerID *uuid.UUID
	RevisionID        *uuid.UUID
	Access            string
	State             string
	Description       string
	Attempt           *int
	AttemptLimit      *int
	AccessRequestedAt *time.Time
	AccessGrantedAt   *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type FactoryPullRequestLinkedRun struct {
	Run          CanvasRun
	Description  string
	Access       string
	State        string
	Attempt      *int
	AttemptLimit *int
	Revision     *FactoryPullRequestRevision
}

type FactoryPullRequestParams struct {
	Provider   string
	ExternalID string
	Repository string
	Number     int64
	URL        string
	Title      string
	State      string
	MergedAt   *time.Time
	ClosedAt   *time.Time
	Automation *factory.AutomationRef
	Run        *factory.RunRef
}

type FactoryPullRequestPatch struct {
	ExternalID *string
	Repository *string
	URL        *string
	Title      *string
	State      *string
	MergedAt   *time.Time
	ClosedAt   *time.Time
	Automation *factory.AutomationRef
	Run        *factory.RunRef
}

type FactoryPullRequestLookup struct {
	ID         uuid.UUID
	Provider   string
	ExternalID string
	Repository string
	Number     int64
	URL        string
}

type FactoryPullRequestFilter struct {
	WorkOrderID     *uuid.UUID
	WorkOrderIDs    []uuid.UUID
	WorkOrderNumber *int64
	State           string
	MergedFrom      *time.Time
	MergedTo        *time.Time
	ClosedFrom      *time.Time
	ClosedTo        *time.Time
}

func (FactoryPullRequest) TableName() string {
	return "factory_pull_requests"
}

func (FactoryPullRequestRun) TableName() string {
	return "factory_pull_request_runs"
}

func (p *FactoryPullRequest) Ref() *factory.PullRequestRef {
	if p == nil {
		return nil
	}
	return &factory.PullRequestRef{
		ID:         p.ID,
		Provider:   p.Provider,
		Repository: p.Repository,
		Number:     p.Number,
		URL:        p.URL,
		Title:      p.Title,
		State:      p.State,
	}
}

func (o *FactoryWorkOrder) CreatePullRequest(
	tx *gorm.DB,
	params FactoryPullRequestParams,
) (*FactoryPullRequest, error) {
	normalized, err := normalizeFactoryPullRequestParams(params)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	mergedAt, closedAt := pullRequestLifecycleStamps(
		normalized.State,
		"",
		normalized.MergedAt,
		normalized.ClosedAt,
		nil,
		nil,
		now,
	)

	pullRequest := &FactoryPullRequest{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Provider:       normalized.Provider,
		ExternalID:     normalized.ExternalID,
		Repository:     normalized.Repository,
		Number:         normalized.Number,
		URL:            normalized.URL,
		Title:          normalized.Title,
		State:          normalized.State,
		MergedAt:       mergedAt,
		ClosedAt:       closedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err = tx.Transaction(func(inner *gorm.DB) error {
		if createErr := inner.Clauses(clause.Returning{}).Create(pullRequest).Error; createErr != nil {
			return mapFactoryPullRequestConstraintError(createErr)
		}
		return o.RecordPullRequestAdded(inner, pullRequest.Ref(), params.Automation, params.Run)
	})
	if err != nil {
		return nil, err
	}

	return pullRequest, nil
}

func (p *FactoryPullRequest) Update(tx *gorm.DB, patch FactoryPullRequestPatch) error {
	nextProvider := p.Provider
	nextRepository := p.Repository
	nextNumber := p.Number
	nextURL := p.URL
	nextTitle := p.Title
	nextState := p.State
	nextExternalID := p.ExternalID

	if patch.Repository != nil {
		nextRepository = strings.TrimSpace(*patch.Repository)
	}
	if patch.URL != nil {
		nextURL = strings.TrimSpace(*patch.URL)
	}
	if patch.Title != nil {
		nextTitle = strings.TrimSpace(*patch.Title)
	}
	if patch.State != nil {
		nextState = strings.TrimSpace(*patch.State)
	}
	if patch.ExternalID != nil {
		trimmed := strings.TrimSpace(*patch.ExternalID)
		if trimmed == "" {
			nextExternalID = nil
		} else {
			nextExternalID = &trimmed
		}
	}

	normalized, err := normalizeFactoryPullRequestParams(FactoryPullRequestParams{
		Provider:   nextProvider,
		Repository: nextRepository,
		Number:     nextNumber,
		URL:        nextURL,
		Title:      nextTitle,
		State:      nextState,
	})
	if err != nil {
		return err
	}
	normalized.ExternalID = nextExternalID

	now := time.Now()
	mergedAt, closedAt := pullRequestLifecycleStamps(
		normalized.State,
		p.State,
		patch.MergedAt,
		patch.ClosedAt,
		p.MergedAt,
		p.ClosedAt,
		now,
	)

	p.ExternalID = normalized.ExternalID
	p.Repository = normalized.Repository
	p.URL = normalized.URL
	p.Title = normalized.Title
	p.State = normalized.State
	p.MergedAt = mergedAt
	p.ClosedAt = closedAt
	p.UpdatedAt = now

	return tx.Transaction(func(inner *gorm.DB) error {
		if saveErr := inner.Save(p).Error; saveErr != nil {
			return mapFactoryPullRequestConstraintError(saveErr)
		}

		order := &FactoryWorkOrder{
			ID:             p.WorkOrderID,
			OrganizationID: p.OrganizationID,
			FactoryID:      p.FactoryID,
		}
		return order.RecordPullRequestUpdated(inner, p.Ref(), patch.Automation, patch.Run)
	})
}

func (f *Factory) FindPullRequest(tx *gorm.DB, filter FactoryPullRequestLookup) (*FactoryPullRequest, error) {
	query := tx.Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID)

	switch {
	case filter.ID != uuid.Nil:
		query = query.Where("id = ?", filter.ID)
	case strings.TrimSpace(filter.Provider) != "" && strings.TrimSpace(filter.ExternalID) != "":
		query = query.Where("provider = ? AND external_id = ?", strings.TrimSpace(filter.Provider), strings.TrimSpace(filter.ExternalID))
	case strings.TrimSpace(filter.Provider) != "" && strings.TrimSpace(filter.Repository) != "" && filter.Number > 0:
		query = query.Where(
			"provider = ? AND repository = ? AND number = ?",
			strings.TrimSpace(filter.Provider),
			strings.TrimSpace(filter.Repository),
			filter.Number,
		)
	case strings.TrimSpace(filter.URL) != "":
		normalizedURL, err := normalizePullRequestURL(strings.TrimSpace(filter.URL))
		if err != nil {
			return nil, err
		}
		query = query.Where("url = ?", normalizedURL)
	default:
		return nil, ErrFactoryPullRequestLookupIncomplete
	}

	var pullRequest FactoryPullRequest
	err := query.First(&pullRequest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryPullRequestNotFound
		}
		return nil, err
	}

	return &pullRequest, nil
}

func (f *Factory) ListPullRequests(tx *gorm.DB, filter FactoryPullRequestFilter) ([]FactoryPullRequest, error) {
	query := tx.Model(&FactoryPullRequest{}).
		Select("factory_pull_requests.*").
		Joins("JOIN factory_work_orders ON factory_work_orders.id = factory_pull_requests.work_order_id").
		Where("factory_pull_requests.organization_id = ? AND factory_pull_requests.factory_id = ?", f.OrganizationID, f.ID)

	if filter.WorkOrderID != nil {
		query = query.Where("factory_pull_requests.work_order_id = ?", *filter.WorkOrderID)
	}
	if len(filter.WorkOrderIDs) > 0 {
		query = query.Where("factory_pull_requests.work_order_id IN ?", filter.WorkOrderIDs)
	}
	if filter.WorkOrderNumber != nil {
		query = query.Where("factory_work_orders.factory_id = ? AND factory_work_orders.number = ?", f.ID, *filter.WorkOrderNumber)
	}
	if filter.State != "" {
		if !factoryPullRequestStates[filter.State] {
			return nil, fmt.Errorf("%w: invalid state filter %q", ErrFactoryPullRequestInvalid, filter.State)
		}
		query = query.Where("factory_pull_requests.state = ?", filter.State)
	}

	var pullRequests []FactoryPullRequest
	err := query.
		Order("factory_work_orders.number ASC").
		Order("factory_pull_requests.created_at ASC").
		Order("factory_pull_requests.id ASC").
		Find(&pullRequests).
		Error
	if err != nil {
		return nil, err
	}
	return pullRequests, nil
}

func (p *FactoryPullRequest) LinkRun(tx *gorm.DB, runID uuid.UUID, description string) error {
	_, err := p.CreateActivity(tx, FactoryPullRequestActivityParams{
		RunID:       runID,
		Description: description,
		Access:      FactoryPullRequestAccessConcurrent,
	})
	return err
}

func ListPullRequestRuns(tx *gorm.DB, pullRequestIDs []uuid.UUID) (map[uuid.UUID][]FactoryPullRequestLinkedRun, error) {
	runsByPullRequest := map[uuid.UUID][]FactoryPullRequestLinkedRun{}
	if len(pullRequestIDs) == 0 {
		return runsByPullRequest, nil
	}

	var links []FactoryPullRequestRun
	err := tx.
		Where("pull_request_id IN ?", pullRequestIDs).
		Order("created_at ASC").
		Find(&links).
		Error
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return runsByPullRequest, nil
	}

	runIDs := make([]uuid.UUID, 0, len(links))
	seen := map[uuid.UUID]bool{}
	for _, link := range links {
		if seen[link.RunID] {
			continue
		}
		seen[link.RunID] = true
		runIDs = append(runIDs, link.RunID)
	}

	var runs []CanvasRun
	err = tx.Where("id IN ?", runIDs).Find(&runs).Error
	if err != nil {
		return nil, err
	}

	runByID := make(map[uuid.UUID]CanvasRun, len(runs))
	for _, run := range runs {
		runByID[run.ID] = run
	}

	revisionIDs := make([]uuid.UUID, 0)
	seenRevisions := map[uuid.UUID]bool{}
	for _, link := range links {
		if link.RevisionID == nil || seenRevisions[*link.RevisionID] {
			continue
		}
		seenRevisions[*link.RevisionID] = true
		revisionIDs = append(revisionIDs, *link.RevisionID)
	}
	revisionByID := map[uuid.UUID]FactoryPullRequestRevision{}
	if len(revisionIDs) > 0 {
		var revisions []FactoryPullRequestRevision
		if err := tx.Where("id IN ?", revisionIDs).Find(&revisions).Error; err != nil {
			return nil, err
		}
		for _, revision := range revisions {
			revisionByID[revision.ID] = revision
		}
	}

	seenRunByPullRequest := map[uuid.UUID]map[uuid.UUID]bool{}
	for _, link := range links {
		run, ok := runByID[link.RunID]
		if !ok {
			continue
		}
		seen := seenRunByPullRequest[link.PullRequestID]
		if seen == nil {
			seen = map[uuid.UUID]bool{}
			seenRunByPullRequest[link.PullRequestID] = seen
		}
		if seen[link.RunID] {
			continue
		}
		seen[link.RunID] = true
		linked := FactoryPullRequestLinkedRun{
			Run:          run,
			Description:  link.Description,
			Access:       link.Access,
			State:        link.State,
			Attempt:      link.Attempt,
			AttemptLimit: link.AttemptLimit,
		}
		if link.RevisionID != nil {
			if revision, ok := revisionByID[*link.RevisionID]; ok {
				copied := revision
				linked.Revision = &copied
			}
		}
		runsByPullRequest[link.PullRequestID] = append(runsByPullRequest[link.PullRequestID], linked)
	}

	return runsByPullRequest, nil
}

// FindWorkOrderIDsByLinkedRunIDs returns the work order each canvas run is
// linked to through factory_pull_request_runs.
func FindWorkOrderIDsByLinkedRunIDs(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	result := map[uuid.UUID]uuid.UUID{}
	if len(runIDs) == 0 {
		return result, nil
	}

	type row struct {
		RunID       uuid.UUID
		WorkOrderID uuid.UUID
	}
	var rows []row
	err := tx.Table("factory_pull_request_runs").
		Select("factory_pull_request_runs.run_id, factory_pull_requests.work_order_id").
		Joins("JOIN factory_pull_requests ON factory_pull_requests.id = factory_pull_request_runs.pull_request_id").
		Where("factory_pull_request_runs.run_id IN ?", runIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.RunID] = row.WorkOrderID
	}
	return result, nil
}

func ListPullRequestsByWorkOrderIDs(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID][]FactoryPullRequest, error) {
	result := map[uuid.UUID][]FactoryPullRequest{}
	if len(workOrderIDs) == 0 {
		return result, nil
	}

	var pullRequests []FactoryPullRequest
	err := tx.
		Where("work_order_id IN ?", workOrderIDs).
		Order("created_at ASC").
		Order("id ASC").
		Find(&pullRequests).
		Error
	if err != nil {
		return nil, err
	}

	for _, pullRequest := range pullRequests {
		result[pullRequest.WorkOrderID] = append(result[pullRequest.WorkOrderID], pullRequest)
	}
	return result, nil
}

func ListFactoryPullRequests(tx *gorm.DB, factoryID uuid.UUID, filter FactoryPullRequestFilter) ([]FactoryPullRequest, error) {
	query := tx.Where("factory_id = ?", factoryID)

	switch filter.State {
	case FactoryPullRequestStateMerged:
		query = query.Where("merged_at IS NOT NULL")
	case FactoryPullRequestStateClosed:
		query = query.Where("closed_at IS NOT NULL AND merged_at IS NULL")
	case "":
	default:
		if !factoryPullRequestStates[filter.State] {
			return nil, fmt.Errorf("%w: invalid state filter %q", ErrFactoryPullRequestInvalid, filter.State)
		}
		query = query.Where("state = ?", filter.State)
	}

	if filter.WorkOrderID != nil {
		query = query.Where("work_order_id = ?", *filter.WorkOrderID)
	}
	if filter.MergedFrom != nil {
		query = query.Where("merged_at >= ?", *filter.MergedFrom)
	}
	if filter.MergedTo != nil {
		query = query.Where("merged_at < ?", *filter.MergedTo)
	}
	if filter.ClosedFrom != nil {
		query = query.Where("closed_at >= ?", *filter.ClosedFrom)
	}
	if filter.ClosedTo != nil {
		query = query.Where("closed_at < ?", *filter.ClosedTo)
	}

	var pullRequests []FactoryPullRequest
	err := query.
		Order("merged_at DESC NULLS LAST").
		Order("closed_at DESC NULLS LAST").
		Find(&pullRequests).
		Error
	if err != nil {
		return nil, err
	}
	return pullRequests, nil
}

// ListFactoryPullRequestNumbers returns the numbers of every pull request
// SuperPlane opened in a repository, whatever state it reached.
//
// The velocity sync uses it to exclude SuperPlane work from the people merges
// it stores. State is deliberately not filtered: a pull request whose merge
// webhook has not arrived yet is still SuperPlane's, and must not be counted as
// people output in the meantime.
func ListFactoryPullRequestNumbers(tx *gorm.DB, factoryID uuid.UUID, repository string) ([]int64, error) {
	repository = strings.ToLower(strings.TrimSpace(repository))
	if repository == "" {
		return nil, nil
	}

	var numbers []int64
	err := tx.Model(&FactoryPullRequest{}).
		Where("factory_id = ? AND LOWER(repository) = ?", factoryID, repository).
		Pluck("number", &numbers).Error
	if err != nil {
		return nil, err
	}
	return numbers, nil
}

type normalizedPullRequestParams struct {
	Provider   string
	ExternalID *string
	Repository string
	Number     int64
	URL        string
	Title      string
	State      string
	MergedAt   *time.Time
	ClosedAt   *time.Time
}

func normalizeFactoryPullRequestParams(params FactoryPullRequestParams) (normalizedPullRequestParams, error) {
	provider := strings.ToLower(strings.TrimSpace(params.Provider))
	if provider == "" {
		provider = FactoryPullRequestProviderGitHub
	}
	if !validFactoryPullRequestProvider(provider) {
		return normalizedPullRequestParams{}, fmt.Errorf("%w: unsupported provider %q", ErrFactoryPullRequestInvalid, provider)
	}

	rawURL, err := normalizePullRequestURL(params.URL)
	if err != nil {
		return normalizedPullRequestParams{}, err
	}

	parsedProvider, parsedRepository, parsedNumber, err := parsePullRequestURL(rawURL)
	if err != nil {
		return normalizedPullRequestParams{}, err
	}
	if parsedProvider != "" && parsedProvider != provider {
		return normalizedPullRequestParams{}, fmt.Errorf("%w: url host does not match provider %q", ErrFactoryPullRequestInvalid, provider)
	}

	repository := strings.TrimSpace(params.Repository)
	if repository == "" {
		repository = parsedRepository
	}
	if !validPullRequestRepository(repository) {
		return normalizedPullRequestParams{}, fmt.Errorf("%w: repository must use the owner/name format", ErrFactoryPullRequestInvalid)
	}

	number := params.Number
	if number == 0 {
		number = parsedNumber
	}
	if number <= 0 {
		return normalizedPullRequestParams{}, fmt.Errorf("%w: number must be positive", ErrFactoryPullRequestInvalid)
	}

	state := strings.ToLower(strings.TrimSpace(params.State))
	if state == "" {
		state = FactoryPullRequestStateOpen
	}
	if !factoryPullRequestStates[state] {
		return normalizedPullRequestParams{}, fmt.Errorf("%w: invalid state %q", ErrFactoryPullRequestInvalid, state)
	}

	var externalID *string
	if trimmed := strings.TrimSpace(params.ExternalID); trimmed != "" {
		externalID = &trimmed
	}

	return normalizedPullRequestParams{
		Provider:   provider,
		ExternalID: externalID,
		Repository: repository,
		Number:     number,
		URL:        rawURL,
		Title:      strings.TrimSpace(params.Title),
		State:      state,
		MergedAt:   params.MergedAt,
		ClosedAt:   params.ClosedAt,
	}, nil
}

func normalizePullRequestURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: url is required", ErrFactoryPullRequestInvalid)
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: url must be http(s)", ErrFactoryPullRequestInvalid)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: url must be http(s)", ErrFactoryPullRequestInvalid)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("%w: url must be http(s)", ErrFactoryPullRequestInvalid)
	}

	parsed.Fragment = ""
	normalized := strings.TrimRight(parsed.String(), "/")
	return normalized, nil
}

func parsePullRequestURL(raw string) (provider, repository string, number int64, err error) {
	if matches := githubPullRequestURLPattern.FindStringSubmatch(raw); len(matches) == 4 {
		parsedNumber, parseErr := strconv.ParseInt(matches[3], 10, 64)
		if parseErr != nil {
			return "", "", 0, fmt.Errorf("%w: url number is invalid", ErrFactoryPullRequestInvalid)
		}
		return FactoryPullRequestProviderGitHub, matches[1] + "/" + matches[2], parsedNumber, nil
	}
	if matches := bitbucketPullRequestURLPattern.FindStringSubmatch(raw); len(matches) == 4 {
		parsedNumber, parseErr := strconv.ParseInt(matches[3], 10, 64)
		if parseErr != nil {
			return "", "", 0, fmt.Errorf("%w: url number is invalid", ErrFactoryPullRequestInvalid)
		}
		return FactoryPullRequestProviderBitbucket, matches[1] + "/" + matches[2], parsedNumber, nil
	}
	return "", "", 0, nil
}

func validFactoryPullRequestProvider(provider string) bool {
	return slices.Contains(factoryPullRequestProviders, provider)
}

func validPullRequestRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func pullRequestLifecycleStamps(
	state, previousState string,
	explicitMerged, explicitClosed, existingMerged, existingClosed *time.Time,
	now time.Time,
) (mergedAt *time.Time, closedAt *time.Time) {
	mergedAt = existingMerged
	if mergedAt == nil {
		switch {
		case explicitMerged != nil:
			mergedAt = explicitMerged
		case state == FactoryPullRequestStateMerged && previousState != FactoryPullRequestStateMerged:
			stamp := now
			mergedAt = &stamp
		}
	}

	closedAt = existingClosed
	if closedAt == nil {
		switch {
		case explicitClosed != nil:
			closedAt = explicitClosed
		case state == FactoryPullRequestStateClosed && previousState != FactoryPullRequestStateClosed:
			stamp := now
			closedAt = &stamp
		}
	}

	return mergedAt, closedAt
}

func mapFactoryPullRequestConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case factoryPullRequestProviderRepoNumberConstraint,
			factoryPullRequestURLConstraint,
			factoryPullRequestProviderExternalConstraint:
			return ErrFactoryPullRequestAlreadyExists
		}
	}

	return err
}

func mapFactoryPullRequestRunConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case factoryPullRequestRunUniqueConstraint:
			return ErrFactoryPullRequestRunAlreadyLinked
		case factoryPullRequestActivityActiveUniqueConstraint:
			return ErrFactoryPullRequestActivityDuplicate
		}
	}

	return err
}
