import { cloneElement, isValidElement, type PointerEvent, type ReactElement, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { cn } from "@/lib/utils";

import { INTENT_DOCUMENT_TITLE, type IntentDocument } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { useRefineDocumentModel } from "./useRefineDocumentModel";
import { WorkOrderIntentConfidenceFooter } from "./WorkOrderIntentConfidenceFooter";
import { WorkOrderIntentPlan } from "./WorkOrderIntentPlan";
import { WorkOrderIntentRequest, type IntentAnalysisChat } from "./WorkOrderIntentRequest";
import type { SplitRunSource } from "./splitRunSource";

const REQUEST_PANE_BASE_CLASS = "flex min-h-0 min-w-0 w-full flex-1 flex-col";
const REQUEST_PANE_SPLIT_CLASS =
  "flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0";
const REFINE_SPLIT_EASE = "lg:duration-300 lg:ease-[cubic-bezier(0.32,0.72,0,1)] motion-reduce:lg:transition-none";

export type { IntentAnalysisChat } from "./WorkOrderIntentRequest";

/**
 * Description-tab reading pane. Refine chat starts as one column. A sticky
 * plan control opens the spec on the right. After Start, the left pane shows
 * source context. The summary stays on the right.
 */
type WorkOrderIntentDocumentProps = {
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
};

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
    confidence,
    isAnalyzing,
    resultFooter,
    analysis,
    contextSidebar,
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
            title={document.title || INTENT_DOCUMENT_TITLE}
            document={document}
            streamKey={streamKey}
            streamReady={streamReady}
            isAnalyzing={isAnalyzing}
            resultAfterBody={resultAfterBody}
            resultFooter={resultFooter}
            confidence={confidence}
            refineOpen={refineOpen}
            collapsed={chatSolo}
            planWidth={planWidth}
            animateSplit={refineOpen && !split.isResizing}
            contextSidebar={contextSidebar}
            isResizing={split.isResizing}
            onResize={split.startResize}
          />
        ) : null}
      </div>
    </article>
  );
}

function withClosedDecision(
  analysisChat: IntentAnalysisChat | undefined,
  showClosedDecision: boolean,
  resultFooter?: ReactNode,
): IntentAnalysisChat | undefined {
  if (!analysisChat) {
    return undefined;
  }
  return {
    ...analysisChat,
    closedDecision: showClosedDecision ? <ClosedPlanActions resultFooter={resultFooter} /> : undefined,
  };
}

function requestPaneClassName(refineOpen: boolean, showPlanPane: boolean, isResizing: boolean) {
  if (!refineOpen) {
    return showPlanPane ? REQUEST_PANE_SPLIT_CLASS : REQUEST_PANE_BASE_CLASS;
  }
  return cn(
    REQUEST_PANE_BASE_CLASS,
    "border-b border-border lg:w-[var(--intent-left)] lg:flex-none lg:border-r lg:border-b-0",
    showPlanPane && "lg:min-w-[14rem]",
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

function ClosedPlanActions({ resultFooter }: { resultFooter?: ReactNode }) {
  const actions =
    isValidElement(resultFooter) && typeof resultFooter.type !== "string"
      ? cloneElement(resultFooter as ReactElement<{ actionsOnly?: boolean }>, { actionsOnly: true })
      : resultFooter;
  return <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">{actions}</div>;
}

function IntentSpecColumn({
  title,
  document,
  streamKey,
  streamReady,
  isAnalyzing,
  resultAfterBody,
  resultFooter,
  confidence,
  refineOpen,
  collapsed = false,
  planWidth,
  animateSplit = false,
  contextSidebar,
  isResizing,
  onResize,
}: {
  title: string;
  document: IntentDocument;
  streamKey?: string;
  streamReady?: boolean;
  isAnalyzing: boolean;
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  confidence?: WorkOrderCheckPresentation;
  refineOpen: boolean;
  collapsed?: boolean;
  planWidth?: string;
  animateSplit?: boolean;
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
      <div
        className={cn(
          "flex min-h-0 min-w-0 flex-col overflow-hidden",
          collapsed && "pointer-events-none max-lg:hidden",
          animateSplit && `lg:w-[var(--intent-plan)] lg:flex-none lg:transition-[width] ${REFINE_SPLIT_EASE}`,
          !collapsed && !animateSplit && "flex-1 lg:min-w-[16rem]",
        )}
        style={animateSplit ? { ["--intent-plan" as string]: planWidth } : undefined}
        data-testid="split-run-intent-result"
        data-state={collapsed ? "closed" : "open"}
        aria-hidden={collapsed || undefined}
        inert={collapsed || undefined}
      >
        <header className="flex shrink-0 items-start justify-between gap-3 px-5 pt-5 pb-2">
          <h2 className="min-w-0 text-[17px] leading-6 font-semibold tracking-tight text-foreground">{title}</h2>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5" data-testid="split-run-intent-body">
          <WorkOrderIntentPlan
            document={document}
            streamKey={streamKey}
            streamReady={streamReady}
            isAnalyzing={isAnalyzing}
          />
          {resultAfterBody}
        </div>
        {refineOpen ? null : (
          <IntentSpecFooter
            confidence={confidence}
            isAnalyzing={isAnalyzing}
            resultFooter={resultFooter}
            contextSidebar={Boolean(contextSidebar)}
          />
        )}
      </div>
    </>
  );
}

function IntentSpecFooter({
  confidence,
  isAnalyzing,
  resultFooter,
  contextSidebar,
}: {
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
  resultFooter?: ReactNode;
  contextSidebar: boolean;
}) {
  if (contextSidebar) {
    return resultFooter;
  }
  return (
    <>
      <WorkOrderIntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
      {resultFooter}
    </>
  );
}
