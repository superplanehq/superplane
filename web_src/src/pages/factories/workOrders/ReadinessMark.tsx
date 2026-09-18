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

function scoreRows(clarity?: number, confidence?: number): ScoreRow[] {
  return [
    { key: "clarity", label: CLARITY_CHECK_NAME, short: "Clarity", score: clarity },
    { key: "confidence", label: CONFIDENCE_CHECK_NAME, short: "Confidence", score: confidence },
  ];
}

const BADGE_TONE: Record<ConfidenceBand, string> = {
  High: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  Medium: "border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-400",
  Low: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
};

const BADGE_MUTED = "border-border bg-muted/40 text-muted-foreground";

/**
 * Board card scores as two light badges, name and number, tinted by band.
 * Same pill style as the Agent question chip. The tooltip carries the
 * verdict headline.
 */
export function CardScoreBadges({
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
          className={cn("pointer-events-auto inline-flex shrink-0 items-center gap-1", className)}
        >
          {rows.map((row) => (
            <span
              key={row.key}
              data-testid={testId ? `${testId}-${row.key}` : undefined}
              className={cn(
                "inline-flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-[10px] font-medium leading-none",
                row.score == null ? BADGE_MUTED : BADGE_TONE[confidenceBandForScore(clampConfidenceScore(row.score))],
              )}
            >
              <span>{row.short}</span>
              <span className="tabular-nums">{row.score == null ? "–" : clampConfidenceScore(row.score)}</span>
            </span>
          ))}
        </span>
      </TooltipTrigger>
      <TooltipContent>
        <span className="block font-medium" data-testid={testId ? `${testId}-verdict` : undefined}>
          {readiness.headline}
        </span>
      </TooltipContent>
    </Tooltip>
  );
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
