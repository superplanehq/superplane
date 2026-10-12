import type { FactoriesFactoryPullRequest, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryAutomations, useWorkOrder, useWorkOrderArtifacts } from "@/hooks/useFactoryData";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { THINKING_LEVEL_MEDIUM } from "@/lib/thinkingLevel";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ArrowLeft } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useLocation, useNavigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, firstFactoryLineId, workOrderBoardLineIdFromSearch } from "../lib/factoryPagePaths";
import { boardLineIdFromNavigationState, displayedBoardLineId } from "../lib/workOrderNumberResolution";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { formatWorkOrderIdentifier } from "../lib/workspaceKey";
import { OwnerTimeCostRow } from "../pages/work-order-popup-redesign/popupShared";
import { SPLIT_RUN_ANALYZING_NOTE } from "../pages/work-order-split-run/splitRunFooter";
import { ANALYSIS_PLANNING_COPY } from "../pages/work-order-split-run/useAnalysisPlanningSession";
import type { IntentAnalysisChat } from "../pages/work-order-split-run/intentAnalysisChat";
import { WorkOrderSplitRunOverview } from "../pages/work-order-split-run/WorkOrderSplitRunOverview";
import { DRAFT_START_MODEL_AUTO } from "../pages/work-order-split-run/draftStartModel";
import { factoryPlanningEnabled } from "../pages/planningSettingsModel";
import {
  columnAppsFromFactoryApps,
  splitRunFixtureForWorkOrder,
  type SplitRunFixture,
} from "../pages/work-order-split-run/splitRunMocks";
import { useSplitRunFooterActions } from "../pages/work-order-split-run/useSplitRunFooterActions";
import { useSplitRunFooterCloser } from "../pages/work-order-split-run/useSplitRunFooterCloser";
import { useSplitRunWorkOrderEdits } from "../pages/work-order-split-run/useSplitRunWorkOrderEdits";
import { draftStartAction, footerMutationHandlers } from "../pages/work-order-split-run/workOrderPopupActions";
import { useColumnAppCheckRuns } from "../pages/work-order-split-run/useColumnAppCheckRuns";
import { useWorkOrderPRFeedbackLog } from "../pages/useWorkOrderPRFeedbackRunHref";
import { MOBILE_TASK_COPY } from "./mobileCopy";
import { MobileDraftStartModelSelect } from "./MobileDraftStartModelSelect";
import { MobileTaskSections, PhoneTaskReview, Section } from "./MobileTaskSections";
import { useMobileRefineChat } from "./useMobileRefineChat";

function taskBackLineId(
  search: string,
  locationState: unknown,
  factory: { lines?: Array<{ id?: string }> } | null | undefined,
  order: FactoriesWorkOrder | undefined,
  taskPending: boolean,
): string | undefined {
  return displayedBoardLineId({
    queryLineId: workOrderBoardLineIdFromSearch(search),
    navigationLineId: boardLineIdFromNavigationState(locationState),
    lines: factory?.lines ?? [],
    order,
    fallbackLineId: taskPending ? undefined : firstFactoryLineId(factory),
  });
}

/**
 * Full-screen task view for the phone shell. Everything stacks in one
 * scrolling column so titles, notes, and the activity log stay readable on a
 * narrow screen. Actions come from the same footer model as the desktop popup.
 */
export function MobileTaskDetailPage() {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const { orderNumber = "" } = useParams<{ orderNumber?: string }>();
  const { search, state: locationState } = useLocation();
  const navigate = useNavigate();
  const { data: order, isLoading, isError } = useWorkOrder(organizationId, factoryId, orderNumber);
  const taskPending = isLoading && !order;
  const boardLineId = taskBackLineId(search, locationState, factory, order, taskPending);
  const backToBoard = () => {
    if (!boardLineId) {
      return;
    }
    navigate(factoryHomePath(organizationId, routeSegment, boardLineId));
  };

  usePageTitle([order?.title ?? "Task", factory?.name ?? "Workspace"]);

  if (isLoading && !order) {
    return (
      <MobileTaskFrame onBack={backToBoard} backDisabled={!boardLineId}>
        <p className="px-4 py-8 text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.loading}</p>
      </MobileTaskFrame>
    );
  }
  if (!order?.id || isError) {
    return (
      <MobileTaskFrame onBack={backToBoard} backDisabled={!boardLineId}>
        <div className="px-4 py-8" data-testid="mobile-task-not-found">
          <p className="text-[15px] font-semibold text-foreground">{MOBILE_TASK_COPY.notFound}</p>
          <p className="mt-1 text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.notFoundHelp}</p>
        </div>
      </MobileTaskFrame>
    );
  }
  return (
    <LoadedMobileTask
      key={order.id}
      order={order}
      orderId={order.id}
      lineId={boardLineId}
      lineName={factory?.lines?.find((line) => line.id === boardLineId)?.name}
      onBack={backToBoard}
      backDisabled={!boardLineId}
    />
  );
}

