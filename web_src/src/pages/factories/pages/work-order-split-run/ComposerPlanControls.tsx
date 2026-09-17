import type { ReactNode } from "react";
import { EyeOff, FileText } from "lucide-react";

import { Frame, FramePanel } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import {
  DRAFT_READINESS_NOTES,
  draftReadiness,
  type DraftReadiness,
  type DraftReadinessTone,
} from "../../lib/draftReadiness";
import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { ScoreEvidenceRow, type ScoreEvidenceValue } from "../../workOrders/ScoreEvidence";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { PlanChipStatus } from "./planChipStatus";

export type ComposerScore = ScoreEvidenceValue;

const SCORE_TEST_IDS = {
  clarity: "split-run-intent-composer-score",
  confidence: "split-run-intent-composer-confidence",
} as const;

const VERDICT_DOT: Record<DraftReadinessTone, string> = {
  analyzing: "text-[color:var(--status-draft-dot)]",
  pending: "text-[color:var(--status-draft-dot)]",
  blocked: "text-[color:var(--status-failed-dot)]",
  caution: "text-[color:var(--status-waiting-dot)]",
  ready: "text-[color:var(--status-completed-dot)]",
};

/** The verdict text repeats the button for pending and ready, so only warnings keep it. */
const VERDICT_WITH_TEXT: readonly DraftReadinessTone[] = ["blocked", "caution"];

/** While the agent works the strip says so, even when older scores exist. */
function stripReadiness(clarity?: ComposerScore, confidence?: ComposerScore, isAnalyzing = false): DraftReadiness {
  if (isAnalyzing) {
    return { tone: "analyzing", ...DRAFT_READINESS_NOTES.analyzing };
  }
  return draftReadiness({ clarity: clarity?.score, confidence: confidence?.score });
}

/**
 * Decision strip above the refine composer. Row one is the verdict and the
 * draft actions. Row two is the evidence: both scores and the Plan toggle.
 */
export function ComposerPlanStack({
  open,
  clarity,
  confidence,
  isAnalyzing = false,
  canTogglePlan = true,
  planStatus,
  onToggle,
  actions,
}: {
  open: boolean;
  clarity?: ComposerScore;
  confidence?: ComposerScore;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  actions?: ReactNode;
}) {
  const readiness = stripReadiness(clarity, confidence, isAnalyzing);
  return (
    <Frame dense className="w-full min-w-0" data-testid="split-run-intent-status-card">
      <FramePanel fit className="flex flex-col gap-1.5 px-3 py-2" data-testid="split-run-intent-plan-updated">
        <div className="flex min-w-0 items-center gap-3">
          <Verdict readiness={readiness} />
          {actions ? (
            <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">{actions}</div>
          ) : null}
        </div>
        <div className="flex min-w-0 items-center gap-2" data-testid="split-run-intent-composer-chips">
          <ScoreEvidenceRow
            clarity={clarity}
            confidence={confidence}
            isAnalyzing={isAnalyzing}
            testIds={SCORE_TEST_IDS}
          />
          {canTogglePlan ? (
            <PlanToggle
              open={open}
              isAnalyzing={isAnalyzing}
              planStatus={planStatus}
              onToggle={onToggle}
              className="ml-auto"
            />
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
  return (
    <span className={cn("mt-1.5 inline-flex size-2 shrink-0 rounded-full bg-current", VERDICT_DOT[tone])} aria-hidden />
  );
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
