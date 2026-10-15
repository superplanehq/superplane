import { useId, useState } from "react";

import { ChevronRight, FileText } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Frame, FramePanel } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { CheckStatusIcon } from "../../workOrders/CheckStatusIcon";
import { ReadinessDot } from "../../workOrders/ReadinessMark";
import { workOrderCheckStatus, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ScoreEvidenceRow, type ScoreEvidenceDetails, type ScoreEvidenceValue } from "../../workOrders/ScoreEvidence";
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
  questionsPending = false,
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
  questionsPending?: boolean;
  showClarity?: boolean;
  showConfidence?: boolean;
  isAnalyzing?: boolean;
  canTogglePlan?: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  creditVerdict?: ComposerCreditVerdict;
}) {
  const analyzing = isAnalyzing && !creditVerdict;
  const metrics = reviewMetrics?.length ? reviewMetrics : undefined;
  const checks = useConfidenceChecks({ confidence, metrics, showConfidence, analyzing, questionsPending });
  return (
    <Frame
      dense
      className="relative my-3 w-full min-w-0"
      role="region"
      aria-label={canTogglePlan ? "Plan" : "Planning assessment"}
      data-testid="split-run-intent-status-card"
    >
      <FramePanel fit className="overflow-hidden p-0" data-testid="split-run-intent-plan-updated">
        <div className="px-3.5 py-2.5" data-testid="split-run-intent-composer-chips">
          <PlanningReviewHeading
            open={open}
            title={title}
            analyzing={analyzing}
            canTogglePlan={canTogglePlan}
            questionsPending={questionsPending}
            planStatus={planStatus}
            onToggle={onToggle}
            creditVerdict={creditVerdict}
          />
          <ScoreEvidenceRow
            clarity={clarity}
            confidence={confidence}
            showClarity={showClarity}
            showConfidence={showConfidence}
            isAnalyzing={analyzing}
            confidenceDetails={checks}
            testIds={SCORE_TEST_IDS}
            className={cn(canTogglePlan && "pl-6")}
          />
        </div>
        {checks?.open && metrics ? <ReviewMetricsSection id={checks.controlsId} metrics={metrics} /> : null}
      </FramePanel>
    </Frame>
  );
}

function PlanningReviewHeading({
  open,
  title,
  analyzing,
  canTogglePlan,
  questionsPending,
  planStatus,
  onToggle,
  creditVerdict,
}: {
  open: boolean;
  title?: string;
  analyzing: boolean;
  canTogglePlan: boolean;
  questionsPending: boolean;
  planStatus?: PlanChipStatus;
  onToggle?: () => void;
  creditVerdict?: ComposerCreditVerdict;
}) {
  return (
    <>
      {creditVerdict ? <PlanningCreditNotice verdict={creditVerdict} /> : null}
      {canTogglePlan ? (
        <div
          data-testid={creditVerdict ? undefined : "split-run-intent-verdict"}
          data-tone={analyzing ? "analyzing" : undefined}
        >
          <PlanToggle open={open} title={title} isAnalyzing={analyzing} planStatus={planStatus} onToggle={onToggle} />
        </div>
      ) : creditVerdict ? null : (
        <PlanningAssessmentHeading analyzing={analyzing} questionsPending={questionsPending} />
      )}
    </>
  );
}

function useConfidenceChecks({
  confidence,
  metrics,
  showConfidence,
  analyzing,
  questionsPending,
}: {
  confidence?: ComposerScore;
  metrics?: WorkOrderCheckPresentation[];
  showConfidence: boolean;
  analyzing: boolean;
  questionsPending: boolean;
}): ScoreEvidenceDetails | undefined {
  const controlsId = useId();
  const maxScore = confidence?.maxScore ?? 3;
  const scoreKey = `${confidence?.score}/${maxScore}:${metrics?.map((metric) => `${metric.key ?? metric.id}:${metric.score}`).join("|")}`;
  const [userChoice, setUserChoice] = useState<{ scoreKey: string; open: boolean }>();
  const open =
    userChoice?.scoreKey === scoreKey
      ? userChoice.open
      : !questionsPending && (confidence?.score ?? maxScore) < maxScore;
  if (!metrics || !showConfidence || analyzing || confidence?.score == null) {
    return undefined;
  }
  return { open, controlsId, checks: metrics, onToggle: () => setUserChoice({ scoreKey, open: !open }) };
}

