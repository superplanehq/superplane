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
import { planningReviewLevel } from "../lib/planningReviewScore";
import { workOrderCheckStatus, type WorkOrderCheckLevel } from "../lib/workOrderChecks";
import { ConfidenceMeter } from "./ConfidenceMeter";

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

const NUMBER_TONE: Record<ConfidenceBand, string> = {
  High: "text-success",
  Medium: "text-warning",
  Low: "text-destructive",
};

/**
 * Tooltip uses `bg-foreground`, so it is dark in light mode and light in
 * dark mode. Invert the usual card tones.
 */
const TOOLTIP_RESULT_TONE: Record<WorkOrderCheckLevel, string> = {
  positive: "text-emerald-300 dark:text-emerald-700",
  neutral: "text-slate-300 dark:text-slate-700",
  caution: "text-amber-300 dark:text-amber-700",
  critical: "text-red-300 dark:text-red-700",
};

/**
 * Board card scores: Clarity stays a pill. Confidence uses the same step
 * meter as the plan card. The tooltip carries the verdict headline.
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
  const readiness = draftReadiness({
    clarity: hasReview || !showClarity ? undefined : clarity,
    confidence: showConfidence ? confidence : undefined,
  });
  const rows = scoreRows(clarity, confidence, {
    showClarity: hasReview ? false : showClarity,
    showConfidence,
  });
  const metricSpeech = (reviewMetrics ?? []).map((metric) => {
    const status = reviewMetricStatus(metric);
    return `${metric.name} ${status.label}`;
  });
  const speech = [readiness.headline, ...(metricSpeech.length > 0 ? metricSpeech : rows.map(scoreSpeech))].join(". ");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          role="img"
          aria-label={speech}
          data-testid={testId}
          data-tone={readiness.tone}
          className={cn("pointer-events-auto inline-flex shrink-0 items-center gap-1", className)}
        >
          {rows.map((row) =>
            row.key === "confidence" ? (
              <ConfidenceChip key={row.key} score={row.score} testId={testId ? `${testId}-${row.key}` : undefined} />
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
          {readiness.headline}
        </span>
        {reviewMetrics && reviewMetrics.length > 0 ? (
          <span className="mt-1 grid grid-cols-[auto_auto] gap-x-2 gap-y-0.5">
            {reviewMetrics.map((metric) => {
              const status = reviewMetricStatus(metric);
              return (
                <Fragment key={metric.key}>
                  <span>{metric.name}</span>
                  <span className={tooltipResultTone(metric)}>{status.label}</span>
                </Fragment>
              );
            })}
          </span>
        ) : null}
      </TooltipContent>
    </Tooltip>
  );
}

function ConfidenceChip({ score, testId }: { score?: number; testId?: string }) {
  const value = score == null ? undefined : clampConfidenceScore(score);
  return (
    <span
      data-testid={testId}
      className="inline-flex items-center gap-1 whitespace-nowrap text-[10px] font-medium leading-none text-muted-foreground"
    >
      <span>Confidence</span>
      {value == null ? (
        <span>–</span>
      ) : (
        <>
          <ConfidenceMeter
            score={value}
            showTooltip={false}
            decorative
            testId={testId ? `${testId}-meter` : undefined}
          />
          <span className={cn("tabular-nums", NUMBER_TONE[confidenceBandForScore(value)])}>{value}</span>
        </>
      )}
    </span>
  );
}

function reviewMetricStatus(metric: { key: string; name: string; score: number; level?: WorkOrderCheckLevel }) {
  return workOrderCheckStatus({
    name: metric.name,
    key: metric.key,
    score: metric.score,
    level: metric.level ?? planningReviewLevel(metric.score),
  });
}

function tooltipResultTone(metric: { score: number; level?: WorkOrderCheckLevel }) {
  return TOOLTIP_RESULT_TONE[metric.level ?? planningReviewLevel(metric.score)];
}

function scoreText(score?: number) {
  return score == null ? "–" : `${clampConfidenceScore(score)}/${CONFIDENCE_SCORE_MAX}`;
}

function scoreSpeech(row: ScoreRow) {
  if (row.score == null) {
    return `${row.label} no score yet`;
  }
  return `${row.label} ${clampConfidenceScore(row.score)} of ${CONFIDENCE_SCORE_MAX}`;
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
  const speech = [readiness.headline, ...rows.map(scoreSpeech)].join(". ");

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