type MobileTaskModel = {
  fixture: SplitRunFixture;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading: boolean;
  pullRequests: FactoriesFactoryPullRequest[];
  canUpdate: boolean;
  lineName?: string;
  planningEnabled: boolean;
  dispatchDraft?: (model?: string, thinkingLevel?: string) => Promise<void>;
  onArchive: ReturnType<typeof footerMutationHandlers>["onArchive"];
  onReject: ReturnType<typeof footerMutationHandlers>["onReject"];
  onStop: ReturnType<typeof footerMutationHandlers>["onStop"];
  startBusy: boolean;
  actionBusy: boolean;
  columnAppRunQueries: ReturnType<typeof useColumnAppCheckRuns>["queries"];
  footerActions: ReturnType<typeof useSplitRunFooterActions>;
  ownerEdits: Pick<
    ReturnType<typeof useSplitRunWorkOrderEdits>,
    "owner" | "assigneeIds" | "canEdit" | "ownerBusy" | "saveOwner"
  >;
};

/** Loads everything the task screen shows and wires the footer actions. */
function useMobileTaskModel(
  order: FactoriesWorkOrder,
  orderId: string,
  lineId: string | undefined,
  rawLineName: string | undefined,
  onDone: () => void,
): MobileTaskModel {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const canUpdate = canAct("work_orders", "update");
  const lineName = rawLineName?.trim() || undefined;
  const { data: artifacts = [], isLoading: artifactsLoading } = useWorkOrderArtifacts(
    organizationId,
    factoryId,
    orderId,
  );
  const { data: handlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const pullRequests = order.pullRequests ?? [];
  const prFeedbackRuns = useWorkOrderPRFeedbackLog(pullRequests, handlers);
  const closer = useSplitRunFooterCloser(organizationId, factoryId, order);
  const { resolveUser } = useOrgUserLookup(organizationId);
  const backlogAnalysis = useFactoryBacklogAnalysis(organizationId, factoryId);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const columnApps = columnAppsFromFactoryApps(apps);
  const columnAppCheckRuns = useColumnAppCheckRuns(order.checks, columnApps, pullRequests);

  const fixture = splitRunFixtureForWorkOrder(order, {
    checks: order.checks,
    artifacts,
    lineId,
    lineName,
    demoArtifacts: false,
    prFeedbackRuns,
    analysisRuns: backlogAnalysis.runsByWorkOrder.get(orderId) ?? [],
    columnApps,
    isAnalyzing: backlogAnalysis.analyzingOrderIds.has(orderId),
    stoppedBy: closer.actor,
    closer,
    resolveUser,
    columnAppRuns: columnAppCheckRuns.lookup,
  });
  const ownerEdits = useSplitRunWorkOrderEdits({
    organizationId,
    factoryId,
    orderId,
    canUpdate,
    title: fixture.title,
    description: fixture.descriptionText ?? "",
    owner: fixture.owner,
    assigneeIds: fixture.assigneeIds ?? [],
    footerKind: fixture.footer.kind,
  });
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, onDone);
  const dispatchDraft = lineName
    ? (model?: string, thinkingLevel?: string) => cardActions.onDispatch(orderId, { lineName, model, thinkingLevel })
    : undefined;

  return {
    fixture,
    artifacts,
    artifactsLoading,
    pullRequests,
    canUpdate,
    lineName,
    planningEnabled: factoryPlanningEnabled(factory),
    dispatchDraft,
    onArchive: mutations.onArchive,
    onReject: mutations.onReject,
    onStop: mutations.onStop,
    startBusy: cardActions.dispatchingOrderIds.has(orderId),
    actionBusy: footerActions.busy,
    columnAppRunQueries: columnAppCheckRuns.queries,
    footerActions,
    ownerEdits,
  };
}

