import { Fragment } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import {
  clampConfidenceScore,
  CLARITY_CHECK_NAME,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../lib/confidenceScore";
import { DRAFT_READINESS_SHORT_LABEL, draftReadiness, type DraftReadinessTone } from "../lib/draftReadiness";
import { PLANNING_REVIEW_SCORE_MAX, planningReviewLevel } from "../lib/planningReviewScore";
import { mergeConfidenceCheckLevel, type MergeConfidenceCardCheck } from "../lib/mergeConfidenceScore";
import { workOrderCheckDisplayName, workOrderCheckLevelLabel, type WorkOrderCheckLevel } from "../lib/workOrderChecks";
import { CheckStatusIcon } from "./CheckStatusIcon";
import { CheckStatusBars, ConfidenceMeter } from "./ConfidenceMeter";

const DOT_TONE: Record<DraftReadinessTone, string> = {
  analyzing: "text-[color:var(--status-draft-dot)]",
  pending: "text-[color:var(--status-draft-dot)]",
  blocked: "text-[color:var(--status-failed-dot)]",
  caution: "text-[color:var(--status-waiting-dot)]",
  ready: "text-[color:var(--status-completed-dot)]",
};

/** The 8px verdict dot. The refine strip and the board card share it. */
export function ReadinessDot({ tone, className }: { tone: DraftReadinessTone; className?: string }) {
  return (
    <span
      className={cn("inline-flex size-2 shrink-0 rounded-full bg-current", DOT_TONE[tone], className)}
      aria-hidden
    />
  );
}

type ScoreRow = { key: "clarity" | "confidence"; label: string; short: string; score?: number };

function scoreRows(
  clarity?: number,
  confidence?: number,
  visibility: { showClarity?: boolean; showConfidence?: boolean } = {},
): ScoreRow[] {
  const rows: ScoreRow[] = [];
  if (visibility.showClarity !== false) {
    rows.push({ key: "clarity", label: CLARITY_CHECK_NAME, short: "Clarity", score: clarity });
  }
  if (visibility.showConfidence !== false) {
    rows.push({ key: "confidence", label: CONFIDENCE_CHECK_NAME, short: "Confidence", score: confidence });
  }
  return rows;
}

const BADGE_TONE: Record<ConfidenceBand, string> = {
  High: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  Medium: "border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-400",
  Low: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
};

const BADGE_MUTED = "border-border bg-muted/40 text-muted-foreground";

/**
 * Board card scores: Clarity stays a pill. Confidence and Merge use the
 * same per-check bars as the plan card. Legacy scores keep their meter.
 */
export function CardScoreBadges({
  clarity,
  confidence,
  showClarity = true,
  showConfidence = true,
  reviewMetrics,
  className,
  testId,
}: {
  clarity?: number;
  confidence?: number;
  showClarity?: boolean;
  showConfidence?: boolean;
  reviewMetrics?: { key: string; name: string; score: number; level?: WorkOrderCheckLevel }[];
  className?: string;
  testId?: string;
}) {
  const hasReview = Boolean(reviewMetrics && reviewMetrics.length > 0);
  const scoreMax = hasReview ? PLANNING_REVIEW_SCORE_MAX : CONFIDENCE_SCORE_MAX;
  const readiness = draftReadiness({
    clarity: hasReview || !showClarity ? undefined : clarity,
    confidence: showConfidence ? confidence : undefined,
    scoreMax,
  });
  const rows = scoreRows(clarity, confidence, {
    showClarity: hasReview ? false : showClarity,
    showConfidence,
  });
  const breakdownMetrics = hasReview ? reviewMetrics : undefined;
  const heading = hasReview ? "Confidence in autonomous completion" : readiness.headline;
  const metricSpeech = (breakdownMetrics ?? []).map((metric) => {
    return `${metric.name}: ${workOrderCheckLevelLabel(metric.level ?? planningReviewLevel(metric.score))}`;
  });
  const speech = [
    heading,
    ...(metricSpeech.length > 0 ? metricSpeech : rows.map((row) => scoreSpeech(row, scoreMax))),
  ].join(". ");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          role="img"
          tabIndex={0}
          aria-label={speech}
          data-testid={testId}
          data-tone={readiness.tone}
          className={cn("pointer-events-auto inline-flex shrink-0 items-center gap-1", className)}
        >
          {rows.map((row) =>
            row.key === "confidence" ? (
              <ConfidenceChip
                key={row.key}
                score={row.score}
                max={scoreMax}
                checks={breakdownMetrics?.map((metric) => ({
                  id: metric.key,
                  level: metric.level ?? planningReviewLevel(metric.score),
                }))}
                testId={testId ? `${testId}-${row.key}` : undefined}
              />
            ) : (
              <span
                key={row.key}
                data-testid={testId ? `${testId}-${row.key}` : undefined}
                className={cn(
                  "inline-flex items-center gap-1 whitespace-nowrap rounded-full border px-1.5 py-0.5 text-[10px] font-medium leading-none",
                  row.score == null ? BADGE_MUTED : BADGE_TONE[confidenceBandForScore(clampConfidenceScore(row.score))],
                )}
              >
                <span>{row.short}</span>
                <span className="tabular-nums">{scoreText(row.score)}</span>
              </span>
            ),
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent>
        <span className="block font-medium" data-testid={testId ? `${testId}-verdict` : undefined}>
          {heading}
        </span>
        {breakdownMetrics ? (
          <span className="mt-1.5 flex flex-col gap-1.5">
            {breakdownMetrics.map((metric) => {
              const level = metric.level ?? planningReviewLevel(metric.score);
              return (
                <span key={metric.key} className="flex items-center gap-2">
                  <CheckStatusIcon
                    level={level}
                    label={workOrderCheckLevelLabel(level)}
                    className="[&>:first-child]:stroke-transparent"
                  />
                  <span>{metric.name}</span>
                </span>
              );
            })}
          </span>
        ) : null}
      </TooltipContent>
    </Tooltip>
  );
}

function ConfidenceChip({
  score,
  max,
  checks,
  testId,
}: {
  score?: number;
  max: number;
  checks?: { id: string; level: WorkOrderCheckLevel }[];
  testId?: string;
}) {
  const value = score == null ? undefined : clampConfidenceScore(score, max);
  return (
    <span
      data-testid={testId}
      className="inline-flex items-center gap-1 whitespace-nowrap text-[10px] font-medium leading-none text-muted-foreground"
    >
      <span>Confidence</span>
      {checks?.length ? (
        <CheckStatusBars checks={checks} testId={testId ? `${testId}-meter` : undefined} />
      ) : value == null ? (
        <span>–</span>
      ) : (
        <ConfidenceMeter
          score={value}
          max={max}
          showTooltip={false}
          decorative
          testId={testId ? `${testId}-meter` : undefined}
        />
      )}
    </span>
  );
}

const MERGE_METER_BARS = 3;

/**
 * Board footer mark. The Confidence prefix matches the backlog card.
 * Each check colors its own bar. Legacy aggregate scores use three steps.
 * Hover opens the same ordered check list on the right.
 */
export function MergeConfidenceChip({
  score,
  maxScore,
  checks = [],
  testId,
}: {
  score: number;
  maxScore: number;
  checks?: MergeConfidenceCardCheck[];
  testId?: string;
}) {
  const value = clampConfidenceScore(score, maxScore);
  const rows = mergeConfidenceRows(checks);
  const speech = rows.length
    ? `Merge confidence. ${rows.map((row) => `${row.name}: ${row.label}`).join(". ")}`
    : `Merge confidence ${value} of ${maxScore}`;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          role="img"
          aria-label={speech}
          tabIndex={0}
          className="pointer-events-auto inline-flex shrink-0 items-center gap-1 whitespace-nowrap text-[10px] font-medium leading-none text-muted-foreground"
        >
          <span>Confidence</span>
          {rows.length > 0 ? (
            <CheckStatusBars checks={rows} testId={testId ? `${testId}-meter` : undefined} />
          ) : (
            <ConfidenceMeter
              score={mergeMeterBars(value, maxScore)}
              max={MERGE_METER_BARS}
              showTooltip={false}
              decorative
              testId={testId ? `${testId}-meter` : undefined}
            />
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent side="right" data-testid={testId ? `${testId}-popup` : undefined}>
        <span className="block font-medium">Merge confidence</span>
        {rows.length > 0 ? (
          <span className="mt-1.5 flex flex-col gap-1.5">
            {rows.map((row) => (
              <span key={row.id} className="flex items-center gap-2">
                <CheckStatusIcon level={row.level} label={row.label} className="[&>:first-child]:stroke-transparent" />
                <span>{row.name}</span>
              </span>
            ))}
          </span>
        ) : null}
      </TooltipContent>
    </Tooltip>
  );
}

function mergeConfidenceRows(checks: MergeConfidenceCardCheck[]) {
  return checks.map((check) => {
    const name = workOrderCheckDisplayName(check);
    const level = mergeConfidenceCheckLevel(check.key, check.score);
    return { id: check.key || name, name, label: workOrderCheckLevelLabel(level), level };
  });
}

function mergeMeterBars(score: number, maxScore: number): number {
  if (maxScore <= MERGE_METER_BARS) {
    return score;
  }
  const band = confidenceBandForScore(score, maxScore);
  if (band === "High") {
    return MERGE_METER_BARS;
  }
  if (band === "Medium") {
    return MERGE_METER_BARS - 1;
  }
  return score <= 0 ? 0 : 1;
}

function scoreText(score?: number, max: number = CONFIDENCE_SCORE_MAX) {
  return score == null ? "–" : `${clampConfidenceScore(score, max)}/${max}`;
}

function scoreSpeech(row: ScoreRow, max: number = CONFIDENCE_SCORE_MAX) {
  if (row.score == null) {
    return `${row.label} no score yet`;
  }
  return `${row.label} ${clampConfidenceScore(row.score, max)} of ${max}`;
}

/**
 * Board card verdict: tone dot and one short word. The tooltip and the
 * accessible name carry the full headline and both scores.
 */
export function CardReadinessMark({
  clarity,
  confidence,
  className,
  testId,
}: {
  clarity?: number;
  confidence?: number;
  className?: string;
  testId?: string;
}) {
  const readiness = draftReadiness({ clarity, confidence });
  const rows = scoreRows(clarity, confidence);
  const speech = [readiness.headline, ...rows.map((row) => scoreSpeech(row))].join(". ");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          role="img"
          aria-label={speech}
          data-testid={testId}
          data-tone={readiness.tone}
          className={cn(
            "pointer-events-auto inline-flex shrink-0 items-center gap-1.5 text-[11px] leading-4 text-muted-foreground",
            className,
          )}
        >
          <ReadinessDot tone={readiness.tone} />
          <span className="whitespace-nowrap">{DRAFT_READINESS_SHORT_LABEL[readiness.tone]}</span>
        </span>
      </TooltipTrigger>
      <TooltipContent>
        <span className="block font-medium" data-testid={testId ? `${testId}-verdict` : undefined}>
          {readiness.headline}
        </span>
        <span className="mt-1 grid grid-cols-[auto_auto] gap-x-2 gap-y-0.5">
          {rows.map((row) => (
            <Fragment key={row.key}>
              <span>{row.label}</span>
              <span className="tabular-nums" data-testid={testId ? `${testId}-${row.key}` : undefined}>
                {scoreText(row.score)}
              </span>
            </Fragment>
          ))}
        </span>
      </TooltipContent>
    </Tooltip>
  );
}
