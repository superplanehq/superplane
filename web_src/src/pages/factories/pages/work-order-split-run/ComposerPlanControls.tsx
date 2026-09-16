import type { ReactNode } from "react";
import { EyeOff, FileText, Sparkle } from "lucide-react";

import { CountButton } from "@/components/examples/c-button-38";
import { Badge } from "@/components/reui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

import { CONFIDENCE_SCORE_MAX, confidenceBandForScore, type ConfidenceBand } from "../../lib/confidenceScore";
import { CONFIDENCE_ANALYZING_TOOLTIP, ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { PlanChipStatus } from "./planChipStatus";

const FALLBACK_WHY = "The analysis scored how clear this work is.";

const SCORE_TONE: Record<ConfidenceBand, string> = {
  High: "text-success",
  Medium: "text-warning",
  Low: "text-destructive",
};

export function ComposerPlanStack({
  open,
  score,
  scoreSummary,
  isAnalyzing = false,
  canTogglePlan = true,
  planStatus,
  onToggle,
  actions,
}: {
  open: boolean;
  score?: number;
  scoreSummary?: string;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  actions?: ReactNode;
}) {
  const showScore = score != null || isAnalyzing;
  const showPlan = canTogglePlan;
  const showActions = Boolean(actions);
  if (!showScore && !showPlan && !showActions) {
    return null;
  }

  return (
    <div className="flex w-full min-w-0 shrink-0 flex-col" data-testid="split-run-intent-plan-updated">
      <div className="flex flex-wrap items-center gap-1.5" data-testid="split-run-intent-composer-chips">
        <ScoreChip score={score} scoreSummary={scoreSummary} isAnalyzing={isAnalyzing} />
        {canTogglePlan ? (
          <PlanToggle open={open} isAnalyzing={isAnalyzing} planStatus={planStatus} onToggle={onToggle} />
        ) : null}
        {actions ? (
          <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">{actions}</div>
        ) : null}
      </div>
    </div>
  );
}

function PlanToggle({
  open,
  isAnalyzing,
  planStatus,
  onToggle,
}: {
  open: boolean;
  isAnalyzing: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      aria-expanded={open}
      aria-pressed={open}
      aria-label={CREATE_WITH_AGENT_COPY.plan}
      onClick={onToggle}
    >
      <span className="relative inline-flex size-4 shrink-0 items-center justify-center">
        <FileText
          aria-hidden
          className={cn(
            "size-4 transition-all duration-300",
            open ? "scale-0 -rotate-90 opacity-0" : "scale-100 rotate-0 opacity-100",
          )}
        />
        <EyeOff
          aria-hidden
          className={cn(
            "absolute size-4 transition-all duration-300",
            open ? "scale-100 rotate-0 opacity-100" : "scale-0 rotate-90 opacity-0",
          )}
        />
      </span>
      {CREATE_WITH_AGENT_COPY.plan}
      <PlanStatusMark isAnalyzing={isAnalyzing} planStatus={planStatus} />
    </Button>
  );
}

function PlanStatusMark({ isAnalyzing, planStatus }: { isAnalyzing: boolean; planStatus?: PlanChipStatus }) {
  if (isAnalyzing) {
    return (
      <ConfidenceAnalyzingIndicator
        testId="split-run-intent-plan-chip-analyzing"
        showTooltip={false}
        decorative
        className="shrink-0"
      />
    );
  }
  if (planStatus === "updated") {
    return (
      <Badge
        variant="info-light"
        size="xs"
        aria-hidden
        data-testid="split-run-intent-plan-status"
        className="dark:border-warning/25 dark:bg-warning/15 dark:text-warning"
      >
        {CREATE_WITH_AGENT_COPY.planStatusUpdated}
      </Badge>
    );
  }
  if (planStatus === "ready") {
    return (
      <Badge variant="success-light" size="xs" aria-hidden data-testid="split-run-intent-plan-status">
        {CREATE_WITH_AGENT_COPY.planReady}
      </Badge>
    );
  }
  return null;
}

function ScoreChip({
  score,
  scoreSummary,
  isAnalyzing,
}: {
  score?: number;
  scoreSummary?: string;
  isAnalyzing: boolean;
}) {
  if (score == null && !isAnalyzing) {
    return null;
  }

  const showMatrix = isAnalyzing || score == null;
  const label =
    showMatrix || score == null
      ? CREATE_WITH_AGENT_COPY.clarity
      : `${CREATE_WITH_AGENT_COPY.clarity} ${score}/${CONFIDENCE_SCORE_MAX}`;
  const why = score == null || isAnalyzing ? CONFIDENCE_ANALYZING_TOOLTIP : scoreSummary?.trim() || FALLBACK_WHY;
  const countClassName =
    showMatrix || score == null
      ? "min-w-7 self-stretch py-0"
      : cn("font-semibold", SCORE_TONE[confidenceBandForScore(score)]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <CountButton
          type="button"
          size="sm"
          aria-label={label}
          data-testid={showMatrix ? undefined : "split-run-intent-composer-score"}
          countClassName={countClassName}
          count={
            showMatrix ? (
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
      </PopoverTrigger>
      <PopoverContent className="z-[80] w-80 gap-0 overflow-hidden p-0" align="start" side="top">
        <div className="border-b border-primary/10 bg-primary/5 p-2">
          <div className="flex items-center gap-2 font-semibold text-primary">
            <Sparkle className="size-4" aria-hidden="true" />
            <span>{CREATE_WITH_AGENT_COPY.clarity}</span>
          </div>
        </div>
        <div className="space-y-3 p-2">
          <p className="leading-relaxed text-muted-foreground" data-testid="split-run-intent-confidence-copy">
            {why}
          </p>
        </div>
      </PopoverContent>
    </Popover>
  );
}
