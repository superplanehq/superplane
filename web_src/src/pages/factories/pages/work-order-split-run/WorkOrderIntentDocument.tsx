import { cloneElement, isValidElement, type PointerEvent, type ReactElement, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { useRevealAfterPending } from "@/hooks/useRevealAfterPending";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/ui/skeleton";

import { liveDraftReadiness, type DraftReadinessTone } from "../../lib/draftReadiness";
import { INTENT_DOCUMENT_TITLE, type IntentDocument } from "../../lib/intentDocument";
import { LOADING_REVEAL_CLASSNAME } from "../../lib/loadingReveal";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { hasPendingPlanningQuestions } from "./planningReviewState";
import { useRefineDocumentModel } from "./useRefineDocumentModel";
import { WorkOrderIntentConfidenceFooter } from "./WorkOrderIntentConfidenceFooter";
import { IntentDocumentSkeleton } from "./IntentDocumentSkeleton";
import { WorkOrderIntentPlan } from "./WorkOrderIntentPlan";
import type { IntentAnalysisChat } from "./intentAnalysisChat";
import { WorkOrderIntentRequest } from "./WorkOrderIntentRequest";
import type { SplitRunSource } from "./splitRunSource";

const REQUEST_PANE_BASE_CLASS = "flex min-h-0 min-w-0 w-full flex-1 flex-col";
const REQUEST_PANE_SPLIT_CLASS =
  "flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0";
const REFINE_SPLIT_EASE = "lg:duration-300 lg:ease-[cubic-bezier(0.32,0.72,0,1)] motion-reduce:lg:transition-none";

export type { IntentAnalysisChat } from "./intentAnalysisChat";

/**
 * Description-tab reading pane. Refine chat starts as one column. A
 * plan card opens the spec on the right. After Start, the left pane shows
 * source context. The summary stays on the right.
 */
type WorkOrderIntentDocumentProps = {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  reviewMetrics?: WorkOrderCheckPresentation[];
  isAnalyzing?: boolean;
  files?: FilesFile[];
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  analysis?: IntentAnalysisChat;
  contextSidebar?: ReactNode;
  streamKey?: string;
  streamReady?: boolean;
  source?: SplitRunSource;
};

export function WorkOrderIntentDocument({
  title,
  description,
  artifacts,
  clarity,
  confidence,
  reviewMetrics,
  isAnalyzing = false,
  files,
  resultAfterBody,
  resultFooter,
  analysis,
  contextSidebar,
  streamKey,
  streamReady = true,
  source,
}: WorkOrderIntentDocumentProps) {
  const {
    refineOpen,
    split,
    document,
    sessionTitle,
    showPlanPane,
    chatSolo,
    mountPlanPane,
    planWidth,
    chatWidth,
    showClosedDecision,
    analysisChat,
  } = useRefineDocumentModel({
    title,
    description,
    artifacts,
    clarity,
    confidence,
    reviewMetrics,
    isAnalyzing,
    resultFooter,
    analysis,
    contextSidebar,
    skipDescriptionFallback: !streamReady,
  });
  return (
    <article
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="split-run-intent-document"
      data-refine-chat-solo={chatSolo ? "" : undefined}
      data-refine-plan-open={refineOpen && showPlanPane ? "" : undefined}
    >
      <div ref={split.containerRef} className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <IntentRequestPane
          refineOpen={refineOpen}
          showPlanPane={showPlanPane}
          chatWidth={chatWidth}
          percent={split.percent}
          isResizing={split.isResizing}
          title={sessionTitle}
          description={description}
          files={files}
          source={source}
          contextSidebar={contextSidebar}
          analysis={withClosedDecision(analysisChat, showClosedDecision, resultFooter)}
        />
        {mountPlanPane ? (
          <IntentSpecColumn
            title={document.title || (streamReady ? INTENT_DOCUMENT_TITLE : "")}
            document={document}
            streamKey={streamKey}
            streamReady={streamReady}
            isAnalyzing={isAnalyzing}
            resultAfterBody={resultAfterBody}
            resultFooter={resultFooter}
            clarity={clarity}
            confidence={confidence}
            showClarity={analysisChat?.showClarity !== false}
            showConfidence={analysisChat?.showConfidence !== false}
            refineOpen={refineOpen}
            collapsed={chatSolo}
            planWidth={planWidth}
            animateSplit={refineOpen && !split.isResizing}
            contextSidebar={contextSidebar}
            isResizing={split.isResizing}
            onResize={split.startResize}
            onClosePlan={analysisChat?.onTogglePlan}
          />
        ) : null}
      </div>
    </article>
  );
}

/** Start stays available, but questions and plan updates take priority. */
function withClosedDecision(
  analysisChat: IntentAnalysisChat | undefined,
  showClosedDecision: boolean,
  resultFooter: ReactNode,
): IntentAnalysisChat | undefined {
  if (!analysisChat) {
    return undefined;
  }
  if (!showClosedDecision) {
    return { ...analysisChat, closedDecision: undefined, modelSelect: undefined };
  }
  const startTone = planningStartTone(analysisChat);
  return {
    ...analysisChat,
    startDiscouraged: !analysisChat.creditVerdict && (startTone === "caution" || startTone === "blocked"),
    prioritizeImplementation:
      Boolean(analysisChat.canTogglePlan) &&
      startTone === "ready" &&
      analysisChat.view.machineStatus === "waiting" &&
      !hasPendingPlanningQuestions(analysisChat.view),
    closedDecision: <ClosedPlanActions resultFooter={resultFooter} startTone={startTone} />,
  };
}

function planningStartTone(analysis: IntentAnalysisChat): DraftReadinessTone {
  if (analysis.creditVerdict) {
    return "blocked";
  }
  return liveDraftReadiness({
    clarity: analysis.showClarity !== false ? analysis.clarity?.score : undefined,
    confidence: analysis.showConfidence !== false ? analysis.confidence?.score : undefined,
    scoreMax: analysis.confidence?.maxScore,
    isAnalyzing:
      analysis.isAnalyzing || analysis.view.machineStatus === "starting" || analysis.view.machineStatus === "running",
  }).tone;
}

function requestPaneClassName(refineOpen: boolean, showPlanPane: boolean, isResizing: boolean) {
  if (!refineOpen) {
    return showPlanPane ? REQUEST_PANE_SPLIT_CLASS : REQUEST_PANE_BASE_CLASS;
  }
  return cn(
    REQUEST_PANE_BASE_CLASS,
    "border-b border-border lg:w-[var(--intent-left)] lg:flex-none lg:border-r lg:border-b-0",
    showPlanPane && "lg:min-w-[14rem]",
    showPlanPane && "max-lg:hidden",
    !isResizing && `lg:transition-[width] ${REFINE_SPLIT_EASE}`,
  );
}

function IntentRequestPane({
  refineOpen,
  showPlanPane,
  chatWidth,
  percent,
  isResizing,
  title,
  description,
  files,
  source,
  contextSidebar,
  analysis,
}: {
  refineOpen: boolean;
  showPlanPane: boolean;
  chatWidth: string;
  percent: number;
  isResizing: boolean;
  title: string;
  description: string;
  files?: FilesFile[];
  source?: SplitRunSource;
  contextSidebar?: ReactNode;
  analysis?: IntentAnalysisChat;
}) {
  return (
    <div
      className={requestPaneClassName(refineOpen, showPlanPane, isResizing)}
      style={{
        ["--intent-left" as string]: refineOpen ? chatWidth : showPlanPane ? `${percent}%` : undefined,
      }}
      data-testid="split-run-intent-request"
    >
      {contextSidebar ? (
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{contextSidebar}</div>
      ) : (
        <WorkOrderIntentRequest
          title={title}
          description={description}
          files={files}
          analysis={analysis}
          source={source}
        />
      )}
    </div>
  );
}

function ClosedPlanActions({ resultFooter, startTone }: { resultFooter?: ReactNode; startTone: DraftReadinessTone }) {
  const actions =
    isValidElement(resultFooter) && typeof resultFooter.type !== "string"
      ? cloneElement(resultFooter as ReactElement<{ actionsOnly?: boolean; startTone?: DraftReadinessTone }>, {
          actionsOnly: true,
          startTone,
        })
      : resultFooter;
  return <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">{actions}</div>;
}

function intentSpecPending(streamReady: boolean | undefined, isAnalyzing: boolean) {
  return !streamReady && !isAnalyzing;
}

function intentDocumentHasBody(document: IntentDocument) {
  return Boolean(document.summary.trim() || document.plan.trim());
}

function IntentSpecColumn({
  title,
  document,
  streamKey,
  streamReady,
  isAnalyzing,
  resultAfterBody,
  resultFooter,
  clarity,
  confidence,
  showClarity,
  showConfidence,
  refineOpen,
  collapsed = false,
  planWidth,
  animateSplit = false,
  contextSidebar,
  isResizing,
  onResize,
  onClosePlan,
}: {
  title: string;
  document: IntentDocument;
  streamKey?: string;
  streamReady?: boolean;
  isAnalyzing: boolean;
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  showClarity?: boolean;
  showConfidence?: boolean;
  refineOpen: boolean;
  collapsed?: boolean;
  planWidth?: string;
  animateSplit?: boolean;
  contextSidebar?: ReactNode;
  isResizing: boolean;
  onResize: (event: PointerEvent<HTMLDivElement>) => void;
  onClosePlan?: () => void;
}) {
  const pending = intentSpecPending(streamReady, isAnalyzing);
  const reveal = useRevealAfterPending(pending);
  return (
    <>
      <IntentSpecResizeHandle collapsed={collapsed} isResizing={isResizing} onResize={onResize} />
      <IntentSpecResult
        title={title}
        document={document}
        streamKey={streamKey}
        streamReady={streamReady}
        isAnalyzing={isAnalyzing}
        showSkeleton={pending && !intentDocumentHasBody(document)}
        reveal={reveal}
        resultAfterBody={resultAfterBody}
        resultFooter={resultFooter}
        clarity={clarity}
        confidence={confidence}
        showClarity={showClarity}
        showConfidence={showConfidence}
        refineOpen={refineOpen}
        collapsed={collapsed}
        planWidth={planWidth}
        animateSplit={animateSplit}
        contextSidebar={contextSidebar}
        onClosePlan={onClosePlan}
      />
    </>
  );
}

function IntentSpecResizeHandle({
  collapsed,
  isResizing,
  onResize,
}: {
  collapsed: boolean;
  isResizing: boolean;
  onResize: (event: PointerEvent<HTMLDivElement>) => void;
}) {
  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize the request and plan"
      data-testid="split-run-intent-resize-handle"
      onPointerDown={onResize}
      className={cn(
        "group relative z-10 hidden w-2 shrink-0 cursor-col-resize bg-transparent lg:block",
        collapsed && "lg:hidden",
      )}
    >
      <div
        aria-hidden
        className={cn(
          "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-border",
          isResizing && "bg-border",
        )}
      />
    </div>
  );
}

