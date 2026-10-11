import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";

import { EyeOff, FileText } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Frame, FramePanel } from "@/components/reui/frame";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { ConfidenceAnalyzingIndicator } from "../../workOrders/ConfidenceMeter";
import { ReadinessDot } from "../../workOrders/ReadinessMark";
import { planningReviewAtMax } from "../../lib/planningReviewScore";
import { workOrderCheckStatus, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ScoreEvidenceRow, type ScoreEvidenceDrawer, type ScoreEvidenceValue } from "../../workOrders/ScoreEvidence";
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
  const drawerMetrics = drawerMetricsFor(reviewMetrics);
  const drawer = useConfidenceDrawer();
  return (
    <Frame
      dense
      className="relative my-3 w-full min-w-0"
      role="region"
      aria-label={canTogglePlan ? "Plan" : "Planning assessment"}
      data-testid="split-run-intent-status-card"
    >
      {drawerMetrics && drawer.control.open ? (
        <ReviewMetricsDrawer
          id={drawer.control.controlsId}
          metrics={drawerMetrics}
          onHoverChange={drawer.onHoverChange}
        />
      ) : null}
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
            confidenceDrawer={drawerMetrics ? drawer.control : undefined}
            testIds={SCORE_TEST_IDS}
          />
          {canTogglePlan ? (
            <PlanToggle open={open} isAnalyzing={analyzing} planStatus={planStatus} onToggle={onToggle} />
          ) : null}
        </div>
      </FramePanel>
    </Frame>
  );
}

/**
 * At 3/3 there is nothing to fix, so the chip shows the summary hover
 * card instead of the per-check drawer.
 */
function drawerMetricsFor(metrics?: WorkOrderCheckPresentation[]): WorkOrderCheckPresentation[] | undefined {
  if (!metrics || metrics.length === 0 || planningReviewAtMax(metrics)) {
    return undefined;
  }
  return metrics;
}

const DRAWER_CLOSE_DELAY_MS = 120;

/**
 * Hover peeks the drawer, click pins it, Escape closes it. The close
 * delay lets the pointer cross the gap between the chip and the drawer.
 */
function useConfidenceDrawer(): { control: ScoreEvidenceDrawer; onHoverChange: (hovering: boolean) => void } {
  const controlsId = useId();
  const [pinned, setPinned] = useState(false);
  const [hovering, setHovering] = useState(false);
  const closeTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const open = pinned || hovering;

  const onHoverChange = (next: boolean) => {
    clearTimeout(closeTimer.current);
    if (next) {
      setHovering(true);
      return;
    }
    closeTimer.current = setTimeout(() => setHovering(false), DRAWER_CLOSE_DELAY_MS);
  };

  const onToggle = () => {
    clearTimeout(closeTimer.current);
    setPinned((current) => {
      if (current) {
        setHovering(false);
      }
      return !current;
    });
  };

  useEffect(() => () => clearTimeout(closeTimer.current), []);

  useEffect(() => {
    if (!open) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }
      setPinned(false);
      setHovering(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open]);

  return { control: { open, pinned, controlsId, onHoverChange, onToggle }, onHoverChange };
}

/**
 * The three review checks in a drawer above the plan card: a list row per
 * check with its name, a verdict badge, and the agent's one-line reason
 * below. The drawer floats, so it never moves the card or the composer.
 */
function ReviewMetricsDrawer({
  id,
  metrics,
  onHoverChange,
}: {
  id: string;
  metrics: WorkOrderCheckPresentation[];
  onHoverChange: (hovering: boolean) => void;
}) {
  const { ref, side } = useDrawerSide();
  return (
    <div
      ref={ref}
      id={id}
      role="group"
      aria-label="Confidence checks"
      data-testid="split-run-intent-review-metrics"
      data-side={side}
      className={cn(
        "sp-confidence-drawer absolute inset-x-0 z-[60] grid grid-cols-3 divide-x divide-border overflow-hidden rounded-lg border border-border bg-popover shadow-md",
        side === "above" ? "bottom-full mb-2" : "top-full mt-2",
      )}
      onMouseEnter={() => onHoverChange(true)}
      onMouseLeave={() => onHoverChange(false)}
    >
      {metrics.map((metric) => (
        <ReviewMetricCell key={metric.key ?? metric.id} metric={metric} />
      ))}
    </div>
  );
}

/**
 * The drawer prefers the space above the plan card. When the card sits near
 * the top of its scroll area, the drawer would get clipped there, so it
 * opens below the card instead. Measured once per open, before first paint.
 */
function useDrawerSide() {
  const ref = useRef<HTMLDivElement>(null);
  const [side, setSide] = useState<"above" | "below">("above");
  useLayoutEffect(() => {
    const node = ref.current;
    if (!node) {
      return;
    }
    if (node.getBoundingClientRect().top < clippingAncestorTop(node)) {
      setSide("below");
    }
  }, []);
  return { ref, side };
}

function clippingAncestorTop(node: HTMLElement): number {
  for (let parent = node.parentElement; parent; parent = parent.parentElement) {
    if (window.getComputedStyle(parent).overflowY !== "visible") {
      return parent.getBoundingClientRect().top;
    }
  }
  return 0;
}

function ReviewMetricCell({ metric }: { metric: WorkOrderCheckPresentation }) {
  const status = workOrderCheckStatus(metric);
  return (
    <div className="sp-review-metric min-w-0 px-4 py-3">
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-[13px] leading-5 font-medium text-foreground">{metric.name}</span>
        <Badge variant="outline" className={cn("shrink-0 border", status.badgeClassName)}>
          {status.label}
        </Badge>
      </div>
      {metric.summary ? <p className="mt-1 text-[12px] leading-5 text-muted-foreground">{metric.summary}</p> : null}
    </div>
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