/** The checks use the full width of the plan card when review is needed. */
function ReviewMetricsSection({ id, metrics }: { id: string; metrics: WorkOrderCheckPresentation[] }) {
  const attentionSummary = reviewAttentionSummary(metrics);
  return (
    <section
      id={id}
      aria-label="Confidence checks"
      data-testid="split-run-intent-review-metrics"
      className="border-t border-border/70 bg-muted/20 px-4"
    >
      {attentionSummary ? (
        <p className="border-b border-border/60 py-2.5 text-sm font-semibold text-foreground">{attentionSummary}</p>
      ) : null}
      <div className="divide-y divide-border/60">
        {metrics.map((metric) => (
          <ReviewMetricRow key={metric.key ?? metric.id} metric={metric} />
        ))}
      </div>
    </section>
  );
}

function reviewAttentionSummary(metrics: WorkOrderCheckPresentation[]): string | undefined {
  const names = metrics.filter((metric) => metric.score < metric.maxScore).map((metric) => metric.name);
  if (names.length === 0) {
    return undefined;
  }
  if (names.length === 1) {
    return `${names[0]} needs review.`;
  }
  if (names.length === 2) {
    return `${names[0]} and ${names[1]} need review.`;
  }
  return `${names.length} checks need review.`;
}

function ReviewMetricRow({ metric }: { metric: WorkOrderCheckPresentation }) {
  const status = workOrderCheckStatus(metric);
  return (
    <div className="flex min-w-0 gap-2.5 py-3">
      <CheckStatusIcon level={metric.level} label={status.label} className="mt-0.5" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-foreground">{metric.name}</p>
        {metric.summary ? <p className="mt-0.5 text-sm text-foreground/75">{metric.summary}</p> : null}
      </div>
    </div>
  );
}

function PlanningAssessmentHeading({ analyzing, questionsPending }: { analyzing: boolean; questionsPending: boolean }) {
  return (
    <div
      className="flex min-w-0 items-center gap-2.5"
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
      ) : null}
      <div className="min-w-0">
        <p className="break-words text-[13px] leading-5 font-medium text-foreground">Planning assessment</p>
        {analyzing ? <p className="text-[12px] leading-4 text-muted-foreground">Analyzing task…</p> : null}
        {questionsPending && !analyzing ? (
          <p className="mt-0.5 text-[13px] leading-5 text-foreground/75">
            There is no plan yet. Answer the questions to help the agent create one.
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
  title,
  isAnalyzing,
  planStatus,
  onToggle,
}: {
  open: boolean;
  title?: string;
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
      className="group h-auto min-h-8 max-w-full justify-start gap-2 px-2 py-1 text-left hover:bg-muted/70"
    >
      <span className="flex min-w-0 items-center gap-2.5">
        {isAnalyzing ? (
          <ConfidenceAnalyzingIndicator
            testId="split-run-intent-verdict-analyzing"
            showTooltip={false}
            decorative
            className="shrink-0"
          />
        ) : (
          <FileText aria-hidden className="size-4 shrink-0 text-foreground/65" />
        )}
        <span className="min-w-0 break-words whitespace-normal text-[14px] leading-5 font-semibold text-foreground">
          {title || "Plan"}
        </span>
      </span>
      <span className="inline-flex shrink-0 items-center gap-1 text-xs font-medium text-foreground/70 group-hover:text-foreground">
        {open ? "Hide plan" : "Open plan"}
        <ChevronRight aria-hidden className={cn("size-3.5 transition-transform", open && "rotate-90")} />
        <PlanStatusMark isAnalyzing={isAnalyzing} planStatus={planStatus} />
      </span>
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