function specResultClassName(collapsed: boolean, animateSplit: boolean) {
  return cn(
    "flex min-h-0 min-w-0 flex-col overflow-hidden",
    collapsed && "pointer-events-none max-lg:hidden",
    animateSplit && `lg:w-[var(--intent-plan)] lg:flex-none lg:transition-[width] ${REFINE_SPLIT_EASE}`,
    !collapsed && !animateSplit && "flex-1 lg:min-w-[16rem]",
  );
}

function IntentSpecResult({
  title,
  document,
  streamKey,
  streamReady,
  isAnalyzing,
  showSkeleton,
  reveal,
  resultAfterBody,
  resultFooter,
  clarity,
  confidence,
  showClarity,
  showConfidence,
  refineOpen,
  collapsed,
  planWidth,
  animateSplit,
  contextSidebar,
  onClosePlan,
}: {
  title: string;
  document: IntentDocument;
  streamKey?: string;
  streamReady?: boolean;
  isAnalyzing: boolean;
  showSkeleton: boolean;
  reveal: boolean;
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  showClarity?: boolean;
  showConfidence?: boolean;
  refineOpen: boolean;
  collapsed: boolean;
  planWidth?: string;
  animateSplit: boolean;
  contextSidebar?: ReactNode;
  onClosePlan?: () => void;
}) {
  return (
    <div
      className={specResultClassName(collapsed, animateSplit)}
      style={animateSplit ? { ["--intent-plan" as string]: planWidth } : undefined}
      data-testid="split-run-intent-result"
      data-reveal={reveal ? "" : undefined}
      data-state={collapsed ? "closed" : "open"}
      aria-hidden={collapsed || undefined}
      inert={collapsed || undefined}
    >
      {refineOpen && onClosePlan ? (
        <div className="shrink-0 px-5 pt-3 lg:hidden">
          <Button type="button" variant="ghost" size="sm" onClick={onClosePlan}>
            <ArrowLeft aria-hidden className="size-4" />
            Back to conversation
          </Button>
        </div>
      ) : null}
      <header className="flex shrink-0 items-start justify-between gap-3 px-5 pt-5 pb-2">
        <h2 className="min-h-6 min-w-0 text-[17px] leading-6 font-semibold tracking-tight text-foreground">
          {showSkeleton ? (
            <Skeleton className="h-6 w-2/3" aria-hidden />
          ) : (
            <span className={cn(reveal && LOADING_REVEAL_CLASSNAME)}>{title}</span>
          )}
        </h2>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5" data-testid="split-run-intent-body">
        {showSkeleton ? (
          <IntentDocumentSkeleton />
        ) : (
          <div className={cn(reveal && LOADING_REVEAL_CLASSNAME)}>
            <WorkOrderIntentPlan
              document={document}
              streamKey={streamKey}
              streamReady={streamReady}
              isAnalyzing={isAnalyzing}
            />
          </div>
        )}
        {resultAfterBody}
      </div>
      {refineOpen ? null : (
        <IntentSpecFooter
          clarity={clarity}
          confidence={confidence}
          showClarity={showClarity}
          showConfidence={showConfidence}
          isAnalyzing={isAnalyzing}
          resultFooter={resultFooter}
          contextSidebar={Boolean(contextSidebar)}
        />
      )}
    </div>
  );
}

function IntentSpecFooter({
  clarity,
  confidence,
  showClarity = true,
  showConfidence = true,
  isAnalyzing,
  resultFooter,
  contextSidebar,
}: {
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  showClarity?: boolean;
  showConfidence?: boolean;
  isAnalyzing: boolean;
  resultFooter?: ReactNode;
  contextSidebar: boolean;
}) {
  if (contextSidebar) {
    return resultFooter;
  }
  return (
    <>
      <WorkOrderIntentConfidenceFooter
        clarity={clarity}
        confidence={confidence}
        showClarity={showClarity}
        showConfidence={showConfidence}
        isAnalyzing={isAnalyzing}
      />
      {resultFooter}
    </>
  );
}
