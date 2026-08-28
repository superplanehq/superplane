package factories

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	velocityPeriodDaysDefault = 7
	velocityPeriodDaysMax     = 30
	velocitySearchMaxPages    = 3
	velocitySearchPageSize    = 100
)

var githubPRURLRegex = regexp.MustCompile(`github\.com/(?:repos/)?([^/]+)/([^/]+)/(?:pull|pulls)/(\d+)`)

func DescribeFactoryVelocity(
	ctx context.Context,
	reg *registry.Registry,
	organizationID string,
	req *pb.DescribeFactoryVelocityRequest,
) (*pb.DescribeFactoryVelocityResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	period := clampPeriodDays(int(req.GetPeriodDays()))

	db := database.DB(ctx)
	if _, err := models.FindFactory(db, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	now := time.Now().In(time.Local)
	buckets := buildDayBuckets(now, period)

	repoOwner, repoName, hasRepo := parseOwnerRepo(req.GetRepository())

	artifacts, err := listVelocityArtifacts(db, factoryID, buckets[0].start, buckets[len(buckets)-1].end)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	superplaneMerges, superplaneWaste := classifySuperPlaneArtifacts(artifacts, repoOwner, repoName, hasRepo)

	hasPeople := false
	peopleSearchFailed := false
	var peopleHits []peopleMerge
	if hasRepo && req.GetIntegrationId() != "" {
		peopleHits, hasPeople, peopleSearchFailed = loadPeopleCohort(peopleCohortRequest{
			ctx:           ctx,
			reg:           reg,
			tx:            db,
			orgID:         orgID,
			factoryID:     factoryID,
			integrationID: req.GetIntegrationId(),
			repoOwner:     repoOwner,
			repoName:      repoName,
			from:          buckets[0].start,
			endExclusive:  buckets[len(buckets)-1].end,
		})
	}

	fillBuckets(buckets, superplaneMerges, superplaneWaste, peopleHits)

	yesterdayIdx := len(buckets) - 2
	if yesterdayIdx < 0 {
		yesterdayIdx = 0
	}
	yesterday := buckets[yesterdayIdx]

	totals := aggregateTotals(buckets, hasPeople)

	points := make([]*pb.DescribeFactoryVelocityDay, 0, len(buckets))
	for i := range buckets {
		b := &buckets[i]
		points = append(points, &pb.DescribeFactoryVelocityDay{
			Day:              dayLabel(b.start, period, i),
			Date:             timestamppb.New(b.start),
			SuperplaneMerged: int32(b.superplaneMerged),
			PeopleMerged:     int32(b.peopleMerged),
			Waste:            int32(b.waste),
		})
	}

	return &pb.DescribeFactoryVelocityResponse{
		Yesterday: &pb.DescribeFactoryVelocityYesterday{
			Date:             timestamppb.New(calendarDayUTCNoon(yesterday.start)),
			SuperplaneMerged: int32(yesterday.superplaneMerged),
			Waste:            int32(yesterday.waste),
		},
		Totals:             totals,
		Points:             points,
		Repository:         joinOwnerRepo(repoOwner, repoName),
		HasPeopleCohort:    hasPeople,
		PeopleSearchFailed: peopleSearchFailed,
	}, nil
}

func clampPeriodDays(v int) int {
	if v <= 0 {
		return velocityPeriodDaysDefault
	}
	if v > velocityPeriodDaysMax {
		return velocityPeriodDaysMax
	}
	return v
}

type dayBucket struct {
	start            time.Time
	end              time.Time
	superplaneMerged int
	peopleMerged     int
	waste            int
}

func dayLabel(start time.Time, periodDays, index int) string {
	if periodDays <= 7 {
		return start.Format("Mon")
	}
	dayNumber := index + 1
	if dayNumber == 1 || index%5 == 0 || index == periodDays-1 {
		return strconv.Itoa(dayNumber)
	}
	return ""
}

func buildDayBuckets(now time.Time, periodDays int) []dayBucket {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	buckets := make([]dayBucket, periodDays)
	for i := 0; i < periodDays; i++ {
		start := today.AddDate(0, 0, -(periodDays - 1 - i))
		buckets[i] = dayBucket{start: start, end: start.AddDate(0, 0, 1)}
	}
	return buckets
}

// calendarDayUTCNoon returns 12:00 UTC on the civil date of t in t's location.
// The UI formats this instant in UTC so the label matches the server calendar
// day, independent of the browser timezone.
func calendarDayUTCNoon(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

type prArtifactMeta struct {
	url        string
	owner      string
	repo       string
	number     int
	mergedAt   *time.Time
	closedAt   *time.Time
	isMerged   bool
	isClosedNM bool
}

func listVelocityArtifacts(tx *gorm.DB, factoryID uuid.UUID, from, to time.Time) ([]prArtifactMeta, error) {
	merged, err := models.ListFactoryPullRequests(tx, factoryID, models.FactoryPullRequestFilter{
		State:      models.FactoryPullRequestStateMerged,
		MergedFrom: &from,
		MergedTo:   &to,
	})
	if err != nil {
		return nil, err
	}

	closed, err := models.ListFactoryPullRequests(tx, factoryID, models.FactoryPullRequestFilter{
		State:      models.FactoryPullRequestStateClosed,
		ClosedFrom: &from,
		ClosedTo:   &to,
	})
	if err != nil {
		return nil, err
	}

	out := make([]prArtifactMeta, 0, len(merged)+len(closed))
	for i := range merged {
		if meta, ok := toPRMeta(&merged[i]); ok {
			meta.isMerged = true
			out = append(out, meta)
		}
	}
	for i := range closed {
		if meta, ok := toPRMeta(&closed[i]); ok {
			meta.isClosedNM = true
			out = append(out, meta)
		}
	}
	return out, nil
}

func listKnownSuperPlanePRs(tx *gorm.DB, factoryID uuid.UUID) ([]prArtifactMeta, error) {
	// Every SuperPlane PR URL, including pull requests that predate
	// merged_at / closed_at. People search still returns those PRs;
	// subtracting only windowed timestamps would count them as People.
	pullRequests, err := models.ListFactoryPullRequests(tx, factoryID, models.FactoryPullRequestFilter{})
	if err != nil {
		return nil, err
	}

	out := make([]prArtifactMeta, 0, len(pullRequests))
	for i := range pullRequests {
		if meta, ok := toPRMeta(&pullRequests[i]); ok {
			out = append(out, meta)
		}
	}
	return out, nil
}

func toPRMeta(pullRequest *models.FactoryPullRequest) (prArtifactMeta, bool) {
	if pullRequest == nil || strings.TrimSpace(pullRequest.URL) == "" {
		return prArtifactMeta{}, false
	}

	owner, repo, ok := parseOwnerRepo(pullRequest.Repository)
	number := int(pullRequest.Number)
	if !ok || number == 0 {
		owner, repo, number = parsePRURL(pullRequest.URL)
	}
	return prArtifactMeta{
		url:      pullRequest.URL,
		owner:    owner,
		repo:     repo,
		number:   number,
		mergedAt: pullRequest.MergedAt,
		closedAt: pullRequest.ClosedAt,
	}, true
}

func parsePRURL(url string) (owner, repo string, number int) {
	m := githubPRURLRegex.FindStringSubmatch(url)
	if len(m) != 4 {
		return "", "", 0
	}
	owner = strings.ToLower(m[1])
	repo = strings.ToLower(m[2])
	_, _ = fmt.Sscanf(m[3], "%d", &number)
	return owner, repo, number
}

func parseOwnerRepo(repository string) (owner, repo string, ok bool) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return "", "", false
	}
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return strings.ToLower(parts[0]), strings.ToLower(parts[1]), true
}

