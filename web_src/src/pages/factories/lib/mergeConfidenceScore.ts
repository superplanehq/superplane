import type { FactoriesFactoryPullRequest, FactoriesFactoryPullRequestActivity } from "@/api-client";

import { pullRequestsSpanRepositories } from "./workOrderPullRequest";

import { workOrderCheckStatus, type WorkOrderCheckLevel, type WorkOrderCheckPresentation } from "./workOrderChecks";

/** Check keys written by the merge confidence automation. */
export const MERGE_CONFIDENCE_METRIC_KEYS = [
  "risk-review",
  "performance-review",
  "security-review",
  "drift-review",
  "reversibility-review",
] as const;

const LOWER_IS_BETTER_KEYS = new Set<string>(["risk-review", "drift-review"]);

/** Performance, security, and reversibility improve as the score rises. Other checks, including custom ones, improve as the score falls. */
const HIGHER_IS_BETTER_KEYS = new Set<string>(["performance-review", "security-review", "reversibility-review"]);

export type MergeConfidenceCardCheck = {
  key?: string;
  name?: string;
  score: number;
  maxScore: number;
};

type MergeConfidenceHeadlineInput = {
  key?: string;
  name?: string;
  score?: number;
  maxScore?: number;
};

/**
 * Same bands the server stores on the check. The board list has the score
 * only, so the card derives the level when it builds the hover list.
 */
export function mergeConfidenceCheckLevel(key: string | undefined, score: number): WorkOrderCheckLevel {
  const value = Math.round(score);
  if (key != null && HIGHER_IS_BETTER_KEYS.has(key)) {
    if (value <= 2) {
      return "critical";
    }
    if (value <= 3) {
      return "caution";
    }
    return "positive";
  }
  if (value >= 4) {
    return "critical";
  }
  if (value >= 3) {
    return "caution";
  }
  return "positive";
}

const LEVEL_SEVERITY: Record<WorkOrderCheckLevel, number> = {
  positive: 0,
  neutral: 1,
  caution: 2,
  critical: 3,
};

export const MERGE_CONFIDENCE_SCORE_NAME = "Merge confidence";

/** Light wash in the same hue as the score, so the panel matches the bar. */
const PANEL_TONE: Record<string, string> = {
  High: "[--frame-panel-bg:var(--color-emerald-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-emerald-500)_16%,var(--color-card))]",
  Healthy:
    "[--frame-panel-bg:var(--color-emerald-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-emerald-500)_16%,var(--color-card))]",
  Medium:
    "[--frame-panel-bg:var(--color-orange-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-orange-500)_16%,var(--color-card))]",
  Caution:
    "[--frame-panel-bg:var(--color-orange-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-orange-500)_16%,var(--color-card))]",
  Low: "[--frame-panel-bg:var(--color-red-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-red-500)_16%,var(--color-card))]",
  Critical:
    "[--frame-panel-bg:var(--color-red-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-red-500)_16%,var(--color-card))]",
  Neutral:
    "[--frame-panel-bg:var(--color-slate-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-slate-400)_16%,var(--color-card))]",
};

export function mergeConfidencePanelTone(check: Pick<WorkOrderCheckPresentation, "name" | "score" | "level">): string {
  return PANEL_TONE[workOrderCheckStatus(check).label] ?? PANEL_TONE.Neutral;
}

export interface MergeConfidenceCheckGroup {
  check: WorkOrderCheckPresentation;
  metrics: WorkOrderCheckPresentation[];
  assessmentFailed: boolean;
}

export function isMergeConfidenceMetric(check: { key?: string }): boolean {
  return check.key != null && (MERGE_CONFIDENCE_METRIC_KEYS as readonly string[]).includes(check.key);
}

export type MergeConfidenceCard = {
  score: number;
  maxScore: number;
  checks: MergeConfidenceCardCheck[];
};

/**
 * Board cards use the same headline as the task summary. A partial set still
 * scores. Planning checks and other custom checks stay out.
 */
export function mergeConfidenceHeadline(
  checks: readonly MergeConfidenceHeadlineInput[] | undefined,
): MergeConfidenceCard | undefined {
  const metrics = weakestPerKey(
    (checks ?? []).flatMap((check): MergeConfidenceCardCheck[] => {
      const score = check.score;
      const maxScore = check.maxScore ?? 0;
      if (!isMergeConfidenceMetric(check) || score == null || maxScore <= 0) {
        return [];
      }
      return [{ key: check.key, name: check.name, score, maxScore }];
    }),
  );
  if (metrics.length === 0) {
    return undefined;
  }
  const ordered = orderMergeConfidenceMetrics(metrics);
  const headline = headlineScore(ordered);
  if (headline.maxScore <= 0) {
    return undefined;
  }
  return { ...headline, checks: ordered };
}

