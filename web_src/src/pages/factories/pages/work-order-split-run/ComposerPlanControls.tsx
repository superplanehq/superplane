import type { ReactNode } from "react";
import { PanelRight, PanelRightClose, Sparkle } from "lucide-react";

import { CountButton } from "@/components/examples/c-button-38";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { CONFIDENCE_SCORE_MAX, confidenceBandForScore, type ConfidenceBand } from "../../lib/confidenceScore";
import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";

const SCORE_TONE: Record<ConfidenceBand, string> = {
  High: "text-success",
  Medium: "text-warning",
  Low: "text-destructive",
};

export function ComposerPlanStack({
  open,
  score,
  isAnalyzing = false,
  canTogglePlan = true,
  onToggle,
  actions,
}: {
  open: boolean;
  score?: number;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  onToggle?: () => void;
  actions?: ReactNode;
}) {
  return (
    <div className="flex shrink-0 flex-col bg-background pb-1 pt-1" data-testid="split-run-intent-plan-updated">
      <div className="flex flex-wrap items-center gap-1.5" data-testid="split-run-intent-composer-chips">
        <ScoreChip score={score} isAnalyzing={isAnalyzing} />
        {canTogglePlan ? <PlanToggle open={open} onToggle={onToggle} /> : null}
        {actions ? (
          <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">{actions}</div>
        ) : null}
      </div>
    </div>
  );
}

function PlanToggle({ open, onToggle }: { open: boolean; onToggle?: () => void }) {
  const label = open ? CREATE_WITH_AGENT_COPY.hidePlan : CREATE_WITH_AGENT_COPY.showPlan;

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      aria-expanded={open}
      aria-pressed={open}
      aria-label={label}
      onClick={onToggle}
    >
      <span className="relative inline-flex size-4 shrink-0 items-center justify-center">
        <PanelRight
          aria-hidden
          className={cn(
            "size-4 transition-all duration-300",
            open ? "scale-0 -rotate-90 opacity-0" : "scale-100 rotate-0 opacity-100",
          )}
        />
        <PanelRightClose
          aria-hidden
          className={cn(
            "absolute size-4 transition-all duration-300",
            open ? "scale-100 rotate-0 opacity-100" : "scale-0 rotate-90 opacity-0",
          )}
        />
      </span>
      {label}
    </Button>
  );
}

function ScoreChip({ score, isAnalyzing }: { score?: number; isAnalyzing: boolean }) {
  if (score == null && !isAnalyzing) {
    return null;
  }

  const label =
    score == null
      ? CREATE_WITH_AGENT_COPY.clarity
      : `${CREATE_WITH_AGENT_COPY.clarity} ${score}/${CONFIDENCE_SCORE_MAX}`;

  return (
    <CountButton
      type="button"
      size="sm"
      tabIndex={-1}
      aria-label={label}
      data-testid={score == null ? undefined : "split-run-intent-composer-score"}
      className="pointer-events-none"
      countClassName={score == null ? undefined : cn("font-semibold", SCORE_TONE[confidenceBandForScore(score)])}
      count={
        score == null ? (
          <ConfidenceAnalyzingIndicator
            testId="split-run-intent-plan-analyzing"
            showTooltip={false}
            decorative
            className="shrink-0"
          />
        ) : (
          `${score}/${CONFIDENCE_SCORE_MAX}`
        )
      }
    >
      <Sparkle aria-hidden="true" />
      {CREATE_WITH_AGENT_COPY.clarity}
    </CountButton>
  );
}