func joinOwnerRepo(owner, repo string) string {
	if owner == "" || repo == "" {
		return ""
	}
	return owner + "/" + repo
}

func classifySuperPlaneArtifacts(artifacts []prArtifactMeta, repoOwner, repoName string, hasRepo bool) (merges, waste []prArtifactMeta) {
	for _, a := range artifacts {
		if hasRepo && (a.owner != repoOwner || a.repo != repoName) {
			continue
		}
		switch {
		case a.isMerged:
			merges = append(merges, a)
		case a.isClosedNM:
			waste = append(waste, a)
		}
	}
	return merges, waste
}

type peopleMerge struct {
	url      string
	mergedAt time.Time
}

type peopleCohortRequest struct {
	ctx           context.Context
	reg           *registry.Registry
	tx            *gorm.DB
	orgID         uuid.UUID
	factoryID     uuid.UUID
	integrationID string
	repoOwner     string
	repoName      string
	from          time.Time
	endExclusive  time.Time
}

// loadPeopleCohort loads GitHub People merges and subtracts SuperPlane PRs.
// Scan or search errors drop the People series and report failure so SuperPlane
// counts still return.
func loadPeopleCohort(req peopleCohortRequest) (hits []peopleMerge, hasPeople, failed bool) {
	known, err := listKnownSuperPlanePRs(req.tx, req.factoryID)
	if err != nil {
		log.WithContext(req.ctx).WithError(err).Warn("factory velocity: SuperPlane PR scan failed")
		return nil, false, true
	}

	found, err := searchPeopleMerges(
		req.ctx,
		req.reg,
		req.orgID,
		req.integrationID,
		req.repoOwner,
		req.repoName,
		req.from,
		req.endExclusive,
	)
	if err != nil {
		log.WithContext(req.ctx).WithError(err).Warn("factory velocity: GitHub people search failed")
		return nil, false, true
	}

	return subtractSuperPlaneHits(found, known), true, false
}

