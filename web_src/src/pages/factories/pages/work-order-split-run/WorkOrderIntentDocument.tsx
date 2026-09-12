import { useState, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { cn } from "@/lib/utils";

import { INTENT_DOCUMENT_TITLE } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { splitRunIntentDocument } from "./splitRunPopupModel";
import { WorkOrderIntentConfidenceFooter } from "./WorkOrderIntentConfidenceFooter";
import { WorkOrderIntentPlan } from "./WorkOrderIntentPlan";
import { WorkOrderIntentRequest, type IntentAnalysisChat } from "./WorkOrderIntentRequest";
import { DEFAULT_INTENT_LEFT_PERCENT, useSplitRunPanePercent } from "./useSplitRunPanePercent";

const SESSION_TITLE_FALLBACK = "Task";

export type { IntentAnalysisChat } from "./WorkOrderIntentRequest";

/**
 * Description-tab reading pane. Drafts keep analysis chat on the left and
 * the plan plus confidence on the right. After Start, the left pane shows
 * source context and confidence. The plan stays on the right.
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
  analysis,
  contextSidebar,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing?: boolean;
  files?: FilesFile[];
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  analysis?: IntentAnalysisChat;
  contextSidebar?: ReactNode;
}) {
  const [showPlan, setShowPlan] = useState(false);
  const split = useSplitRunPanePercent({ defaultPercent: DEFAULT_INTENT_LEFT_PERCENT, minPercent: 28, maxPercent: 68 });
  const document = splitRunIntentDocument({ artifacts, description });
  const sessionTitle = title.trim() || SESSION_TITLE_FALLBACK;

  return (
    <article className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="split-run-intent-document">
      <div ref={split.containerRef} className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <div
          className="flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0"
          style={{ ["--intent-left" as string]: `${split.percent}%` }}
          data-testid="split-run-intent-request"
        >
          {contextSidebar ? (
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-hidden">{contextSidebar}</div>
              <WorkOrderIntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
            </div>
          ) : (
            <WorkOrderIntentRequest title={sessionTitle} description={description} files={files} analysis={analysis} />
          )}
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
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5" data-testid="split-run-intent-body">
            <WorkOrderIntentPlan
              document={document}
              expanded={showPlan}
              onToggle={() => setShowPlan((current) => !current)}
            />
            {resultAfterBody}
          </div>
          {contextSidebar ? null : (
            <WorkOrderIntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
          )}
          {resultFooter}
        </div>
      </div>
    </article>
  );
}