function LoadedMobileTask({
  order,
  orderId,
  lineId,
  lineName,
  onBack,
  backDisabled,
}: {
  order: FactoriesWorkOrder;
  orderId: string;
  lineId?: string;
  lineName?: string;
  onBack: () => void;
  backDisabled?: boolean;
}) {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const model = useMobileTaskModel(order, orderId, lineId, lineName, onBack);
  const { fixture, artifacts, pullRequests } = model;
  const draftStart = useMobileDraftStart(model, lineId || lineName || "");
  const refine = useMobileRefineChat({
    organizationId,
    factoryId,
    factoryKey: routeSegment,
    factory,
    orderId,
    fixture,
    artifacts,
    canUpdate: model.canUpdate,
    modelSelect: draftStart.modelSelect,
  });
  const review = (withModelSelect: boolean) => (
    <PhoneTaskReview
      model={model}
      draftStart={draftStart}
      scope={{ organizationId, factoryId, factoryKey: routeSegment, orderId, orderNumber: order.number }}
      withModelSelect={withModelSelect}
    />
  );
  const canStopRun = Boolean(model.canUpdate && organizationId && factoryId && orderId);
  const onStopRun = (run: { appId: string; runId: string }) => void model.footerActions.handleStopAutomation(run);
  const onRerunStep = (phase: { stepIndex?: number | null }) =>
    void model.footerActions.handleStop("rerun-step", {
      kind: "failed",
      lineName: fixture.lineName,
      stepIndex: phase.stepIndex ?? undefined,
    });
  if (refine.chat) {
    return (
      <MobileTaskFrame onBack={onBack} backDisabled={backDisabled} scroll={false}>
        {model.columnAppRunQueries}
        <MobileRefineBody
          order={order}
          orderId={orderId}
          model={model}
          chat={refine.chat}
          loadFailed={refine.loadFailed}
          review={review(false)}
          summary={
            <div className="mb-5" data-testid="mobile-task-detail">
              <MobileTaskHeader
                order={order}
                fixture={fixture}
                organizationId={organizationId}
                ownerEdits={model.ownerEdits}
              />
            </div>
          }
        />
      </MobileTaskFrame>
    );
  }

  return (
    <MobileTaskFrame onBack={onBack} backDisabled={backDisabled}>
      {model.columnAppRunQueries}
      <article className="flex flex-col gap-5 px-4 pt-3 pb-8" data-testid="mobile-task-detail">
        <MobileTaskHeader
          order={order}
          fixture={fixture}
          organizationId={organizationId}
          ownerEdits={model.ownerEdits}
        />
        {refine.sessionMissing ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {MOBILE_TASK_COPY.noRefinementSession}
          </p>
        ) : null}
        {review(true)}
        <Section title={MOBILE_TASK_COPY.description}>
          {fixture.descriptionText?.trim() ? (
            <MarkdownContent content={fixture.descriptionText} variant="workspace" organizationId={organizationId} />
          ) : (
            <p className="text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.noDescription}</p>
          )}
        </Section>
        <MobileTaskSections
          order={order}
          orderId={orderId}
          fixture={fixture}
          artifacts={artifacts}
          pullRequests={pullRequests}
          factoryKey={routeSegment}
          orderNumber={order.number ? String(order.number) : undefined}
          canStopRun={canStopRun}
          actionBusy={model.actionBusy}
          onStopRun={onStopRun}
          onRerunStep={onRerunStep}
        />
      </article>
    </MobileTaskFrame>
  );
}

/**
 * Start choice for a draft. The model resets when the start line changes so
 * Start never sends a model from another line.
 */
function useMobileDraftStart(model: MobileTaskModel, startLineKey: string) {
  const { organizationId, factoryId } = useFactoriesLayout();
  const [draftLineKey, setDraftLineKey] = useState(startLineKey);
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const [draftThinking, setDraftThinking] = useState(THINKING_LEVEL_MEDIUM);
  if (draftLineKey !== startLineKey) {
    setDraftLineKey(startLineKey);
    setDraftModel(DRAFT_START_MODEL_AUTO);
  }
  const { fixture } = model;
  const onStart = draftStartAction(
    fixture.footer.kind,
    model.dispatchDraft,
    draftModel,
    model.planningEnabled ? draftThinking : undefined,
  );
  const modelSelect = showsPhoneDraftModelSelect(model.planningEnabled, fixture.footer) ? (
    <MobileDraftStartModelSelect
      organizationId={organizationId}
      factoryId={factoryId}
      lineName={model.lineName}
      model={draftModel}
      thinkingLevel={draftThinking}
      disabled={!model.canUpdate || model.startBusy}
      onChange={({ model: nextModel, thinkingLevel }) => {
        setDraftModel(nextModel);
        setDraftThinking(thinkingLevel);
      }}
    />
  ) : undefined;
  return { onStart, modelSelect };
}

/**
 * Refine chat for a Planning draft. The task header scrolls above the
 * request in the chat log. Like the desktop popup, the chat shows the
 * agent's live work, not the automation phase list or its log. The
 * composer, the question form, and Start stay at the bottom of the frame,
 * above the bottom bar.
 */