func searchPeopleMerges(
	ctx context.Context,
	reg *registry.Registry,
	orgID uuid.UUID,
	integrationID string,
	repoOwner, repoName string,
	from, endExclusive time.Time,
) ([]peopleMerge, error) {
	integrationUUID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid integration id")
	}

	client, err := newVelocityGitHubClient(reg, orgID, integrationUUID)
	if err != nil {
		return nil, err
	}

	fromDate, toDate := githubMergedDateRange(from, endExclusive)
	query := fmt.Sprintf(
		"repo:%s/%s is:pr is:merged merged:%s..%s",
		repoOwner,
		repoName,
		fromDate,
		toDate,
	)

	opts := &github.SearchOptions{
		Sort:        "updated",
		Order:       "desc",
		ListOptions: github.ListOptions{PerPage: velocitySearchPageSize},
	}

	var hits []peopleMerge
	for page := 0; page < velocitySearchMaxPages; page++ {
		result, resp, err := client.SearchIssues(ctx, query, opts)
		if err != nil {
			return nil, grpcerrors.FailedPrecondition(err, "failed to search GitHub for merged pull requests")
		}
		for _, issue := range result.Issues {
			if issue == nil || issue.HTMLURL == nil {
				continue
			}
			hits = append(hits, peopleMerge{
				url:      *issue.HTMLURL,
				mergedAt: issue.GetClosedAt().Time,
			})
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return hits, nil
}

// githubMergedDateRange returns inclusive UTC calendar dates that cover the
// local window. GitHub Search interprets YYYY-MM-DD as UTC. fillBuckets then
// drops hits that fall outside the local day buckets.
func githubMergedDateRange(from, endExclusive time.Time) (string, string) {
	lastInstant := endExclusive.Add(-time.Nanosecond)
	return from.UTC().Format("2006-01-02"), lastInstant.UTC().Format("2006-01-02")
}

func newVelocityGitHubClient(reg *registry.Registry, orgID, integrationID uuid.UUID) (*common.Client, error) {
	if reg == nil {
		return nil, grpcerrors.FailedPrecondition(nil, "integration registry is unavailable")
	}

	instance, err := models.FindIntegration(orgID, integrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "integration not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load integration")
	}

	if instance.AppName != "github" {
		return nil, grpcerrors.InvalidArgument(nil, "integration is not a GitHub integration")
	}
	if instance.State != models.IntegrationStateReady {
		return nil, grpcerrors.FailedPrecondition(nil, "integration is not ready")
	}

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		reg.Encryptor,
		reg,
		nil,
	)

	client, err := common.NewClient(integrationCtx, reg.HTTPContext())
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to build GitHub client")
	}
	return client, nil
}

