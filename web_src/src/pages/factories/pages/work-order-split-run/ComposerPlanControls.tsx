import { useState, type ReactNode } from "react";
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
const DRAWER_EASE = "duration-300 ease-[cubic-bezier(0.32,0.72,0,1)] motion-reduce:transition-none";

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
  summaryOpen: summaryOpenProp,
  onToggleSummary,
  actions,
}: {
  open: boolean;
  score?: number;
  scoreSummary?: string;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  summaryOpen?: boolean;
  onToggleSummary?: () => void;
  actions?: ReactNode;
}) {
  const [uncontrolledSummaryOpen, setUncontrolledSummaryOpen] = useState(true);
  const summaryOpen = summaryOpenProp ?? uncontrolledSummaryOpen;
  const toggleSummary = onToggleSummary ?? (() => setUncontrolledSummaryOpen((current) => !current));
  const body = score == null ? undefined : scoreSummary?.trim() || FALLBACK_WHY;
  const chips = (
    <ComposerChipRow
      open={open}
      score={score}
      isAnalyzing={isAnalyzing}
      canTogglePlan={canTogglePlan}
      planStatus={planStatus}
      summaryOpen={summaryOpen}
      body={body}
      toggleSummary={toggleSummary}
      onToggle={onToggle}
      actions={actions}
    />
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
      <ClaritySummaryDrawer open={summaryOpen} body={body} />
    </Frame>
  );
}

function ComposerChipRow({
  open,
  score,
  isAnalyzing,
  canTogglePlan,
  planStatus,
  summaryOpen,
  body,
  toggleSummary,
  onToggle,
  actions,
}: {
  open: boolean;
  score?: number;
  isAnalyzing: boolean;
  canTogglePlan: boolean;
  planStatus?: PlanChipStatus;
  summaryOpen: boolean;
  body?: string;
  toggleSummary: () => void;
  onToggle?: () => void;
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="split-run-intent-composer-chips">
      <ScoreChip
        score={score}
        isAnalyzing={isAnalyzing}
        expanded={Boolean(body) && summaryOpen}
        onToggle={body ? toggleSummary : undefined}
      />
      {canTogglePlan ? (
        <PlanToggle open={open} isAnalyzing={isAnalyzing} planStatus={planStatus} onToggle={onToggle} />
      ) : null}
      {actions ? (
        <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">{actions}</div>
      ) : null}
    </div>
  );
}

function ClaritySummaryDrawer({ open, body }: { open: boolean; body: string }) {
  return (
    <div
      className={cn("grid transition-[grid-template-rows]", DRAWER_EASE, open ? "grid-rows-[1fr]" : "grid-rows-[0fr]")}
      data-testid="split-run-intent-confidence-drawer"
      data-state={open ? "open" : "closed"}
      aria-hidden={open ? undefined : true}
      inert={open ? undefined : true}
    >
      <div className="min-h-0 overflow-hidden">
        <FramePanel fit>
          <p className="text-[13px] leading-5 text-muted-foreground" data-testid="split-run-intent-confidence-copy">
            {body}
          </p>
        </FramePanel>
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
      aria-expanded={onToggle ? open : undefined}
      aria-pressed={onToggle ? open : undefined}
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
  isAnalyzing,
  expanded = false,
  onToggle,
}: {
  score?: number;
  isAnalyzing: boolean;
  expanded?: boolean;
  onToggle?: () => void;
}) {
  const showMatrix = isAnalyzing;
  const label =
    showMatrix || score == null
      ? CREATE_WITH_AGENT_COPY.clarity
      : `${CREATE_WITH_AGENT_COPY.clarity} ${score}/${CONFIDENCE_SCORE_MAX}`;
  const countClassName = showMatrix
    ? "min-w-7 self-stretch py-0"
    : score == null
      ? "text-muted-foreground"
      : cn("font-semibold", SCORE_TONE[confidenceBandForScore(score)]);

  return (
    <CountButton
      type="button"
      size="sm"
      aria-label={label}
      aria-expanded={onToggle ? expanded : undefined}
      aria-pressed={onToggle ? expanded : undefined}
      onClick={onToggle}
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
        ) : score == null ? (
          "–"
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
