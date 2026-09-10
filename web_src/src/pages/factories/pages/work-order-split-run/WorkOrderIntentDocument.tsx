import { useState } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { Switch } from "@/components/ui/switch";
import { MarkdownContent } from "@/pages/app/Markdown";

import { WorkOrderDescription } from "../../WorkOrderDescription";
import { CONFIDENCE_CHECK_NAME, CONFIDENCE_SCORE_MAX } from "../../lib/confidenceScore";
import { INTENT_DOCUMENT_TITLE, type IntentDocument } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ConfidenceAnalyzingIndicator, ConfidenceMeter } from "../../workOrders/ConfidenceMeter";
import { splitRunIntentDocument } from "./splitRunPopupModel";

const SESSION_TITLE_FALLBACK = "Task";

/**
 * Description-tab reading pane: original request as a chat, generated
 * summary or plan, and a sticky confidence footer.
 */
export function WorkOrderIntentDocument({
  title,
  description,
  artifacts,
  confidence,
  isAnalyzing = false,
  files,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing?: boolean;
  files?: FilesFile[];
}) {
  const [showPlan, setShowPlan] = useState(false);
  const document = splitRunIntentDocument({ artifacts, description });
  const hasPlan = Boolean(document.plan.trim());
  const sessionTitle = title.trim() || SESSION_TITLE_FALLBACK;

  return (
    <article
      className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-border bg-card"
      data-testid="split-run-intent-document"
    >
      <div className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[minmax(13rem,0.42fr)_minmax(0,1fr)]">
        <div className="flex min-h-0 flex-col border-b border-border lg:border-r lg:border-b-0">
          <header
            className="sticky top-0 z-10 shrink-0 border-b border-border bg-card px-4 py-3"
            data-testid="split-run-intent-session"
          >
            <h2 className="truncate text-[13px] leading-5 font-medium text-foreground">{sessionTitle}</h2>
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
            <div className="max-w-[92%]">
              <div
                className="rounded-2xl bg-muted/70 px-3.5 py-3"
                data-testid="split-run-description"
                aria-label="Request"
              >
                {description.trim() ? (
                  <WorkOrderDescription description={description} files={files} collapsible={false} />
                ) : (
                  <p className="text-[13px] text-muted-foreground">No request yet.</p>
                )}
              </div>
            </div>
          </div>
          <IntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
        </div>

        <div className="flex min-h-0 flex-col">
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
          </div>
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
        className="shrink-0 border-t border-border px-4 py-3"
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
        className="shrink-0 border-t border-border px-4 py-3"
        data-testid="split-run-overview-checks"
        aria-label={CONFIDENCE_CHECK_NAME}
      >
        <p className="text-[13px] text-muted-foreground">No confidence score yet.</p>
      </footer>
    );
  }

  return (
    <footer
      className="shrink-0 border-t border-border px-4 py-3"
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
