import type { ReactNode } from "react";
import { EyeOff, FileText } from "lucide-react";

import { Frame, FramePanel } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { liveDraftReadiness, type DraftReadiness, type DraftReadinessTone } from "../../lib/draftReadiness";
import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { ReadinessDot } from "../../workOrders/ReadinessMark";
import { ScoreEvidenceRow, type ScoreEvidenceValue } from "../../workOrders/ScoreEvidence";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { PlanChipStatus } from "./planChipStatus";

export type ComposerScore = ScoreEvidenceValue;

const SCORE_TEST_IDS = {
  clarity: "split-run-intent-composer-score",
  confidence: "split-run-intent-composer-confidence",
} as const;

/** The verdict text repeats the button for pending and ready, so only warnings keep it. */
const VERDICT_WITH_TEXT: readonly DraftReadinessTone[] = ["blocked", "caution"];

/** Keeps a strip item on one line until the panel is narrower than its content. */
const STRIP_ROW_ITEM_CLASS = "min-w-[min(100%,max-content)] max-w-full shrink-0";

/**
 * Decision strip above the refine composer. Row one is the verdict alone.
 * Row two is the evidence, both scores, then the controls in the order the
 * user reads them: Plan, model, Start. On a narrow strip the controls wrap
 * to the next line instead of being clipped by the panel.
 */
export function ComposerPlanStack({
  open,
  clarity,
  confidence,
  showClarity = true,
  showConfidence = true,
  isAnalyzing = false,
  canTogglePlan = true,
  planStatus,
  onToggle,
  actions,
  modelSelect,
}: {
  open: boolean;
  clarity?: ComposerScore;
  confidence?: ComposerScore;
  showClarity?: boolean;
  showConfidence?: boolean;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  actions?: ReactNode;
  modelSelect?: ReactNode;
}) {
  const readiness = liveDraftReadiness({
    clarity: showClarity ? clarity?.score : undefined,
    confidence: showConfidence ? confidence?.score : undefined,
    isAnalyzing,
  });
  const showControls = canTogglePlan || Boolean(modelSelect) || Boolean(actions);
  return (
    <Frame dense className="w-full min-w-0" data-testid="split-run-intent-status-card">
      <FramePanel fit className="flex flex-col gap-1.5 px-3 py-2" data-testid="split-run-intent-plan-updated">
        <Verdict readiness={readiness} />
        <div
          className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1"
          data-testid="split-run-intent-composer-chips"
        >
          <ScoreEvidenceRow
            className={STRIP_ROW_ITEM_CLASS}
            clarity={clarity}
            confidence={confidence}
            showClarity={showClarity}
            showConfidence={showConfidence}
            isAnalyzing={isAnalyzing}
            testIds={SCORE_TEST_IDS}
          />
          {showControls ? (
            <div
              className={cn("ml-auto flex flex-wrap items-center justify-end gap-1", STRIP_ROW_ITEM_CLASS)}
              data-testid="split-run-intent-settings"
            >
              {canTogglePlan ? (
                <PlanToggle open={open} isAnalyzing={isAnalyzing} planStatus={planStatus} onToggle={onToggle} />
              ) : null}
              {modelSelect}
              {actions ? <div className="ml-1 flex shrink-0 items-center">{actions}</div> : null}
            </div>
          ) : null}
        </div>
      </FramePanel>
    </Frame>
  );
}

function Verdict({ readiness }: { readiness: DraftReadiness }) {
  const showText = VERDICT_WITH_TEXT.includes(readiness.tone);
  return (
    <div
      key={readiness.headline}
      className="sp-stream-text flex min-w-0 flex-1 items-start gap-2"
      data-testid="split-run-intent-verdict"
      data-tone={readiness.tone}
    >
      <VerdictMark tone={readiness.tone} />
      <div className="min-w-0">
        <p className="text-[13px] leading-5 font-medium text-foreground">{readiness.headline}</p>
        {showText ? <p className="text-[12px] leading-4 text-muted-foreground">{readiness.text}</p> : null}
      </div>
    </div>
  );
}

function VerdictMark({ tone }: { tone: DraftReadinessTone }) {
  if (tone === "analyzing") {
    return (
      <ConfidenceAnalyzingIndicator
        testId="split-run-intent-verdict-analyzing"
        showTooltip={false}
        decorative
        className="mt-1 shrink-0"
      />
    );
  }
  return <ReadinessDot tone={tone} className="mt-1.5" />;
}

function PlanToggle({
  open,
  isAnalyzing,
  planStatus,
  onToggle,
  className,
}: {
  open: boolean;
  isAnalyzing: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  className?: string;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-expanded={onToggle ? open : undefined}
      aria-pressed={onToggle ? open : undefined}
      aria-label={CREATE_WITH_AGENT_COPY.plan}
      onClick={onToggle}
      className={cn("text-muted-foreground hover:text-foreground", className)}
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

/** The matrix while the agent writes. An unread dot when the plan changed and the pane is closed. */
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
  if (planStatus !== "updated") {
    return null;
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          role="status"
          aria-label={CREATE_WITH_AGENT_COPY.planUpdated}
          data-testid="split-run-intent-plan-status"
          className="inline-flex size-2 shrink-0 rounded-full bg-[color:var(--status-waiting-dot)]"
        />
      </TooltipTrigger>
      <TooltipContent>{CREATE_WITH_AGENT_COPY.planUpdated}</TooltipContent>
    </Tooltip>
  );
}
