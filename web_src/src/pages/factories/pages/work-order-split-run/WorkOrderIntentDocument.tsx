import { cloneElement, isValidElement, useState, type PointerEvent, type ReactElement, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { cn } from "@/lib/utils";

import { analysisPlanBody, hasAnalysisPlan } from "../../lib/analysisOutcome";
import { INTENT_DOCUMENT_TITLE } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { latestPlanScore } from "./latestPlanScore";
import { usePlanChipStatus } from "./planChipStatus";
import { useRefineLayoutPreference } from "./refineLayoutPreference";
import { splitRunIntentDocument } from "./splitRunPopupModel";
import { WorkOrderIntentConfidenceFooter } from "./WorkOrderIntentConfidenceFooter";
import { WorkOrderIntentPlan } from "./WorkOrderIntentPlan";
import { WorkOrderIntentRequest, type IntentAnalysisChat } from "./WorkOrderIntentRequest";
import {
  DEFAULT_INTENT_LEFT_PERCENT,
  DEFAULT_REFINE_INTENT_LEFT_PERCENT,
  useSplitRunPanePercent,
} from "./useSplitRunPanePercent";
import type { SplitRunSource } from "./splitRunSource";

const SESSION_TITLE_FALLBACK = "Task";
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
  const [showPlan, setShowPlan] = useState(false);
  const [planOpenedHere, setPlanOpenedHere] = useState(false);
  const layout = useRefineLayoutPreference();
  const hasPlan = hasAnalysisPlan(artifacts);
  const planPaneOpen = layout.planOpen && hasPlan;
  const split = useSplitRunPanePercent({
    defaultPercent: refineOpen ? DEFAULT_REFINE_INTENT_LEFT_PERCENT : DEFAULT_INTENT_LEFT_PERCENT,
    minPercent: 28,
    maxPercent: 68,
  });
  const document = splitRunIntentDocument({ artifacts, description });
  const planStatus = usePlanChipStatus(analysisPlanBody(artifacts), planOpenedHere && planPaneOpen);
  const clarityScore = analysis ? (latestPlanScore(analysis.view.messages) ?? confidence?.score) : confidence?.score;
  const sessionTitle = title.trim() || SESSION_TITLE_FALLBACK;
  const showPlanPane = !refineOpen || planPaneOpen;
  const chatSolo = refineOpen && !planPaneOpen;
  const mountPlanPane = !refineOpen ? showPlanPane : true;
  const planWidth = showPlanPane ? `${100 - split.percent}%` : "0%";
  const chatWidth = showPlanPane ? `${split.percent}%` : "100%";
  const analysisChat = analysis
    ? {
        ...analysis,
        planPaneOpen,
        onTogglePlan: () => {
          setPlanOpenedHere(true);
          layout.togglePlan();
        },
        canTogglePlan: hasPlan,
        clarityExpanded: layout.clarityExpanded,
        onToggleClarity: layout.toggleClarity,
        latestPlanScore: clarityScore,
        latestPlanSummary: confidence?.summary?.trim(),
        planStatus,
        isAnalyzing,
        closedDecision:
          refineOpen && resultFooter && clarityScore != null ? (
            <ClosedPlanActions resultFooter={resultFooter} />
          ) : undefined,
      }
    : undefined;

  return (
    <article
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="split-run-intent-document"
      data-refine-chat-solo={chatSolo ? "" : undefined}
      data-refine-plan-open={refineOpen && planPaneOpen ? "" : undefined}
    >
      <div ref={split.containerRef} className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <div
          className={cn(
            refineOpen
              ? cn(
                  REQUEST_PANE_BASE_CLASS,
                  "border-b border-border lg:w-[var(--intent-left)] lg:flex-none lg:border-r lg:border-b-0",
                  showPlanPane && "lg:min-w-[14rem]",
                  !split.isResizing && `lg:transition-[width] ${REFINE_SPLIT_EASE}`,
                )
              : showPlanPane
                ? REQUEST_PANE_SPLIT_CLASS
                : REQUEST_PANE_BASE_CLASS,
          )}
          style={{
            ["--intent-left" as string]: refineOpen ? chatWidth : showPlanPane ? `${split.percent}%` : undefined,
          }}
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
        {mountPlanPane ? (
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
  showPlan,
  isAnalyzing,
  onTogglePlan,
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
            expanded={showPlan}
            isAnalyzing={isAnalyzing}
            onToggle={onTogglePlan}
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