/** Each pull request repository reports its own scores. The card lists each check once, with its weakest score. */
function weakestPerKey(metrics: MergeConfidenceCardCheck[]): MergeConfidenceCardCheck[] {
  const byKey = new Map<string, MergeConfidenceCardCheck>();
  for (const metric of metrics) {
    const key = metric.key ?? metric.name ?? "";
    const current = byKey.get(key);
    if (!current || confidenceRatio(metric) < confidenceRatio(current)) {
      byKey.set(key, metric);
    }
  }
  return [...byKey.values()];
}

/**
 * Merge confidence scores each repository in its own run. A check without a
 * repository applies to every pull request. GitHub repository names ignore case.
 */
export function checksForRepository<Check extends { repository?: string }>(
  checks: readonly Check[],
  repository: string | undefined,
): Check[] {
  const name = repository?.trim().toLowerCase();
  return checks.filter((check) => !check.repository || !name || check.repository.toLowerCase() === name);
}

export type MergeChecksByPullRequest = {
  /** Merge checks of each pull request repository, keyed by `pullRequestChecksKey`. */
  byRepository: Map<string, WorkOrderCheckPresentation[]>;
  /** Merge checks that belong to no listed pull request. */
  unattached: WorkOrderCheckPresentation[];
};

export function pullRequestChecksKey(repository: string | null | undefined): string {
  return repository?.trim().toLowerCase() ?? "";
}

/**
 * Each pull request row shows the merge checks of its repository. A check
 * without a repository attaches only when the task has one pull request
 * repository; with more, nothing says which pull request it scores.
 */
export function splitMergeChecksByPullRequest(
  checks: WorkOrderCheckPresentation[],
  pullRequestRepositories: readonly (string | null | undefined)[],
): MergeChecksByPullRequest {
  const metrics = consoleCheckList(checks)?.metrics ?? [];
  const keys = [...new Set(pullRequestRepositories.map(pullRequestChecksKey))];
  const byRepository = new Map<string, WorkOrderCheckPresentation[]>();
  const unattached: WorkOrderCheckPresentation[] = [];
  for (const metric of metrics) {
    const key = attachedPullRequestKey(metric, keys);
    if (key === undefined) {
      unattached.push(metric);
      continue;
    }
    byRepository.set(key, [...(byRepository.get(key) ?? []), metric]);
  }
  return { byRepository, unattached };
}

function attachedPullRequestKey(check: WorkOrderCheckPresentation, keys: string[]): string | undefined {
  const key = pullRequestChecksKey(check.repository);
  if (key) return keys.includes(key) ? key : undefined;
  return keys.length === 1 ? keys[0] : undefined;
}

/**
 * The task summary shows merge confidence only. Planning confidence and other
 * checks stay off this list once work has started.
 */
export function consoleCheckList(
  checks: WorkOrderCheckPresentation[],
  pullRequests: FactoriesFactoryPullRequest[] = [],
): MergeConfidenceCheckGroup | null {
  const metrics = orderMergeConfidenceMetrics(checks.filter(isMergeConfidenceMetric));
  if (metrics.length === 0) {
    return null;
  }
  const current = metrics.map((check) => currentMergeConfidenceCheck(check, pullRequests));
  const presented = current.map((result) => result.check);
  return {
    check: mergeConfidenceScore(presented),
    metrics: presented,
    assessmentFailed: current.some((result) => result.assessmentFailed),
  };
}

function currentMergeConfidenceCheck(
  check: WorkOrderCheckPresentation,
  pullRequests: FactoriesFactoryPullRequest[],
): { check: WorkOrderCheckPresentation; assessmentFailed: boolean } {
  if (!check.appId) return { check, assessmentFailed: false };
  const assessment = latestMergeConfidenceActivity(pullRequests, check.repository, (activity) => {
    return activity.run?.canvasId === check.appId;
  });
  const assessmentFailed = isFailedAssessment(assessment);
  if (!assessment || (check.runId === assessment.run?.id && !isStaleAssessment(assessment))) {
    return { check, assessmentFailed };
  }
  // A newer assessment invalidates older scores, even for the same commit.
  return {
    assessmentFailed,
    check: {
      ...check,
      score: 0,
      maxScore: 0,
      scoreSupplied: false,
      level: "neutral",
      summary: "No result was reported for the current assessment.",
      analysis: undefined,
      previousScore: undefined,
      recentScores: undefined,
    },
  };
}

type MergeConfidenceActivity = {
  run?: FactoriesFactoryPullRequestActivity["run"];
  title?: string;
  revision?: string;
  currentRevision?: string;
};

function latestMergeConfidenceActivity(
  pullRequests: FactoriesFactoryPullRequest[],
  repository: string | undefined,
  matches: (activity: MergeConfidenceActivity) => boolean,
): MergeConfidenceActivity | undefined {
  const key = pullRequestChecksKey(repository);
  return pullRequests
    .filter((pr) => !key || pullRequestChecksKey(pr.repository) === key)
    .flatMap((pr) => {
      const activities: FactoriesFactoryPullRequestActivity[] = pr.activities?.length ? pr.activities : (pr.runs ?? []);
      return activities.map((activity) => ({
        run: activity.run,
        title: activity.title,
        revision: activity.revision?.id,
        currentRevision: pr.currentRevision?.id,
      }));
    })
    .filter(matches)
    .sort((left, right) => (right.run?.createdAt ?? "").localeCompare(left.run?.createdAt ?? ""))[0];
}

