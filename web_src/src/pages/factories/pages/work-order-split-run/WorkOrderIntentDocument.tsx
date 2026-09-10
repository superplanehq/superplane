import { useState, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import { CONFIDENCE_CHECK_NAME, CONFIDENCE_SCORE_MAX } from "../../lib/confidenceScore";
import { INTENT_DOCUMENT_TITLE, type IntentDocument } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ConfidenceAnalyzingIndicator, ConfidenceMeter } from "../../workOrders/ConfidenceMeter";
import { splitRunIntentDocument } from "./splitRunPopupModel";
import { DEFAULT_INTENT_LEFT_PERCENT, useSplitRunPanePercent } from "./useSplitRunPanePercent";

const SESSION_TITLE_FALLBACK = "Task";

/**
 * Description-tab reading pane: original request as a chat, generated
 * summary or plan, and sticky confidence plus decision notes on the plan pane.
 */
export function WorkOrderIntentDocument({
  title,
  description,
  artifacts,
  confidence,
  isAnalyzing = false,
  files,
  resultAfterBody,
  resultFooter,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing?: boolean;
  files?: FilesFile[];
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
}) {
  const [showPlan, setShowPlan] = useState(false);
  const split = useSplitRunPanePercent({ defaultPercent: DEFAULT_INTENT_LEFT_PERCENT, minPercent: 28, maxPercent: 68 });
  const document = splitRunIntentDocument({ artifacts, description });
  const hasPlan = Boolean(document.plan.trim());
  const sessionTitle = title.trim() || SESSION_TITLE_FALLBACK;

  return (
    <article className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="split-run-intent-document">
      <div ref={split.containerRef} className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <div
          className="flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0"
          style={{ ["--intent-left" as string]: `${split.percent}%` }}
          data-testid="split-run-intent-request"
        >
          <header className="sticky top-0 z-10 shrink-0 px-5 py-3" data-testid="split-run-intent-session">
            <h2 className="truncate text-[13px] leading-5 font-medium text-foreground">{sessionTitle}</h2>
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            <div className="max-w-[92%]">
              <div
                className="rounded-2xl bg-muted/70 px-3.5 py-3"
                data-testid="split-run-description"
                aria-label="Request"
              >
                {description.trim() ? (
                  <WorkOrderDescription
                    description={description}
                    files={files}
                    previewHeight={FALLBACK_COLLAPSED_MAX_HEIGHT_PX}
                    fadeClassName="from-muted via-muted/80"
                  />
                ) : (
                  <p className="text-[13px] text-muted-foreground">No request yet.</p>
                )}
              </div>
            </div>
          </div>
        </div>

        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize the request and plan"
          data-testid="split-run-intent-resize-handle"
          onPointerDown={split.startResize}
          className="group relative z-10 hidden w-2 shrink-0 cursor-col-resize bg-transparent lg:block"
        >
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-border",
              split.isResizing && "bg-border",
            )}
          />
        </div>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col lg:min-w-[16rem]" data-testid="split-run-intent-result">
          <header className="flex shrink-0 items-start justify-between gap-3 px-5 pt-5 pb-2">
            <h2 className="min-w-0 text-[17px] leading-6 font-semibold tracking-tight text-foreground">
              {document.title || INTENT_DOCUMENT_TITLE}
            </h2>
            <label className="flex shrink-0 items-center gap-2 pt-0.5">
              <span className="text-[12px] text-muted-foreground">{showPlan ? "Plan" : "Summary"}</span>
              <Switch
                checked={showPlan}
                disabled={!hasPlan}
                onCheckedChange={setShowPlan}
                aria-label="Show the full plan"
              />
            </label>
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5" data-testid="split-run-intent-body">
            <IntentDocumentBody document={document} showPlan={showPlan} />
            {resultAfterBody}
          </div>
          <IntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
          {resultFooter}
        </div>
      </div>
    </article>
  );
}

function IntentDocumentBody({ document, showPlan }: { document: IntentDocument; showPlan: boolean }) {
  if (showPlan && document.plan.trim()) {
    return <MarkdownContent content={document.plan} variant="workspace" data-testid="split-run-intent-plan" />;
  }
  if (document.summary.trim()) {
    return <MarkdownContent content={document.summary} variant="workspace" data-testid="split-run-intent-summary" />;
  }
  return <p className="text-[13px] text-muted-foreground">The analysis has not written a plan yet.</p>;
}

function IntentConfidenceFooter({
  confidence,
  isAnalyzing,
}: {
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
}) {
  if (isAnalyzing && !confidence) {
    return (
      <footer
        className="shrink-0 border-t border-border px-5 py-3"
        data-testid="split-run-overview-checks"
        aria-label={CONFIDENCE_CHECK_NAME}
      >
        <div className="flex items-start gap-2">
          <ConfidenceAnalyzingIndicator testId="split-run-intent-confidence-meter" />
          <p className="text-[13px] leading-5 text-muted-foreground">The analysis is still running.</p>
        </div>
      </footer>
    );
  }

  if (!confidence) {
    return (
      <footer
        className="shrink-0 border-t border-border px-5 py-3"
        data-testid="split-run-overview-checks"
        aria-label={CONFIDENCE_CHECK_NAME}
      >
        <p className="text-[13px] text-muted-foreground">No confidence score yet.</p>
      </footer>
    );
  }

  return (
    <footer
      className="shrink-0 border-t border-border px-5 py-3"
      data-testid="split-run-overview-checks"
      aria-label={CONFIDENCE_CHECK_NAME}
    >
      <div className="flex items-start gap-2.5">
        <div className="flex shrink-0 flex-col items-start gap-1 pt-0.5">
          <ConfidenceMeter score={confidence.score} testId="split-run-intent-confidence-meter" />
          <span className="text-[12px] tabular-nums text-muted-foreground">
            {confidence.score}/{confidence.maxScore || CONFIDENCE_SCORE_MAX}
          </span>
        </div>
        <div className="min-w-0">
          <p className="text-[12px] text-muted-foreground">{CONFIDENCE_CHECK_NAME}</p>
          <p className="text-[13px] leading-5 text-foreground" data-testid="split-run-intent-confidence-copy">
            {confidence.summary?.trim() || "The analysis scored how clear this work is."}
          </p>
        </div>
      </div>
    </footer>
  );
}
