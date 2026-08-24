import { cn } from "@/lib/utils";

import {
  clampConfidenceScore,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../lib/confidenceScore";

const FILLED_TONE: Record<ConfidenceBand, string> = {
  High: "bg-emerald-500",
  Medium: "bg-orange-500",
  Low: "bg-red-500",
};

export function ConfidenceMeter({ score, className, testId }: { score: number; className?: string; testId?: string }) {
  const value = clampConfidenceScore(score);
  const band = confidenceBandForScore(value);

  return (
    <span
      role="meter"
      aria-label="Confidence score"
      aria-valuemin={0}
      aria-valuemax={CONFIDENCE_SCORE_MAX}
      aria-valuenow={value}
      aria-valuetext={`${value} of ${CONFIDENCE_SCORE_MAX}`}
      data-testid={testId}
      className={cn("inline-flex items-center gap-0.5", className)}
    >
      {Array.from({ length: CONFIDENCE_SCORE_MAX }, (_, index) => (
        <span
          key={index}
          data-filled={index < value ? "true" : "false"}
          className={cn("h-2 w-1.5 rounded-[1px]", index < value ? FILLED_TONE[band] : "bg-muted-foreground/25")}
        />
      ))}
    </span>
  );
}
