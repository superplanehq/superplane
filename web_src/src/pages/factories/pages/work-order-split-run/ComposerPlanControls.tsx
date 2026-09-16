import type { ReactNode } from "react";
import { EyeOff, FileText, Sparkle } from "lucide-react";

import { CountButton } from "@/components/examples/c-button-38";
import { Badge } from "@/components/reui/badge";
import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { CONFIDENCE_SCORE_MAX, confidenceBandForScore, type ConfidenceBand } from "../../lib/confidenceScore";
import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
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
  const showScore = score != null;
  const showPlan = canTogglePlan;
  const showActions = Boolean(actions);
  const body = score == null ? undefined : scoreSummary?.trim() || FALLBACK_WHY;
  if (!showScore && !showPlan && !showActions) {
    return null;
  }

  const chips = (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="split-run-intent-composer-chips">
      <ScoreChip score={score} isAnalyzing={isAnalyzing} />
      {canTogglePlan ? (
        <PlanToggle open={open} isAnalyzing={isAnalyzing} planStatus={planStatus} onToggle={onToggle} />
      ) : null}
      {actions ? (
        <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">{actions}</div>
      ) : null}
    </div>
  );

  if (!body) {
    return (
      <div className="flex w-full min-w-0 shrink-0 flex-col" data-testid="split-run-intent-plan-updated">
        {chips}
      </div>
    );
  }

  return (
    <Frame dense className="w-full min-w-0" data-testid="split-run-intent-status-card">
      <FrameHeader className="px-3 py-1.5" data-testid="split-run-intent-plan-updated">
        {chips}
      </FrameHeader>
      <FramePanel fit>
        <p className="text-[13px] leading-5 text-muted-foreground" data-testid="split-run-intent-confidence-copy">
          {body}
        </p>
      </FramePanel>
    </Frame>
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

function ScoreChip({ score, isAnalyzing }: { score?: number; isAnalyzing: boolean }) {
  if (score == null) {
    return null;
  }

  const showMatrix = isAnalyzing;
  const label = showMatrix
    ? CREATE_WITH_AGENT_COPY.clarity
    : `${CREATE_WITH_AGENT_COPY.clarity} ${score}/${CONFIDENCE_SCORE_MAX}`;
  const countClassName = showMatrix
    ? "min-w-7 self-stretch py-0"
    : cn("font-semibold", SCORE_TONE[confidenceBandForScore(score)]);

  return (
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
  );
}
