import { useState, type ReactNode } from "react";
import { EyeOff, FileText, Gauge, Sparkle, type LucideIcon } from "lucide-react";

import { CountButton } from "@/components/examples/c-button-38";
import { Badge } from "@/components/reui/badge";
import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { CONFIDENCE_SCORE_MAX, confidenceBandForScore, type ConfidenceBand } from "../../lib/confidenceScore";
import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import {
  composerSummaryBodies,
  resolveOpenSummary,
  summaryToggleTarget,
  type ComposerScore,
} from "./composerScoreSummary";
import type { PlanChipStatus } from "./planChipStatus";
import type { RefineSummaryKind } from "./refineLayoutPreference";

export type { ComposerScore } from "./composerScoreSummary";

const DRAWER_EASE = "duration-300 ease-[cubic-bezier(0.32,0.72,0,1)] motion-reduce:transition-none";

const SCORE_TONE: Record<ConfidenceBand, string> = {
  High: "text-success",
  Medium: "text-warning",
  Low: "text-destructive",
};

type ScoreChipCopy = {
  label: string;
  icon: LucideIcon;
  testId: string;
};

const SCORE_CHIP_COPY: Record<RefineSummaryKind, ScoreChipCopy> = {
  clarity: {
    label: CREATE_WITH_AGENT_COPY.clarity,
    icon: Sparkle,
    testId: "split-run-intent-composer-score",
  },
  confidence: {
    label: CREATE_WITH_AGENT_COPY.confidence,
    icon: Gauge,
    testId: "split-run-intent-composer-confidence",
  },
};

const SUMMARY_KINDS: readonly RefineSummaryKind[] = ["clarity", "confidence"];

/** Uncontrolled fallback for previews. The refine model passes the stored preference. */
function useOpenSummary(
  controlled: RefineSummaryKind | null | undefined,
  onToggle: ((kind: RefineSummaryKind) => void) | undefined,
): [RefineSummaryKind | null, (kind: RefineSummaryKind) => void] {
  const [local, setLocal] = useState<RefineSummaryKind | null>("clarity");
  const toggleLocal = (kind: RefineSummaryKind) => setLocal((current) => (current === kind ? null : kind));
  return [controlled === undefined ? local : controlled, onToggle ?? toggleLocal];
}

export function ComposerPlanStack({
  open,
  clarity,
  confidence,
  isAnalyzing = false,
  canTogglePlan = true,
  planStatus,
  onToggle,
  openSummary: openSummaryProp,
  onToggleSummary,
  actions,
}: {
  open: boolean;
  clarity?: ComposerScore;
  confidence?: ComposerScore;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  openSummary?: RefineSummaryKind | null;
  onToggleSummary?: (kind: RefineSummaryKind) => void;
  actions?: ReactNode;
}) {
  const [openSummary, toggleSummary] = useOpenSummary(openSummaryProp, onToggleSummary);
  const scores: Record<RefineSummaryKind, ComposerScore | undefined> = { clarity, confidence };
  const bodies = composerSummaryBodies(clarity, confidence);
  const shownSummary = resolveOpenSummary(openSummary, bodies);
  const drawerBody = shownSummary ? bodies[shownSummary] : undefined;
  const hasAnyBody = Boolean(bodies.clarity || bodies.confidence);
  return (
    <Frame dense className="w-full min-w-0" data-testid="split-run-intent-status-card">
      <FrameHeader className="px-3 py-1.5" data-testid="split-run-intent-plan-updated">
        <div className="flex flex-wrap items-center gap-1.5" data-testid="split-run-intent-composer-chips">
          {SUMMARY_KINDS.map((kind) => (
            <ScoreChip
              key={kind}
              kind={kind}
              score={scores[kind]?.score}
              isAnalyzing={isAnalyzing}
              expanded={Boolean(bodies[kind]) && shownSummary === kind}
              onToggle={
                bodies[kind] ? () => toggleSummary(summaryToggleTarget(kind, shownSummary, openSummary)) : undefined
              }
            />
          ))}
          {canTogglePlan ? (
            <PlanToggle open={open} isAnalyzing={isAnalyzing} planStatus={planStatus} onToggle={onToggle} />
          ) : null}
          {actions ? (
            <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">{actions}</div>
          ) : null}
        </div>
      </FrameHeader>
      {hasAnyBody ? <ScoreSummaryDrawer open={Boolean(drawerBody)} kind={shownSummary} body={drawerBody} /> : null}
    </Frame>
  );
}

function ScoreSummaryDrawer({ open, kind, body }: { open: boolean; kind: RefineSummaryKind | null; body?: string }) {
  return (
    <div
      className={cn("grid transition-[grid-template-rows]", DRAWER_EASE, open ? "grid-rows-[1fr]" : "grid-rows-[0fr]")}
      data-testid="split-run-intent-summary-drawer"
      data-state={open ? "open" : "closed"}
      data-kind={kind ?? undefined}
      aria-hidden={open ? undefined : true}
      inert={open ? undefined : true}
    >
      <div className="min-h-0 overflow-hidden">
        <FramePanel fit>
          <p className="text-[13px] leading-5 text-muted-foreground" data-testid="split-run-intent-summary-copy">
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
  kind,
  score,
  isAnalyzing,
  expanded = false,
  onToggle,
}: {
  kind: RefineSummaryKind;
  score?: number;
  isAnalyzing: boolean;
  expanded?: boolean;
  onToggle?: () => void;
}) {
  const copy = SCORE_CHIP_COPY[kind];
  const Icon = copy.icon;
  const showMatrix = isAnalyzing;
  const label = showMatrix || score == null ? copy.label : `${copy.label} ${score}/${CONFIDENCE_SCORE_MAX}`;
  const countClassName = showMatrix
    ? "min-w-7 self-stretch py-0"
    : score == null
      ? "text-muted-foreground"
      : cn("font-semibold", SCORE_TONE[confidenceBandForScore(score)]);

  const chip = (
    <CountButton
      type="button"
      size="sm"
      aria-label={label}
      aria-expanded={onToggle ? expanded : undefined}
      aria-pressed={onToggle ? expanded : undefined}
      onClick={onToggle}
      data-testid={showMatrix ? undefined : copy.testId}
      countClassName={countClassName}
      count={
        showMatrix ? (
          <ConfidenceAnalyzingIndicator
            testId={kind === "clarity" ? "split-run-intent-plan-analyzing" : undefined}
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
      <Icon aria-hidden="true" />
      {copy.label}
    </CountButton>
  );
  if (!showMatrix) {
    return chip;
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>{chip}</TooltipTrigger>
      <TooltipContent>{CREATE_WITH_AGENT_COPY.scoreAnalyzing}</TooltipContent>
    </Tooltip>
  );
}