function MobileRefineBody({
  order,
  orderId,
  model,
  chat,
  loadFailed,
  review,
  summary,
}: {
  order: FactoriesWorkOrder;
  orderId: string;
  model: MobileTaskModel;
  chat: IntentAnalysisChat;
  loadFailed: boolean;
  review: ReactNode;
  summary: ReactNode;
}) {
  const { organizationId, factoryId, routeSegment } = useFactoriesLayout();
  const { fixture } = model;
  return (
    <>
      {loadFailed ? (
        <p className="shrink-0 px-4 pt-3 text-[13px] text-destructive" role="alert">
          {ANALYSIS_PLANNING_COPY.failedLoadRefinement}
        </p>
      ) : null}
      <WorkOrderSplitRunOverview
        title={fixture.title}
        description={fixture.descriptionText ?? ""}
        artifacts={model.artifacts}
        artifactsLoading={model.artifactsLoading}
        pullRequests={model.pullRequests}
        checks={fixture.checks}
        isAnalyzing={fixture.footer.note?.headline === SPLIT_RUN_ANALYZING_NOTE.headline}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={routeSegment}
        orderId={orderId}
        orderNumber={order.number}
        files={order.files}
        expandFirstCheck
        resultFooter={review}
        analysis={{ ...chat, leading: summary }}
        source={fixture.source}
      />
    </>
  );
}

function showsPhoneDraftModelSelect(planningEnabled: boolean, footer: SplitRunFixture["footer"]): boolean {
  return planningEnabled && footer.kind === "draft" && footer.actions.some((action) => action.kind === "start");
}

/** Key, status, full title, and the owner/time/cost row. The title wraps instead of truncating. */
function MobileTaskHeader({
  order,
  fixture,
  organizationId,
  ownerEdits,
}: {
  order: FactoriesWorkOrder;
  fixture: SplitRunFixture;
  organizationId: string;
  ownerEdits: MobileTaskModel["ownerEdits"];
}) {
  const { factoryKey } = useFactoriesLayout();
  const statusMeta = getWorkOrderDisplayStatusMeta(getWorkOrderDisplayStatus(order));
  const identifier = fixture.identifier?.trim() || formatWorkOrderIdentifier(factoryKey, order.number);

  return (
    <header className="flex flex-col gap-2">
      <div className="flex items-center gap-2 text-[12px]">
        {identifier ? (
          <span className="font-mono text-muted-foreground" data-testid="mobile-task-key">
            {identifier}
          </span>
        ) : null}
        <span
          className={cn("inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 font-medium", statusMeta.className)}
          data-testid="mobile-task-status"
        >
          <span className={cn("size-1.5 rounded-full", statusMeta.dotClassName)} aria-hidden />
          {statusMeta.label}
        </span>
      </div>
      <h1 className="text-[20px] leading-snug font-semibold tracking-[-0.02em] text-foreground break-words">
        {fixture.title}
      </h1>
      <OwnerTimeCostRow
        fixture={{ ...fixture, owner: ownerEdits.owner }}
        className="mt-0"
        organizationId={organizationId}
        canEditOwner={ownerEdits.canEdit}
        assigneeIds={ownerEdits.assigneeIds}
        ownerBusy={ownerEdits.ownerBusy}
        onOwnerSave={ownerEdits.saveOwner}
        usageByModel={fixture.usageByModel}
        usageByMachineType={fixture.usageByMachineType}
      >
        <span className="text-muted-foreground">
          {fixture.lineName} · {fixture.startedLabel}
        </span>
      </OwnerTimeCostRow>
    </header>
  );
}

/**
 * Back row and task body above the bottom bar. With `scroll` off the body
 * is a fixed-height column: the refine chat scrolls its own log and keeps
 * the composer at the bottom, also when the keyboard shrinks the shell.
 */
function MobileTaskFrame({
  onBack,
  backDisabled = false,
  scroll = true,
  children,
}: {
  onBack: () => void;
  backDisabled?: boolean;
  scroll?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="mobile-task-page">
      <div className="flex h-12 shrink-0 items-center border-b border-border px-2 pt-[env(safe-area-inset-top)]">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onBack}
          disabled={backDisabled}
          className="gap-1.5 text-muted-foreground"
          data-testid="mobile-task-back"
        >
          <ArrowLeft className="size-4" aria-hidden />
          {MOBILE_TASK_COPY.back}
        </Button>
      </div>
      <div
        className={cn(
          "min-h-0 flex-1 pr-[env(safe-area-inset-right)] pl-[env(safe-area-inset-left)]",
          scroll ? "overflow-y-auto" : "flex flex-col overflow-hidden",
        )}
        data-testid="mobile-task-body"
      >
        {children}
      </div>
    </div>
  );
}
