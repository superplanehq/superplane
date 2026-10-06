import { EyeOff, FileText } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Frame, FramePanel } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { ReadinessDot } from "../../workOrders/ReadinessMark";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ScoreEvidenceRow, type ScoreEvidenceValue } from "../../workOrders/ScoreEvidence";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { PlanChipStatus } from "./planChipStatus";
import type { ComposerCreditVerdict } from "./splitRunFooter";

export type ComposerScore = ScoreEvidenceValue;

const SCORE_TEST_IDS = {
  clarity: "split-run-intent-composer-score",
  confidence: "split-run-intent-composer-confidence",
} as const;

/** The latest plan and its assessment stay together in the conversation. */
export function PlanningReview({
  open,
  title,
  clarity,
  confidence,
  reviewMetrics,
  showClarity = true,
  showConfidence = true,
  isAnalyzing = false,
  canTogglePlan = true,
  planStatus,
  onToggle,
  creditVerdict,
}: {
  open: boolean;
  title?: string;
  clarity?: ComposerScore;
  confidence?: ComposerScore;
  reviewMetrics?: WorkOrderCheckPresentation[];
  showClarity?: boolean;
  showConfidence?: boolean;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  creditVerdict?: ComposerCreditVerdict;
}) {
  const analyzing = isAnalyzing && !creditVerdict;
  return (
    <Frame
      dense
      className="my-3 w-full min-w-0"
      role="region"
      aria-label={canTogglePlan ? "Plan" : "Planning assessment"}
      data-testid="split-run-intent-status-card"
    >
      <FramePanel
        fit
        className="flex flex-wrap items-center gap-3 px-3.5 py-3"
        data-testid="split-run-intent-plan-updated"
      >
        {creditVerdict ? (
          <PlanningCreditNotice verdict={creditVerdict} />
        ) : (
          <PlanningHeading title={title} hasPlan={canTogglePlan} analyzing={analyzing} />
        )}
        <div
          className="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2"
          data-testid="split-run-intent-composer-chips"
        >
          <ScoreEvidenceRow
            clarity={clarity}
            confidence={confidence}
            showClarity={showClarity}
            showConfidence={showConfidence}
            isAnalyzing={analyzing}
            testIds={SCORE_TEST_IDS}
          />
          {canTogglePlan ? (
            <PlanToggle open={open} isAnalyzing={analyzing} planStatus={planStatus} onToggle={onToggle} />
          ) : null}
        </div>
      </FramePanel>
      {reviewMetrics && reviewMetrics.length > 0 ? (
        <ul className="border-t border-border px-3.5 py-2" data-testid="split-run-intent-review-metrics">
          {reviewMetrics.map((metric) => (
            <li key={metric.key ?? metric.id} className="flex items-start justify-between gap-3 py-1 text-[12px]">
              <span className="min-w-0">
                <span className="font-medium text-foreground">{metric.name}</span>
                {metric.summary ? <span className="mt-0.5 block text-muted-foreground">{metric.summary}</span> : null}
              </span>
              <span className="shrink-0 tabular-nums text-muted-foreground">{metric.score}/5</span>
            </li>
          ))}
        </ul>
      ) : null}
    </Frame>
  );
}

function PlanningHeading({ title, hasPlan, analyzing }: { title?: string; hasPlan: boolean; analyzing: boolean }) {
  return (
    <div
      className="flex min-w-0 flex-1 basis-48 items-center gap-2.5"
      data-testid="split-run-intent-verdict"
      data-tone={analyzing ? "analyzing" : undefined}
    >
      {analyzing ? (
        <ConfidenceAnalyzingIndicator
          testId="split-run-intent-verdict-analyzing"
          showTooltip={false}
          decorative
          className="shrink-0"
        />
      ) : hasPlan ? (
        <FileText aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      ) : null}
      <div className="min-w-0">
        <p className="break-words text-[13px] leading-5 font-medium text-foreground">
          {hasPlan ? title || "Plan" : "Planning assessment"}
        </p>
        {hasPlan || analyzing ? (
          <p className="text-[12px] leading-4 text-muted-foreground">
            {analyzing ? (hasPlan ? "Updating plan…" : "Analyzing task…") : "Draft plan"}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function PlanningCreditNotice({ verdict }: { verdict: ComposerCreditVerdict }) {
  return (
    <div className="flex min-w-0 flex-1 items-start gap-2" data-testid="split-run-intent-verdict" data-tone="blocked">
      <ReadinessDot tone="blocked" className="mt-1.5" />
      <div className="min-w-0">
        <p className="text-[13px] leading-5 font-medium text-foreground">{verdict.headline}</p>
        <p className="text-[12px] leading-4 text-muted-foreground">
          {verdict.text}
          {verdict.href ? (
            <>
              {" "}
              <Link
                href={verdict.href}
                className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
              >
                {verdict.actionLabel}
              </Link>
            </>
          ) : null}
        </p>
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
      variant="ghost"
      size="sm"
      aria-expanded={onToggle ? open : undefined}
      aria-pressed={onToggle ? open : undefined}
      aria-label={open ? "Hide plan" : "Open plan"}
      onClick={onToggle}
      className="text-muted-foreground hover:text-foreground"
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
      {open ? "Hide plan" : "Open plan"}
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