function isStaleAssessment(assessment: MergeConfidenceActivity): boolean {
  return Boolean(
    assessment.currentRevision && assessment.revision && assessment.currentRevision !== assessment.revision,
  );
}

function isFailedAssessment(assessment: MergeConfidenceActivity | undefined): boolean {
  return Boolean(assessment && !isStaleAssessment(assessment) && assessment.run?.result === "RESULT_FAILED");
}

/**
 * A first assessment can fail before it saves a score. Without a score, no
 * check names the app canvas, so the server activity title identifies it.
 */
function unscoredAssessmentFailed(pullRequests: FactoriesFactoryPullRequest[]): boolean {
  const assessment = latestMergeConfidenceActivity(pullRequests, undefined, (activity) => {
    return activity.title?.trim() === MERGE_CONFIDENCE_SCORE_NAME;
  });
  return isFailedAssessment(assessment);
}

function mergeConfidenceAssessmentFailed(
  checks: WorkOrderCheckPresentation[],
  pullRequests: FactoriesFactoryPullRequest[],
): boolean {
  const assessment = consoleCheckList(checks, pullRequests);
  return assessment ? assessment.assessmentFailed : unscoredAssessmentFailed(pullRequests);
}

// Without multi-repo tasks, the panel keeps one review link and one merge
// confidence list.
export function mergeConfidencePanelFacts(
  checks: WorkOrderCheckPresentation[],
  pullRequests: FactoriesFactoryPullRequest[],
  multiRepo: boolean,
) {
  const currentChecks = consoleCheckList(checks, pullRequests)?.metrics ?? [];
  const assessmentFailed = mergeConfidenceAssessmentFailed(checks, pullRequests);
  if (!multiRepo) {
    // One task list reports the failure. Pull request rows do not repeat it.
    return {
      assessmentFailed,
      failedRepositories: new Set<string>(),
      reviewPullRequests: [],
      mergeChecks: { byRepository: null, unattached: currentChecks },
      showsPullRequestRepository: false,
    };
  }
  const failedRepositories = new Set(
    pullRequests
      .filter((pr) => mergeConfidenceAssessmentFailed(checksForRepository(checks, pr.repository), [pr]))
      .map((pr) => pullRequestChecksKey(pr.repository)),
  );
  return {
    assessmentFailed,
    failedRepositories,
    reviewPullRequests: pullRequests,
    mergeChecks: splitMergeChecksByPullRequest(
      currentChecks,
      pullRequests.map((pullRequest) => pullRequest.repository),
    ),
    showsPullRequestRepository: pullRequestsSpanRepositories(pullRequests),
  };
}

/** Higher is better. Risk and drift are flipped so 5 means the change is safe to merge. */
export function mergeConfidenceScore(metrics: WorkOrderCheckPresentation[]): WorkOrderCheckPresentation {
  const headline = headlineScore(metrics);
  return {
    id: "merge-confidence",
    key: "merge-confidence",
    name: MERGE_CONFIDENCE_SCORE_NAME,
    score: headline.score,
    maxScore: headline.maxScore,
    format: "fraction",
    level: weakestLevel(metrics),
  };
}

function headlineScore(metrics: MergeConfidenceHeadlineInput[]): { score: number; maxScore: number } {
  const ordered = orderMergeConfidenceMetrics(metrics);
  const maxScore = ordered[0]?.maxScore ?? 0;
  const weakest = ordered.reduce((lowest, check) => Math.min(lowest, confidenceRatio(check)), 1);
  return { score: maxScore > 0 ? Math.round(weakest * maxScore) : 0, maxScore };
}

function orderMergeConfidenceMetrics<Metric extends { key?: string }>(metrics: Metric[]): Metric[] {
  return [...metrics].sort((left, right) => metricIndex(left.key) - metricIndex(right.key));
}

function metricIndex(key: string | undefined): number {
  const index = MERGE_CONFIDENCE_METRIC_KEYS.indexOf(key as (typeof MERGE_CONFIDENCE_METRIC_KEYS)[number]);
  return index === -1 ? MERGE_CONFIDENCE_METRIC_KEYS.length : index;
}

function confidenceRatio(check: MergeConfidenceHeadlineInput): number {
  const maxScore = check.maxScore ?? 0;
  const score = check.score ?? 0;
  if (maxScore <= 0) {
    return 0;
  }
  const points = check.key != null && LOWER_IS_BETTER_KEYS.has(check.key) ? maxScore + 1 - score : score;
  return Math.min(maxScore, Math.max(0, points)) / maxScore;
}

function weakestLevel(metrics: WorkOrderCheckPresentation[]): WorkOrderCheckLevel {
  return metrics.reduce<WorkOrderCheckLevel>((weakest, check) => {
    return LEVEL_SEVERITY[check.level] > LEVEL_SEVERITY[weakest] ? check.level : weakest;
  }, "positive");
}
