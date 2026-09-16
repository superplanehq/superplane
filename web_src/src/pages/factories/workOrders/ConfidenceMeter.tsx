import type { CSSProperties } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import {
  clampConfidenceScore,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../lib/confidenceScore";

import "./confidence-analyzing.css";

const FILLED_TONE: Record<ConfidenceBand, string> = {
  High: "bg-emerald-500",
  Medium: "bg-orange-500",
  Low: "bg-red-500",
};

export function ConfidenceMeter({
  score,
  className,
  testId,
  showTooltip = true,
  decorative = false,
  size = "sm",
}: {
  score: number;
  className?: string;
  testId?: string;
  showTooltip?: boolean;
  decorative?: boolean;
  size?: "sm" | "lg";
}) {
  const value = clampConfidenceScore(score);
  const band = confidenceBandForScore(value);
  const scoreLabel = `${value}/${CONFIDENCE_SCORE_MAX}`;
  const barClass = size === "lg" ? "h-2.5 w-2 rounded-[2px]" : "h-2 w-1.5 rounded-[1px]";

  const meter = (
    <span
      role={decorative ? undefined : "meter"}
      aria-hidden={decorative || undefined}
      aria-label={decorative ? undefined : CONFIDENCE_CHECK_NAME}
      aria-valuemin={decorative ? undefined : 0}
      aria-valuemax={decorative ? undefined : CONFIDENCE_SCORE_MAX}
      aria-valuenow={decorative ? undefined : value}
      aria-valuetext={decorative ? undefined : `${value} of ${CONFIDENCE_SCORE_MAX}`}
      data-testid={testId}
      className={cn("pointer-events-auto inline-flex items-center gap-0.5", size === "lg" && "gap-1", className)}
    >
      {Array.from({ length: CONFIDENCE_SCORE_MAX }, (_, index) => (
        <span
          key={index}
          data-filled={index < value ? "true" : "false"}
          className={cn("sp-meter-bar", barClass, index < value ? FILLED_TONE[band] : "bg-muted-foreground/25")}
        />
      ))}
    </span>
  );

  if (!showTooltip) {
    return meter;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{meter}</TooltipTrigger>
      <TooltipContent>
        <span>{CONFIDENCE_CHECK_NAME}</span>
        <span className="ml-1.5 tabular-nums">{scoreLabel}</span>
      </TooltipContent>
    </Tooltip>
  );
}

export const CONFIDENCE_ANALYZING_LABEL = "Analyzing";
export const CONFIDENCE_ANALYZING_TOOLTIP = "Agent is analyzing, refining, and planning this task.";

const MATRIX_DOTS = 16;
const MATRIX_CYCLE_MS = 1200;

/**
 * Round dot matrix while the Backlog automation still analyzes the task.
 * It takes the same slot as the score, so the card does not move when
 * the score arrives.
 */
export function ConfidenceAnalyzingIndicator({
  className,
  testId,
  showTooltip = true,
  decorative = false,
}: {
  className?: string;
  testId?: string;
  showTooltip?: boolean;
  decorative?: boolean;
}) {
  const indicator = (
    <span
      role={decorative ? undefined : "status"}
      aria-hidden={decorative || undefined}
      aria-label={decorative ? undefined : CONFIDENCE_ANALYZING_LABEL}
      data-testid={testId}
      className={cn("pointer-events-auto inline-flex items-center justify-center leading-none", className)}
    >
      <MatrixDotLoader />
    </span>
  );

  if (!showTooltip) {
    return indicator;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{indicator}</TooltipTrigger>
      <TooltipContent>
        <span>{CONFIDENCE_ANALYZING_TOOLTIP}</span>
      </TooltipContent>
    </Tooltip>
  );
}

function MatrixDotLoader() {
  return (
    <span className="t-matrix" data-variant="scan" aria-hidden>
      {Array.from({ length: MATRIX_DOTS }, (_, index) => (
        <i key={index} style={{ "--d": matrixScanDelay(index) } as CSSProperties} />
      ))}
    </span>
  );
}

function matrixScanDelay(index: number): number {
  return (index % 4) * (MATRIX_CYCLE_MS / 10);
}