func subtractSuperPlaneHits(hits []peopleMerge, superplane []prArtifactMeta) []peopleMerge {
	if len(hits) == 0 || len(superplane) == 0 {
		return hits
	}

	superSet := make(map[string]struct{}, len(superplane))
	for _, a := range superplane {
		superSet[normalizePRURL(a.url)] = struct{}{}
		if a.owner != "" && a.repo != "" && a.number != 0 {
			superSet[canonicalPRKey(a.owner, a.repo, a.number)] = struct{}{}
		}
	}

	out := make([]peopleMerge, 0, len(hits))
	for _, h := range hits {
		norm := normalizePRURL(h.url)
		if _, ok := superSet[norm]; ok {
			continue
		}
		owner, repo, number := parsePRURL(h.url)
		if owner != "" && repo != "" && number != 0 {
			if _, ok := superSet[canonicalPRKey(owner, repo, number)]; ok {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

func normalizePRURL(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	return strings.TrimSuffix(u, "/")
}

func canonicalPRKey(owner, repo string, number int) string {
	return fmt.Sprintf("%s/%s#%d", owner, repo, number)
}

func fillBuckets(buckets []dayBucket, sp, waste []prArtifactMeta, people []peopleMerge) {
	starts := make([]time.Time, len(buckets))
	for i := range buckets {
		starts[i] = buckets[i].start
	}
	locate := func(t time.Time) int {
		if len(buckets) == 0 {
			return -1
		}
		local := t.In(buckets[0].start.Location())
		idx := sort.Search(len(starts), func(i int) bool { return starts[i].After(local) }) - 1
		if idx < 0 || idx >= len(buckets) {
			return -1
		}
		if local.Before(buckets[idx].start) || !local.Before(buckets[idx].end) {
			return -1
		}
		return idx
	}

	for _, a := range sp {
		if a.mergedAt == nil {
			continue
		}
		if i := locate(*a.mergedAt); i >= 0 {
			buckets[i].superplaneMerged++
		}
	}
	for _, a := range waste {
		if a.closedAt == nil {
			continue
		}
		if i := locate(*a.closedAt); i >= 0 {
			buckets[i].waste++
		}
	}
	for _, h := range people {
		if i := locate(h.mergedAt); i >= 0 {
			buckets[i].peopleMerged++
		}
	}
}

func aggregateTotals(buckets []dayBucket, hasPeople bool) *pb.DescribeFactoryVelocityTotals {
	sp, people, waste := 0, 0, 0
	for _, b := range buckets {
		sp += b.superplaneMerged
		if hasPeople {
			people += b.peopleMerged
		}
		waste += b.waste
	}

	totals := &pb.DescribeFactoryVelocityTotals{
		SuperplaneMerged: int32(sp),
		PeopleMerged:     int32(people),
		Waste:            int32(waste),
	}
	totalMerged := sp + people
	if totalMerged > 0 && hasPeople {
		totals.SuperplaneSharePct = int32((sp * 100) / totalMerged)
	}
	spClosures := sp + waste
	if spClosures > 0 {
		totals.WastePct = int32((waste * 100) / spClosures)
	}
	return totals
}
