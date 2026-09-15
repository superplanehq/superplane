import { useState, type PointerEvent, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { cn } from "@/lib/utils";

import { INTENT_DOCUMENT_TITLE } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { useFactoryPreviewFlag } from "../factoryPreviewFlagsContext";
import { latestPlanScore } from "./latestPlanScore";
import { SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME, splitRunIntentDocument } from "./splitRunPopupModel";
import { WorkOrderIntentConfidenceFooter } from "./WorkOrderIntentConfidenceFooter";
import { WorkOrderIntentPlan } from "./WorkOrderIntentPlan";
import { WorkOrderIntentRequest, type IntentAnalysisChat } from "./WorkOrderIntentRequest";
import { DEFAULT_INTENT_LEFT_PERCENT, useSplitRunPanePercent } from "./useSplitRunPanePercent";
import type { SplitRunSource } from "./splitRunSource";

const SESSION_TITLE_FALLBACK = "Task";
const REQUEST_PANE_SPLIT_CLASS =
  "flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0";
const REQUEST_PANE_SOLO_CLASS = "flex min-h-0 min-w-0 w-full flex-1 flex-col";

export type { IntentAnalysisChat } from "./WorkOrderIntentRequest";

/**
 * Description-tab reading pane. Refine chat starts as one column. A sticky
 * plan control opens the spec on the right. After Start, the left pane shows
 * source context. The summary stays on the right.
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
  streamKey,
  streamReady = true,
  source,
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
  streamKey?: string;
  streamReady?: boolean;
  source?: SplitRunSource;
}) {
  const refineOpen = Boolean(analysis) && !contextSidebar;
  const oneBarPlanStrip = useFactoryPreviewFlag("oneBarPlanStrip");
  const [showPlan, setShowPlan] = useState(false);
  const [planPaneOpen, setPlanPaneOpen] = useState(false);
  const split = useSplitRunPanePercent({ defaultPercent: DEFAULT_INTENT_LEFT_PERCENT, minPercent: 28, maxPercent: 68 });
  const document = splitRunIntentDocument({ artifacts, description });
  const sessionTitle = title.trim() || SESSION_TITLE_FALLBACK;
  const showPlanPane = !refineOpen || planPaneOpen;
  const chatSolo = refineOpen && !planPaneOpen;
  const analysisChat = analysis
    ? {
        ...analysis,
        planPaneOpen,
        onTogglePlan: () => setPlanPaneOpen((current) => !current),
        latestPlanScore: latestPlanScore(analysis.view.messages),
        closedDecision: chatSolo ? (
          <ClosedPlanActions
            confidence={confidence}
            isAnalyzing={isAnalyzing}
            resultFooter={resultFooter}
            oneBar={oneBarPlanStrip}
          />
        ) : undefined,
      }
    : undefined;

  return (
    <article
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="split-run-intent-document"
      data-refine-chat-solo={chatSolo ? "" : undefined}
    >
      <div ref={split.containerRef} className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <div
          className={showPlanPane ? REQUEST_PANE_SPLIT_CLASS : REQUEST_PANE_SOLO_CLASS}
          style={showPlanPane ? { ["--intent-left" as string]: `${split.percent}%` } : undefined}
          data-testid="split-run-intent-request"
        >
          {contextSidebar ? (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{contextSidebar}</div>
          ) : (
            <WorkOrderIntentRequest
              title={sessionTitle}
              description={description}
              files={files}
              analysis={analysisChat}
              source={source}
            />
          )}
        </div>
        {showPlanPane ? (
          <IntentSpecColumn
            title={document.title || INTENT_DOCUMENT_TITLE}
            document={document}
            streamKey={streamKey}
            streamReady={streamReady}
            showPlan={showPlan}
            isAnalyzing={isAnalyzing}
            onTogglePlan={() => setShowPlan((current) => !current)}
            resultAfterBody={resultAfterBody}
            resultFooter={resultFooter}
            confidence={confidence}
            refineOpen={refineOpen}
            contextSidebar={contextSidebar}
            isResizing={split.isResizing}
            onResize={split.startResize}
          />
        ) : null}
      </div>
    </article>
  );
}

function ClosedPlanActions({
  confidence,
  isAnalyzing,
  resultFooter,
  oneBar,
}: {
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
  resultFooter?: ReactNode;
  oneBar: boolean;
}) {
  return (
    <div
      className={cn(
        "flex min-w-0 flex-wrap items-center justify-end",
        oneBar
          ? "gap-2 [&_[data-testid=split-run-attention-note]]:flex-none [&_[data-testid=split-run-intent-decision-tip]]:hidden [&_[data-testid=split-run-review]]:flex-none"
          : "gap-3",
      )}
    >
      {oneBar ? null : <WorkOrderIntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} flush />}
      {resultFooter}
    </div>
  );
}

function IntentSpecColumn({
  title,
  document,
  streamKey,
  streamReady,
  showPlan,
  isAnalyzing,
  onTogglePlan,
  resultAfterBody,
  resultFooter,
  confidence,
  refineOpen,
  contextSidebar,
  isResizing,
  onResize,
}: {
  title: string;
  document: ReturnType<typeof splitRunIntentDocument>;
  streamKey?: string;
  streamReady?: boolean;
  showPlan: boolean;
  isAnalyzing: boolean;
  onTogglePlan: () => void;
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  confidence?: WorkOrderCheckPresentation;
  refineOpen: boolean;
  contextSidebar?: ReactNode;
  isResizing: boolean;
  onResize: (event: PointerEvent<HTMLDivElement>) => void;
}) {
  return (
    <>
      <div
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize the request and plan"
        data-testid="split-run-intent-resize-handle"
        onPointerDown={onResize}
        className="group relative z-10 hidden w-2 shrink-0 cursor-col-resize bg-transparent lg:block"
      >
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-border",
            isResizing && "bg-border",
          )}
        />
      </div>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col lg:min-w-[16rem]" data-testid="split-run-intent-result">
        <header className="flex shrink-0 items-start justify-between gap-3 px-5 pt-5 pb-2">
          <h2 className="min-w-0 text-[17px] leading-6 font-semibold tracking-tight text-foreground">{title}</h2>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5" data-testid="split-run-intent-body">
          <WorkOrderIntentPlan
            document={document}
            streamKey={streamKey}
            streamReady={streamReady}
            expanded={showPlan}
            isAnalyzing={isAnalyzing}
            onToggle={onTogglePlan}
          />
          {resultAfterBody}
        </div>
        <IntentSpecFooter
          confidence={confidence}
          isAnalyzing={isAnalyzing}
          resultFooter={resultFooter}
          refineOpen={refineOpen}
          contextSidebar={Boolean(contextSidebar)}
        />
      </div>
    </>
  );
}

function IntentSpecFooter({
  confidence,
  isAnalyzing,
  resultFooter,
  refineOpen,
  contextSidebar,
}: {
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
  resultFooter?: ReactNode;
  refineOpen: boolean;
  contextSidebar: boolean;
}) {
  if (contextSidebar) {
    return resultFooter;
  }
  if (refineOpen) {
    return (
      <div
        className={cn(SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME, "justify-between gap-4")}
        data-testid="split-run-intent-decision"
      >
        <WorkOrderIntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} flush />
        {resultFooter}
      </div>
    );
  }
  return (
    <>
      <WorkOrderIntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
      {resultFooter}
    </>
  );
}
