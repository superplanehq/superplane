import { Fragment, useEffect, useState, type CSSProperties } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { prefersReducedMotion } from "@/lib/streamWords";
import { cn } from "@/lib/utils";

import {
  clampConfidenceScore,
  CLARITY_CHECK_NAME,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../lib/confidenceScore";
import { draftReadiness } from "../lib/draftReadiness";

import "./confidence-analyzing.css";

const FILLED_TONE: Record<ConfidenceBand, string> = {
  High: "bg-emerald-500",
  Medium: "bg-orange-500",
  Low: "bg-red-500",
};

type MeterSize = "sm" | "lg" | "xs";

const BAR_CLASS: Record<MeterSize, string> = {
  lg: "h-2.5 w-2 rounded-[2px]",
  sm: "h-2 w-1.5 rounded-[1px]",
  xs: "h-[3px] w-1.5 rounded-[1px]",
};

function MeterBars({ value, size }: { value: number; size: MeterSize }) {
  const band = confidenceBandForScore(value);
  return (
    <>
      {Array.from({ length: CONFIDENCE_SCORE_MAX }, (_, index) => (
        <span
          key={index}
          data-filled={index < value ? "true" : "false"}
          className={cn("sp-meter-bar", BAR_CLASS[size], index < value ? FILLED_TONE[band] : "bg-muted-foreground/25")}
        />
      ))}
    </>
  );
}

export function ConfidenceMeter({
  score,
  label = CONFIDENCE_CHECK_NAME,
  className,
  testId,
  showTooltip = true,
  decorative = false,
  size = "sm",
}: {
  score: number;
  /** Check name read by screen readers and the tooltip. */
  label?: string;
  className?: string;
  testId?: string;
  showTooltip?: boolean;
  decorative?: boolean;
  size?: "sm" | "lg";
}) {
  const value = clampConfidenceScore(score);
  const scoreLabel = `${value}/${CONFIDENCE_SCORE_MAX}`;

  const meter = (
    <span
      role={decorative ? undefined : "meter"}
      aria-hidden={decorative || undefined}
      aria-label={decorative ? undefined : label}
      aria-valuemin={decorative ? undefined : 0}
      aria-valuemax={decorative ? undefined : CONFIDENCE_SCORE_MAX}
      aria-valuenow={decorative ? undefined : value}
      aria-valuetext={decorative ? undefined : `${value} of ${CONFIDENCE_SCORE_MAX}`}
      data-testid={testId}
      className={cn("pointer-events-auto inline-flex items-center gap-0.5", size === "lg" && "gap-1", className)}
    >
      <MeterBars value={value} size={size} />
    </span>
  );

  if (!showTooltip) {
    return meter;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{meter}</TooltipTrigger>
      <TooltipContent>
        <span>{label}</span>
        <span className="ml-1.5 tabular-nums">{scoreLabel}</span>
      </TooltipContent>
    </Tooltip>
  );
}

export const SCORE_PAIR_LABEL = "Clarity and Confidence";

/**
 * Two stacked 5-bar rows in the footprint of one meter: Clarity on top,
 * Confidence below. Board cards use this left of Start. One tooltip lists
 * both scores.
 */
export function ScorePairMeter({
  clarity,
  confidence,
  className,
  testId,
  showTooltip = true,
}: {
  clarity?: number;
  confidence?: number;
  className?: string;
  testId?: string;
  showTooltip?: boolean;
}) {
  const rows: Array<{ key: string; label: string; score?: number }> = [
    { key: "clarity", label: CLARITY_CHECK_NAME, score: clarity },
    { key: "confidence", label: CONFIDENCE_CHECK_NAME, score: confidence },
  ];
  const verdict = draftReadiness({ clarity, confidence }).headline;
  const meter = (
    <span
      role="group"
      aria-label={`${SCORE_PAIR_LABEL}. ${verdict}`}
      data-testid={testId}
      className={cn("pointer-events-auto inline-flex flex-col items-start justify-center gap-[3px]", className)}
    >
      {rows.map((row) => {
        const value = row.score == null ? undefined : clampConfidenceScore(row.score);
        return (
          <span
            key={row.key}
            role="meter"
            aria-label={row.label}
            aria-valuemin={0}
            aria-valuemax={CONFIDENCE_SCORE_MAX}
            aria-valuenow={value}
            aria-valuetext={value == null ? "No score yet" : `${value} of ${CONFIDENCE_SCORE_MAX}`}
            data-testid={testId ? `${testId}-${row.key}` : undefined}
            className="inline-flex items-center gap-0.5"
          >
            <MeterBars value={value ?? 0} size="xs" />
          </span>
        );
      })}
    </span>
  );

  if (!showTooltip) {
    return meter;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{meter}</TooltipTrigger>
      <TooltipContent>
        <span className="block font-medium" data-testid={testId ? `${testId}-verdict` : undefined}>
          {verdict}
        </span>
        <span className="mt-1 grid grid-cols-[auto_auto] gap-x-2 gap-y-0.5">
          {rows.map((row) => (
            <Fragment key={row.key}>
              <span>{row.label}</span>
              <span className="tabular-nums">
                {row.score == null ? "–" : `${clampConfidenceScore(row.score)}/${CONFIDENCE_SCORE_MAX}`}
              </span>
            </Fragment>
          ))}
        </span>
      </TooltipContent>
    </Tooltip>
  );
}

export const CONFIDENCE_ANALYZING_LABEL = "Analyzing";
export const CONFIDENCE_ANALYZING_TOOLTIP = "Agent is analyzing, refining, and planning this task.";

const CONFIDENCE_THINKING_STATES = ["Analyzing", "Refining", "Planning", "Checking"] as const;
const THINK_HOLD_MS = 2000;
const THINK_GAP_MS = 50;
const THINK_SWAP_MS = 150;

const MATRIX_DOTS = 16;
const MATRIX_CYCLE_MS = 1200;
const THINKING_SIZER = CONFIDENCE_THINKING_STATES.reduce((longest, state) =>
  state.length > longest.length ? state : longest,
);

/**
 * Round dot matrix while analysis still runs. The board card passes
 * `showThinkingStates` to cycle Analyzing, Refining, Planning, and
 * Checking next to the dots. Refine chips stay matrix-only.
 */
export function ConfidenceAnalyzingIndicator({
  className,
  testId,
  showTooltip = true,
  decorative = false,
  showThinkingStates = false,
}: {
  className?: string;
  testId?: string;
  showTooltip?: boolean;
  decorative?: boolean;
  showThinkingStates?: boolean;
}) {
  const indicator = (
    <span
      role={decorative ? undefined : "status"}
      aria-hidden={decorative || undefined}
      aria-label={decorative ? undefined : CONFIDENCE_ANALYZING_LABEL}
      data-testid={testId}
      className={cn(
        "pointer-events-auto inline-flex items-center leading-none",
        showThinkingStates ? "gap-1.5 text-[11px] text-muted-foreground" : "justify-center",
        className,
      )}
    >
      <MatrixDotLoader />
      {showThinkingStates ? <ThinkingStatesLabel /> : null}
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

function ThinkingStatesLabel() {
  const { current, outgoing, entering } = useThinkingStates();

  return (
    <span className="t-think">
      <span className="t-think-sizer" aria-hidden>
        {THINKING_SIZER}
      </span>
      {outgoing ? (
        <span className="t-think-text is-exit" data-text={outgoing} aria-hidden>
          {outgoing}
        </span>
      ) : null}
      <span className={cn("t-think-text", entering && "is-enter-start")} data-text={current} aria-hidden>
        {current}
      </span>
    </span>
  );
}

function useThinkingStates() {
  const [current, setCurrent] = useState<(typeof CONFIDENCE_THINKING_STATES)[number]>(CONFIDENCE_THINKING_STATES[0]);
  const [outgoing, setOutgoing] = useState<string | null>(null);
  const [entering, setEntering] = useState(false);

  useEffect(() => {
    let index = 0;
    let cancelled = false;
    const timers: number[] = [];

    const queue = (delay: number, work: () => void) => {
      timers.push(window.setTimeout(work, delay));
    };

    const tick = () => {
      queue(THINK_HOLD_MS, () => {
        if (cancelled) {
          return;
        }
        const next = CONFIDENCE_THINKING_STATES[(index + 1) % CONFIDENCE_THINKING_STATES.length];
        const leaving = CONFIDENCE_THINKING_STATES[index];
        if (prefersReducedMotion()) {
          index = (index + 1) % CONFIDENCE_THINKING_STATES.length;
          setCurrent(next);
          tick();
          return;
        }
        setOutgoing(leaving);
        queue(THINK_GAP_MS, () => {
          if (cancelled) {
            return;
          }
          index = (index + 1) % CONFIDENCE_THINKING_STATES.length;
          setCurrent(next);
          setEntering(true);
          requestAnimationFrame(() => {
            requestAnimationFrame(() => {
              if (!cancelled) {
                setEntering(false);
              }
            });
          });
          queue(THINK_SWAP_MS, () => {
            if (!cancelled) {
              setOutgoing(null);
            }
          });
          tick();
        });
      });
    };

    tick();
    return () => {
      cancelled = true;
      for (const timer of timers) {
        window.clearTimeout(timer);
      }
    };
  }, []);

  return { current, outgoing, entering };
}
