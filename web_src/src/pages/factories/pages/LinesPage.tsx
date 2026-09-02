import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import {
  useFactoryApps,
  useFactoryPullRequests,
  useFactoryWorkOrders,
  useUpdateFactoryLine,
} from "@/hooks/useFactoryData";
import { useCreateFactoryPRFeedbackHandler, useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useCreateFactoryIntake, useFactoryIntakes } from "@/hooks/useFactoryIntakeData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useMe } from "@/hooks/useMe";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { cn } from "@/lib/utils";
import { FEATURE_FACTORY_SENTRY_INTAKE } from "@/lib/experimentalFeatures";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Clock, MoreHorizontal, Pencil, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useLocation, useNavigate, useParams } from "react-router";
import type { BacklogAnalysisRun } from "../lib/backlogAnalysis";
import { ClickToRename } from "../layout/ClickToRename";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { AddIntakePicker } from "./AddIntakePicker";
import { AddPRFeedbackPicker } from "./AddPRFeedbackPicker";
import { BacklogColumn, type BacklogIntakePanel } from "./BacklogColumn";
import { LineBoardOrderCard, LineBoardWorkOrderCard } from "./LineBoardOrderCard";
import {
  buildLinePhaseBoard,
  collectLineBacklogOrders,
  collectLineDoneOrders,
  collectLineVerifyOrders,
  findBacklogAutomationApp,
  findClosureAutomationApp,
  isDoneLineColumn,
  LINE_PHASE_RUNS_PAGE_SIZE,
  visibleLineStageColumns,
  resolveColumnGlyph,
  resolvePhaseRunStatus,
  type LinePhaseColumn,
  type LinePhaseRunCard,
  type PhaseGlyphKind,
} from "../lib/linePhaseRuns";
import { flattenWorkOrderExecutions, isQueuedStepRow } from "../lib/workOrderExecutions";
import {
  latestDispatchForLine,
  canonicalWorkOrderNumber,
  peekOrderFromNavigationState,
  resolvePeekWorkOrder,
  resolveWorkOrderByNumber,
  workOrderRouteNeedsCanonicalRedirect,
} from "../lib/workOrderNumberResolution";
import {
  applyWorkOrderFilters,
  applyWorkOrderScope,
  applyWorkOrderSearch,
  buildWorkOrderListEntries,
  WORK_ORDER_SCOPES,
} from "../lib/workOrderListModel";
import { useWorkOrderListState, type WorkOrderListState } from "../lib/useWorkOrderListState";
import { useWorkOrdersHeaderShortcuts } from "../lib/useWorkOrdersHeaderShortcuts";
import { buildAssigneeFilterOptions } from "../lib/workOrderFilterOptions";
import { FilterChips } from "../workOrders/header/FilterChips";
import { FilterMenu } from "../workOrders/header/FilterMenu";
import { ScopePills } from "../workOrders/header/ScopePills";
import { SearchField } from "../workOrders/header/SearchField";
import {
  WorkOrderBoardLane,
  WorkOrderKanbanBoard,
  workOrderKanbanLaneScrollClassName,
  workOrderKanbanLaneSizeClassName,
  type BoardLaneTone,
} from "../workOrders/WorkOrderBoardChrome";
import { type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";
import { canvasKeyForAutomation, type SplitRunCanvasKey } from "./work-order-split-run/splitRunCanvases";
import { splitRunFixtureForWorkOrder } from "./work-order-split-run/splitRunMocks";
import { useSplitRunFooterCloser } from "./work-order-split-run/useSplitRunFooterCloser";
import {
  editFactoryLinePath,
  factoryAppConfigurePath,
  factoryAppRunPath,
  factoryHomePath,
  factoryIntakePath,
  factoryPRFeedbackPath,
  firstFactoryLineId,
  workOrderDetailPath,
  workOrderBoardLineIdFromSearch,
  intakeIdFromSearch,
  intakeSettingsTabFromSearch,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
  prFeedbackHandlerIdFromSearch,
  prFeedbackSettingsTabFromSearch,
} from "../lib/factoryPagePaths";
import { humanizeLineName } from "../lib/humanizeLineName";
import {
  factoryKanbanPageClassName,
  factorySectionHeaderClassName,
  factoryWorkOrdersBodyClassName,
} from "./factoryPageLayoutStyles";
import { replaceLineStepParallelism } from "../lib/factoryLineFormShared";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import { ParallelismSettingsDialog } from "./ParallelismSettingsDialog";
import { PlanningReviewPopup } from "./PlanningReviewPopup";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import {
  ADD_INTAKE_TEMPLATES,
  apiIntakeSource,
  intakeSourcesFromFactoryIntakes,
  isLineIntakeSourceId,
  type AddIntakeTemplate,
} from "./lineIntakeModel";
import { isIntakeSettingsTab } from "./intakeSourceSettingsModel";
import { useFactoryPreviewFlag } from "./factoryPreviewFlagsContext";
import { IntakeSettingsHost } from "./IntakeSettingsHost";
import { PRFeedbackSettingsHost } from "./PRFeedbackSettingsHost";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  apiPRFeedbackSource,
  hasAvailablePRFeedbackSource,
  isPRFeedbackSettingsTab,
  prFeedbackListenTitle,
  takenPRFeedbackSourceIds,
  type PRFeedbackSource,
} from "./prFeedbackSettingsModel";
import { LaneListenerList, type LaneListener } from "./LaneListenerList";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { usePRFeedbackWorkOrderAttention, useWorkOrderPRFeedbackLog } from "./useWorkOrderPRFeedbackRunHref";
import {
  normalizeColumnColors,
  serializeColumnColors,
  lineBoardColumnLaneClassName,
  type LineBoardColumnColorId,
} from "./lineBoardColumnColors";

function applyVisibleWorkOrders(
  workOrders: FactoriesWorkOrder[],
  factory: FactoriesFactory | null | undefined,
  state: WorkOrderListState,
  currentUserId?: string,
): FactoriesWorkOrder[] {
  const entries = factory ? buildWorkOrderListEntries(workOrders, factory) : [];
  const visibleIds = new Set(
    applyWorkOrderSearch(
      applyWorkOrderFilters(applyWorkOrderScope(entries, state.scope, currentUserId), {
        ...state.filters,
        lineIds: [],
      }),
      state.search,
    ).map((entry) => entry.id),
  );
  return workOrders.filter((order) => {
    const id = order.id;
    if (!id) {
      return false;
    }
    return visibleIds.has(id);
  });
}

export function LinesPage() {
  const { organizationId, factoryId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { lineId: routeLineId, orderNumber: routeOrderNumber } = useParams<{ lineId?: string; orderNumber?: string }>();
  const { search, state: locationState } = useLocation();
  const navigate = useNavigate();
  const intakeOpen = isIntakeSearchOpen(search);
  const intakeId = intakeIdFromSearch(search);
  const intakeSettingsTab = intakeSettingsTabFromSearch(search);
  const prFeedbackOpen = isPRFeedbackSearchOpen(search);
  const prFeedbackSettingsTab = prFeedbackSettingsTabFromSearch(search);
  const prFeedbackHandlerId = prFeedbackHandlerIdFromSearch(search);
  const { data: workOrders = [], isLoading: workOrdersLoading } = useFactoryWorkOrders(organizationId, factoryId);
  const { data: pullRequests = [] } = useFactoryPullRequests(organizationId, factoryId);
  const { data: factoryApps = [] } = useFactoryApps(organizationId, factoryId);
  const { data: me } = useMe(false);
  const { data: prFeedbackHandlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const listState = useWorkOrderListState(factoryId);
  const { data: factoryIntakes = [] } = useFactoryIntakes(organizationId, factoryId);
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const createPRFeedbackHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);
  const configuredIntakes = useMemo(() => intakeSourcesFromFactoryIntakes(factoryIntakes), [factoryIntakes]);
  const showAddIntakeControl = useFactoryPreviewFlag("addIntakeControl");
  const canAddSentryIntake = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_SENTRY_INTAKE);
  const addIntakeTemplates = useMemo(() => {
    const allowedIds = new Set(canAddSentryIntake ? ["github-issues", "sentry-exceptions"] : ["github-issues"]);
    return ADD_INTAKE_TEMPLATES.filter((template) => allowedIds.has(template.id));
  }, [canAddSentryIntake]);
  const [addIntakeOpen, setAddIntakeOpen] = useState(false);
  const [addPRFeedbackOpen, setAddPRFeedbackOpen] = useState(false);
  const [peekHint, setPeekHint] = useState<FactoriesWorkOrder | null>(null);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const {
    addressingFeedbackOrderIds,
    addressingFeedbackLabels,
    waitingOnChecksOrderIds,
    checksPassedOrderIds,
    fixesPausedOrderIds,
  } = usePRFeedbackWorkOrderAttention(pullRequests);

  const canUpdate = canAct("factories", "update");
  const canUpdateWorkOrders = canAct("work_orders", "update");
  const canCreateWorkOrder = canAct("work_orders", "create");
  const visibleWorkOrders = useMemo(
    () => applyVisibleWorkOrders(workOrders, factory, listState, me?.id),
    [factory, listState.filters, listState.scope, listState.search, me?.id, workOrders],
  );
  const lines = useMemo(() => factory?.lines ?? [], [factory?.lines]);
  const permalink = useMemo(
    () => resolveWorkOrderByNumber(workOrders, routeOrderNumber, workOrdersLoading),
    [routeOrderNumber, workOrders, workOrdersLoading],
  );
  const boardLineId = workOrderBoardLineIdFromSearch(search);
  const searchLineId = lines.some((line) => line.id === boardLineId) ? boardLineId : undefined;
  const selectedLineId =
    routeLineId ??
    searchLineId ??
    latestDispatchForLine(permalink.order ?? undefined)?.line?.id ??
    firstFactoryLineId(factory);
  const selectedLine = useMemo(
    () => (selectedLineId ? (lines.find((line) => line.id === selectedLineId) ?? null) : null),
    [lines, selectedLineId],
  );
  const peekOrder = resolvePeekWorkOrder(
    permalink,
    routeOrderNumber,
    peekOrderFromNavigationState(locationState),
    peekHint,
  );

  usePageTitle([selectedLine ? humanizeLineName(selectedLine.name) : "Board", factory?.name ?? "Workspace"]);

  const canonicalNumber = canonicalWorkOrderNumber(permalink.order);
  if (routeOrderNumber && canonicalNumber && workOrderRouteNeedsCanonicalRedirect(permalink, routeOrderNumber)) {
    return <Navigate to={workOrderDetailPath(organizationId, factoryKey, canonicalNumber, boardLineId)} replace />;
  }

  if (!selectedLine) {
    if (routeOrderNumber && permalink.status === "loading") {
      return (
        <div className="flex h-full min-h-0 min-w-0 w-full" data-testid="lines-detail-page">
          <p className="px-6 py-8 text-[13px] text-muted-foreground">Loading task…</p>
        </div>
      );
    }
    return <Navigate to={factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory))} replace />;
  }

  const settingsIntake = intakeOpen ? configuredIntakes.find((intake) => intake.intakeId === intakeId) : undefined;

  const intakePanel: BacklogIntakePanel = {
    sources: configuredIntakes,
    showAddIntake: showAddIntakeControl,
    onOpenSettings: (intake) =>
      navigate(factoryIntakePath(organizationId, factoryKey, selectedLine.id, intake.intakeId)),
    onAddIntake: () => setAddIntakeOpen(true),
  };

  const takenPRFeedbackSources = takenPRFeedbackSourceIds(prFeedbackHandlers);
  const canAddPRFeedback = canUpdate && hasAvailablePRFeedbackSource(takenPRFeedbackSources);

  const verifyListeners: LaneListener[] = prFeedbackHandlers.flatMap((handler) => {
    if (!handler.id) {
      return [];
    }
    return [
      {
        id: handler.id,
        title: prFeedbackListenTitle(handler.source),
        iconSrc: githubIcon,
        iconAlt: "GitHub",
        healthy: handler.healthy !== false,
        needsRepairLabel: "Needs repair",
        settingsLabel: "Open PR feedback settings",
        testId: `lines-verify-listener-${handler.id}`,
        onOpenSettings: () =>
          navigate(factoryPRFeedbackPath(organizationId, factoryKey, selectedLine.id, undefined, handler.id)),
      },
    ];
  });

  const createPRFeedbackFromSource = (source: PRFeedbackSource) => {
    if (takenPRFeedbackSources.includes(source.id)) {
      return;
    }
    setAddPRFeedbackOpen(false);
    createPRFeedbackHandler
      .mutateAsync({ source: apiPRFeedbackSource(source.id), name: source.defaultName })
      .then((handler) => {
        if (!handler.id) {
          return;
        }
        navigate(factoryPRFeedbackPath(organizationId, factoryKey, selectedLine.id, undefined, handler.id));
      })
      .catch((error) => {
        showErrorToast(getApiErrorMessage(error, PR_FEEDBACK_SETTINGS_COPY.createError));
      });
  };

  const createIntakeFromTemplate = (template: AddIntakeTemplate) => {
    setAddIntakeOpen(false);
    if (!isLineIntakeSourceId(template.id)) {
      showErrorToast("This intake template is not available yet.");
      return;
    }
    createIntake
      .mutateAsync({ source: apiIntakeSource(template.id) })
      .then((intake) => {
        if (!intake.canvasId) {
          return;
        }
        navigate(
          factoryAppConfigurePath(organizationId, factoryKey, intake.canvasId, {
            from: "lines",
            lineId: selectedLine.id,
          }),
        );
      })
      .catch((error) => {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to create intake automation"));
      });
  };

  const openWorkOrder = (orderId: string, order?: FactoriesWorkOrder) => {
    const target = order ?? workOrders.find((item) => item.id === orderId) ?? { id: orderId };
    const number = canonicalWorkOrderNumber(target);
    if (!number) {
      setPeekHint(target);
      return;
    }
    navigate(workOrderDetailPath(organizationId, factoryKey, number, selectedLine.id), {
      state: { peekOrder: target },
    });
  };

  const closePeek = () => {
    setPeekHint(null);
    navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id), { replace: true });
  };

  return (
    <div className="flex h-full min-h-0 min-w-0 w-full" data-testid="lines-detail-page">
      {settingsIntake ? (
        <IntakeSettingsHost
          key={settingsIntake.intakeId}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          lineId={selectedLine.id}
          intake={settingsIntake}
          initialTab={isIntakeSettingsTab(intakeSettingsTab) ? intakeSettingsTab : "general"}
          onOpenRun={(run) => {
            if (!run.appId || !run.runId) {
              return;
            }
            navigate(
              factoryAppRunPath(organizationId, factoryKey, run.appId, run.runId, {
                from: "lines",
                lineId: selectedLine.id,
              }),
            );
          }}
          onClose={() => navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id))}
        />
      ) : null}
      <AddIntakePicker
        open={addIntakeOpen}
        onClose={() => setAddIntakeOpen(false)}
        onSelect={createIntakeFromTemplate}
        templates={addIntakeTemplates}
      />
      <AddPRFeedbackPicker
        open={addPRFeedbackOpen}
        onClose={() => setAddPRFeedbackOpen(false)}
        onSelect={createPRFeedbackFromSource}
        takenSourceIds={takenPRFeedbackSources}
      />
      {prFeedbackOpen ? (
        <PRFeedbackSettingsHost
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          lineId={selectedLine.id}
          canUpdate={canUpdate}
          handlerId={prFeedbackHandlerId}
          initialTab={isPRFeedbackSettingsTab(prFeedbackSettingsTab) ? prFeedbackSettingsTab : "general"}
          onCreated={(handlerId) =>
            navigate(factoryPRFeedbackPath(organizationId, factoryKey, selectedLine.id, undefined, handlerId), {
              replace: true,
            })
          }
          onClose={() => navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id))}
        />
      ) : null}
      <div className={factoryKanbanPageClassName}>
        <div className="shrink-0">
          <LineDetailHeader
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={factoryKey}
            line={selectedLine}
            workOrders={workOrders}
            factory={factory}
            state={listState}
            canUpdate={canUpdate}
          />
        </div>
        <div className={factoryWorkOrdersBodyClassName}>
          <LineDetail
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={factoryKey}
            line={selectedLine}
            apps={factoryApps}
            workOrders={visibleWorkOrders}
            canCreateWorkOrder={canCreateWorkOrder || permissionsLoading}
            canUpdate={canUpdate}
            onCreateWorkOrder={openCreateWorkOrder}
            intakePanel={intakePanel}
            onAddIntake={canAddSentryIntake ? () => setAddIntakeOpen(true) : undefined}
            verifyListeners={verifyListeners}
            onAddPRFeedback={canAddPRFeedback ? () => setAddPRFeedbackOpen(true) : undefined}
            workOrderCardContext={{
              organizationId,
              factoryId,
              factoryKey,
              factoryLines: lines,
              canDispatch: canUpdateWorkOrders,
              preferredLineName: selectedLine.name,
              canAssign: canUpdateWorkOrders,
              addressingFeedbackOrderIds,
              addressingFeedbackLabels,
              waitingOnChecksOrderIds,
              checksPassedOrderIds,
              fixesPausedOrderIds,
              ...cardActions,
            }}
            peekOrder={peekOrder ?? undefined}
            onOpenWorkOrder={openWorkOrder}
            onClosePeek={closePeek}
          />
        </div>
      </div>
    </div>
  );
}

function LineDetailHeader({
  organizationId,
  factoryId,
  factoryKey,
  line,
  workOrders,
  factory,
  state,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  workOrders: FactoriesWorkOrder[];
  factory: FactoriesFactory | null;
  state: WorkOrderListState;
  canUpdate: boolean;
}) {
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const searchRef = useWorkOrdersHeaderShortcuts(state);
  const entries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [factory, workOrders]);
  const assigneeOptions = buildAssigneeFilterOptions(entries);
  const title = humanizeLineName(line.name);
  const editHref = line.id ? editFactoryLinePath(organizationId, factoryKey, line.id) : "#";

  const handleRename = async (name: string) => {
    if (!line.id) {
      return;
    }
    try {
      await updateLine.mutateAsync({ lineId: line.id, name });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename line"));
    }
  };

  return (
    <WorkspacePageHeader
      className={factorySectionHeaderClassName}
      data-testid="lines-detail-header"
      title={
        <ClickToRename
          value={title}
          onSave={(name) => void handleRename(name)}
          canEdit={canUpdate && Boolean(line.id)}
          busy={updateLine.isPending}
          testId="lines-board-title"
          ariaLabel="Line name"
          inputClassName="font-medium text-[length:var(--workspace-page-title-size)] leading-[var(--workspace-page-title-line-height)] tracking-[var(--workspace-page-title-tracking)]"
        />
      }
      leading={
        <>
          <ScopePills
            value={state.scope}
            onChange={state.setScope}
            options={WORK_ORDER_SCOPES}
            testIdPrefix="work-orders-scope"
          />
          <FilterMenu state={state} assigneeOptions={assigneeOptions} />
        </>
      }
      actions={
        <>
          <SearchField
            inputRef={searchRef}
            open={state.searchOpen}
            value={state.search}
            onOpen={state.openSearch}
            onChange={state.setSearch}
            onClose={state.closeSearch}
          />
          {canUpdate && line.id ? <ColumnConfigureMenu title={title} href={editHref} testId="lines-edit-menu" /> : null}
        </>
      }
      belowRow={<FilterChips state={state} assigneeOptions={assigneeOptions} />}
    />
  );
}

function LineDetail({
  organizationId,
  factoryId,
  factoryKey,
  line,
  apps,
  workOrders,
  canCreateWorkOrder,
  canUpdate,
  onCreateWorkOrder,
  intakePanel,
  onAddIntake,
  verifyListeners,
  onAddPRFeedback,
  workOrderCardContext,
  peekOrder,
  onOpenWorkOrder,
  onClosePeek,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  workOrders: FactoriesWorkOrder[];
  canCreateWorkOrder: boolean;
  canUpdate: boolean;
  onCreateWorkOrder: () => void;
  intakePanel: BacklogIntakePanel;
  onAddIntake?: () => void;
  verifyListeners: LaneListener[];
  onAddPRFeedback?: () => void;
  workOrderCardContext: WorkOrderCardContext;
  peekOrder?: FactoriesWorkOrder | null;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  onClosePeek: () => void;
}) {
  const steps = line.steps ?? [];
  const fullBoard = useMemo(() => buildLinePhaseBoard(line, workOrders ?? [], apps), [line, workOrders, apps]);
  const verifyOrders = useMemo(() => collectLineVerifyOrders(fullBoard), [fullBoard]);
  const board = useMemo(() => visibleLineStageColumns(fullBoard, verifyOrders), [fullBoard, verifyOrders]);
  const backlogOrders = useMemo(() => collectLineBacklogOrders(workOrders ?? []), [workOrders]);
  const doneOrders = useMemo(
    () => collectLineDoneOrders(workOrders ?? [], line, fullBoard),
    [workOrders, line, fullBoard],
  );
  const peekOrderId = peekOrder?.id ?? null;
  const backlogAnalysis = useFactoryBacklogAnalysis(organizationId, factoryId);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="lines-detail">
      {steps.length === 0 && backlogOrders.length === 0 && doneOrders.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">No phases yet. Edit this line to add app-driven phases.</p>
      ) : (
        <PhaseBoard
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          line={line}
          apps={apps}
          backlogOrders={backlogOrders}
          verifyOrders={verifyOrders}
          doneOrders={doneOrders}
          columns={board}
          canCreateWorkOrder={canCreateWorkOrder}
          canRename={canUpdate}
          onCreateWorkOrder={onCreateWorkOrder}
          intakePanel={intakePanel}
          onAddIntake={onAddIntake}
          verifyListeners={verifyListeners}
          onAddPRFeedback={onAddPRFeedback}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          analyzingOrderIds={backlogAnalysis.analyzingOrderIds}
        />
      )}
      {peekOrderId && peekOrder ? (
        <LineBoardSplitRunPopup
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          lineId={line.id}
          lineName={line.name}
          peekOrderId={peekOrderId}
          peekOrder={peekOrder}
          canDispatch={workOrderCardContext.canDispatch}
          canUpdate={workOrderCardContext.canAssign}
          isDispatching={workOrderCardContext.dispatchingOrderIds.has(peekOrderId)}
          onDispatch={workOrderCardContext.onDispatch}
          analysisRuns={backlogAnalysis.runsByWorkOrder.get(peekOrderId) ?? []}
          onClose={onClosePeek}
        />
      ) : null}
    </div>
  );
}

function LineBoardSplitRunPopup({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  lineName,
  peekOrderId,
  peekOrder,
  canDispatch,
  canUpdate,
  isDispatching,
  onDispatch,
  analysisRuns,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId: string | undefined;
  lineName: string | undefined;
  peekOrderId: string;
  peekOrder: FactoriesWorkOrder;
  canDispatch: boolean;
  canUpdate: boolean;
  isDispatching: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  analysisRuns: BacklogAnalysisRun[];
  onClose: () => void;
}) {
  const { data: peekChecks = [] } = useWorkOrderChecks(organizationId, factoryId, peekOrderId);
  const { data: peekPullRequests = [] } = useFactoryPullRequests(organizationId, factoryId, {
    workOrderIds: [peekOrderId],
  });
  const { data: peekHandlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const prFeedbackRuns = useWorkOrderPRFeedbackLog(peekPullRequests, peekHandlers);
  const closer = useSplitRunFooterCloser(organizationId, factoryId, peekOrder);
  const resolvedLineName = lineName?.trim();
  return (
    <WorkOrderSplitRunPopup
      key={peekOrderId}
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      orderId={peekOrderId}
      orderNumber={peekOrder.number}
      lineId={lineId}
      fixture={splitRunFixtureForWorkOrder(peekOrder, {
        checks: peekChecks,
        lineId,
        demoArtifacts: false,
        prFeedbackRuns,
        analysisRuns,
        stoppedBy: closer.actor,
        closer,
      })}
      canDispatch={canDispatch && Boolean(resolvedLineName)}
      canUpdate={canUpdate}
      isDispatching={isDispatching}
      onDispatch={resolvedLineName ? () => onDispatch(peekOrderId, { lineName: resolvedLineName }) : undefined}
      onClose={onClose}
      fixed
    />
  );
}

function canvasAppIdsForLine(
  line: FactoriesFactoryLine,
  apps: Array<{ id?: string; name?: string }>,
): Record<SplitRunCanvasKey, string | undefined> {
  const backlog = findBacklogAutomationApp(apps);
  const closure = findClosureAutomationApp(apps);
  const ids: Record<SplitRunCanvasKey, string | undefined> = {
    intake: backlog?.id,
    sentry: appIdNamed(apps, "Sentry"),
    slack: appIdNamed(apps, "Slack"),
    planning: undefined,
    implementation: undefined,
    risk: undefined,
    closure: closure?.id,
  };
  for (const step of line.steps ?? []) {
    const appId = step.app?.app;
    if (!appId) {
      continue;
    }
    const app = apps.find((entry) => entry.id === appId);
    const key = canvasKeyForAutomation({
      id: appId,
      name: app?.name,
    });
    if (key === "planning" || key === "implementation" || key === "risk" || key === "closure") {
      ids[key] = appId;
    }
  }
  return ids;
}

function appIdNamed(apps: Array<{ id?: string; name?: string }>, name: string): string | undefined {
  return apps.find((app) => app.id && app.name === name)?.id;
}

export type CanvasExpandHref = (
  key: SplitRunCanvasKey,
  phase?: { appId?: string; runId?: string },
) => string | undefined;

export function canvasExpandHrefForLine(
  organizationId: string,
  factoryKey: string,
  line: FactoriesFactoryLine,
  apps: Array<{ id?: string; name?: string }>,
  order: FactoriesWorkOrder | undefined,
): CanvasExpandHref {
  const appIdByCanvas = canvasAppIdsForLine(line, apps);

  return (key, phase) => {
    const appId = phase?.appId ?? appIdByCanvas[key];
    const runId = phase?.runId ?? executionRunIdForCanvas(order, key, line.id, appId);
    if (!appId || !runId) {
      return undefined;
    }
    return factoryAppRunPath(organizationId, factoryKey, appId, runId, {
      from: "lines",
      lineId: line.id,
      orderNumber: order?.number,
    });
  };
}

function executionRunIdForCanvas(
  order: FactoriesWorkOrder | undefined,
  key: SplitRunCanvasKey,
  lineId: string | undefined,
  appId?: string,
): string | undefined {
  const dispatchExecutions = latestDispatchForLine(order, lineId)?.stepExecutions ?? [];
  const allExecutions = order ? flattenWorkOrderExecutions(order) : [];
  return (
    runIdMatchingApp(dispatchExecutions, appId) ??
    runIdMatchingApp(allExecutions, appId) ??
    runIdMatchingCanvasKey(dispatchExecutions, key) ??
    runIdMatchingCanvasKey(allExecutions, key)
  );
}

function runIdMatchingApp(
  executions: Array<{ run?: { id?: string; appId?: string } }>,
  appId: string | undefined,
): string | undefined {
  if (!appId) {
    return undefined;
  }
  return [...executions].reverse().find((execution) => execution.run?.appId === appId)?.run?.id;
}

function runIdMatchingCanvasKey(
  executions: Array<{ step?: string; run?: { id?: string; appId?: string; appName?: string } }>,
  key: SplitRunCanvasKey,
): string | undefined {
  return [...executions].reverse().find((execution) => {
    const matched = canvasKeyForAutomation({
      id: execution.run?.appId,
      name: execution.run?.appName ?? execution.step,
    });
    return matched === key;
  })?.run?.id;
}

function PhaseBoard({
  organizationId,
  factoryId,
  factoryKey,
  line,
  apps,
  backlogOrders,
  verifyOrders,
  doneOrders,
  columns,
  canCreateWorkOrder,
  canRename,
  onCreateWorkOrder,
  intakePanel,
  onAddIntake,
  verifyListeners,
  onAddPRFeedback,
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  backlogOrders: FactoriesWorkOrder[];
  verifyOrders: FactoriesWorkOrder[];
  doneOrders: FactoriesWorkOrder[];
  columns: LinePhaseColumn[];
  canCreateWorkOrder: boolean;
  canRename: boolean;
  onCreateWorkOrder: () => void;
  intakePanel: BacklogIntakePanel;
  onAddIntake?: () => void;
  verifyListeners: LaneListener[];
  onAddPRFeedback?: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  analyzingOrderIds: ReadonlySet<string>;
}) {
  const [columnColors, setColumnColors] = useState<Record<string, LineBoardColumnColorId | null>>(() =>
    normalizeColumnColors(line.columnColors),
  );
  const columnColorsRef = useRef(columnColors);
  const [columnTitles, setColumnTitles] = useState<Record<string, string>>({});
  const [backlogSize, setBacklogSize] = useState<number | null>(null);
  const [backlogSettingsOpen, setBacklogSettingsOpen] = useState(false);
  const [parallelismByStep, setParallelismByStep] = useState<Record<number, number>>({});
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const lineId = line.id;
  const backlogAutomationApp = findBacklogAutomationApp(apps);
  const backlogAutomationHref =
    backlogAutomationApp && lineId
      ? factoryAppConfigurePath(organizationId, factoryKey, backlogAutomationApp.id, {
          from: "lines",
          lineId,
        })
      : undefined;

  // The line query is the source of truth for persisted colors. Resync
  // when it refetches, but skip while a color save is in flight so a
  // stale cache cannot wipe the optimistic lane color.
  useEffect(() => {
    if (updateLine.isPending) {
      return;
    }
    const next = normalizeColumnColors(line.columnColors);
    columnColorsRef.current = next;
    setColumnColors(next);
  }, [line.columnColors, updateLine.isPending]);

  const setColumnColor = useCallback(
    async (columnKey: string, colorId: LineBoardColumnColorId | null) => {
      const previousColors = columnColorsRef.current;
      const nextColors = { ...previousColors, [columnKey]: colorId };
      columnColorsRef.current = nextColors;
      setColumnColors(nextColors);

      if (!lineId) {
        return;
      }
      try {
        await updateLine.mutateAsync({
          lineId,
          columnColors: serializeColumnColors(nextColors),
        });
      } catch (error) {
        columnColorsRef.current = previousColors;
        setColumnColors(previousColors);
        showErrorToast(getApiErrorMessage(error, "Failed to update column color"));
      }
    },
    [lineId, updateLine],
  );

  const setColumnTitle = useCallback((columnKey: string, title: string) => {
    setColumnTitles((current) => ({ ...current, [columnKey]: title }));
  }, []);

  const saveParallelism = useCallback(
    async (stepIndex: number, value: number) => {
      setParallelismByStep((current) => ({ ...current, [stepIndex]: value }));
      if (!line.id) {
        return;
      }
      try {
        await updateLine.mutateAsync({
          lineId: line.id,
          steps: replaceLineStepParallelism(line.steps, stepIndex, value),
        });
      } catch (error) {
        setParallelismByStep((current) => {
          const next = { ...current };
          delete next[stepIndex];
          return next;
        });
        showErrorToast(getApiErrorMessage(error, "Failed to update parallelism"));
      }
    },
    [line.id, line.steps, updateLine],
  );

  return (
    <WorkOrderKanbanBoard testId="lines-phase-board">
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <BacklogColumn
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          orders={backlogOrders}
          title={columnTitles.backlog ?? "Backlog"}
          size={backlogSize}
          settingsOpen={backlogSettingsOpen}
          onOpenSettings={() => setBacklogSettingsOpen(true)}
          onCloseSettings={() => setBacklogSettingsOpen(false)}
          onSaveSettings={({ name, size }) => {
            setColumnTitle("backlog", name);
            setBacklogSize(size);
            setBacklogSettingsOpen(false);
          }}
          colorId={columnColors.backlog ?? null}
          onColorChange={(colorId) => void setColumnColor("backlog", colorId)}
          canCreateWorkOrder={canCreateWorkOrder}
          canRename={canRename}
          onRename={(title) => setColumnTitle("backlog", title)}
          onCreateWorkOrder={onCreateWorkOrder}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          analyzingOrderIds={analyzingOrderIds}
          intakePanel={intakePanel}
          onAddIntake={onAddIntake}
          automationHref={backlogAutomationHref}
        />
      </div>
      {columns.map((column, index) => {
        const columnKey = `phase-${column.stepIndex}`;
        return (
          <div
            key={`${column.stepIndex}-${column.stepName}`}
            className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}
          >
            {index < columns.length - 1 || columns.length > 0 ? (
              <span className="absolute top-[21px] left-full z-[1] h-px w-3 bg-border" aria-hidden />
            ) : null}
            <PhaseColumn
              organizationId={organizationId}
              factoryKey={factoryKey}
              lineId={lineId}
              column={column}
              title={columnTitles[columnKey] ?? column.stepName}
              parallelism={parallelismByStep[column.stepIndex] ?? column.maxParallelism}
              onSaveParallelism={(value) => void saveParallelism(column.stepIndex, value)}
              colorId={columnColors[columnKey] ?? null}
              onColorChange={(colorId) => void setColumnColor(columnKey, colorId)}
              canRename={canRename}
              onRename={(title) => setColumnTitle(columnKey, title)}
              workOrderCardContext={workOrderCardContext}
              onOpenWorkOrder={onOpenWorkOrder}
            />
          </div>
        );
      })}
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <span className="absolute top-[21px] left-0 z-[1] h-px w-3 -translate-x-full bg-border" aria-hidden />
        <VerifyColumn
          orders={verifyOrders}
          title={columnTitles.verify ?? "Verify"}
          listeners={verifyListeners}
          onAdd={onAddPRFeedback}
          colorId={columnColors.verify ?? null}
          onColorChange={(colorId) => void setColumnColor("verify", colorId)}
          canRename={canRename}
          onRename={(title) => setColumnTitle("verify", title)}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
        />
      </div>
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <span className="absolute top-[21px] left-0 z-[1] h-px w-3 -translate-x-full bg-border" aria-hidden />
        <DoneColumn
          orders={doneOrders}
          title={columnTitles.done ?? "Done"}
          colorId={columnColors.done ?? null}
          onColorChange={(colorId) => void setColumnColor("done", colorId)}
          canRename={canRename}
          onRename={(title) => setColumnTitle("done", title)}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
        />
      </div>
    </WorkOrderKanbanBoard>
  );
}

function VerifyColumn({
  orders,
  title,
  listeners,
  onAdd,
  colorId,
  onColorChange,
  canRename,
  onRename,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  orders: FactoriesWorkOrder[];
  title: string;
  listeners: LaneListener[];
  onAdd?: () => void;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canRename: boolean;
  onRename: (title: string) => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
}) {
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);

  return (
    <WorkOrderBoardLane
      title={title}
      label={title}
      canRename={canRename}
      onRename={onRename}
      titleTestId="lines-column-title-verify"
      count={orders.length}
      tone="neutral"
      surfaceClassName={surfaceClassName}
      emptyDescription="No tasks in Verify."
      className={surfaceClassName ? undefined : "bg-muted"}
      actions={
        <div className="flex shrink-0 items-center gap-0.5">
          {onAdd ? (
            <button
              type="button"
              aria-label={PR_FEEDBACK_SETTINGS_COPY.addHandler}
              title={PR_FEEDBACK_SETTINGS_COPY.addHandler}
              data-testid="lines-verify-add-pr-feedback"
              onClick={onAdd}
              className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <Plus className="size-3.5" aria-hidden />
            </button>
          ) : null}
          <ColumnLaneMenu title={title} testId="lines-verify-menu" colorId={colorId} onColorChange={onColorChange} />
        </div>
      }
      banner={<LaneListenerList listeners={listeners} testId="lines-verify-listeners" />}
      testId="lines-verify-column"
    >
      <ul className={workOrderKanbanLaneScrollClassName} data-testid="lines-verify-column-scroll">
        {orders.map((order) => (
          <li key={order.id}>
            <LineBoardOrderCard
              order={order}
              workOrderCardContext={workOrderCardContext}
              onOpenWorkOrder={onOpenWorkOrder}
            />
          </li>
        ))}
      </ul>
    </WorkOrderBoardLane>
  );
}

function DoneColumn({
  orders,
  title,
  colorId,
  onColorChange,
  canRename,
  onRename,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  orders: FactoriesWorkOrder[];
  title: string;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canRename: boolean;
  onRename: (title: string) => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
}) {
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);

  return (
    <WorkOrderBoardLane
      title={title}
      label={title}
      canRename={canRename}
      onRename={onRename}
      titleTestId="lines-column-title-done"
      count={orders.length}
      tone="done"
      surfaceClassName={surfaceClassName}
      emptyDescription="No tasks in Done."
      className={surfaceClassName ? undefined : "bg-muted"}
      actions={
        <ColumnLaneMenu title={title} testId="lines-done-menu" colorId={colorId} onColorChange={onColorChange} />
      }
      testId="lines-done-column"
    >
      <ul className={workOrderKanbanLaneScrollClassName} data-testid="lines-done-column-scroll">
        {orders.map((order) => (
          <li key={order.id}>
            <LineBoardOrderCard
              order={order}
              workOrderCardContext={workOrderCardContext}
              onOpenWorkOrder={onOpenWorkOrder}
            />
          </li>
        ))}
      </ul>
    </WorkOrderBoardLane>
  );
}

function ColumnConfigureMenu({ title, href, testId }: { title: string; href: string; testId: string }) {
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`${title} menu`}
          className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid={testId}
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuItem onClick={() => navigate(href)} data-testid={`${testId}-edit`}>
          <Pencil className="h-3.5 w-3.5" aria-hidden />
          Edit
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Phase lanes borrow the Tasks lane tints: blue in flight, grey once closed. */
const PHASE_LANE_TONE: Record<PhaseGlyphKind, BoardLaneTone> = {
  running: "running",
  waiting: "running",
  queued: "running",
  failed: "done",
  passed: "done",
  cancelled: "done",
  pending: "neutral",
};

function PhaseColumn({
  organizationId,
  factoryKey,
  lineId,
  column,
  title,
  parallelism,
  onSaveParallelism,
  colorId,
  onColorChange,
  canRename,
  onRename,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  column: LinePhaseColumn;
  title: string;
  parallelism: number;
  onSaveParallelism: (value: number) => void;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canRename: boolean;
  onRename: (title: string) => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
}) {
  const scrollRef = useRef<HTMLUListElement>(null);
  const [visibleCount, setVisibleCount] = useState(LINE_PHASE_RUNS_PAGE_SIZE);
  const [parallelismOpen, setParallelismOpen] = useState(false);
  const agentEditor = useColumnCanvasAgentEditor(organizationId, column.appId);
  const totalRuns = column.runs.length;
  const hasMore = visibleCount < totalRuns;

  const loadMore = useCallback(() => {
    setVisibleCount((current) => Math.min(current + LINE_PHASE_RUNS_PAGE_SIZE, totalRuns));
  }, [totalRuns]);

  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore,
    onLoadMore: loadMore,
  });

  // When the window is short enough that the first page does not overflow,
  // pull the next page so a scrollbar can appear (same pattern as versions tab).
  useEffect(() => {
    loadMoreIfNeeded(scrollRef.current);
  }, [visibleCount, loadMoreIfNeeded]);

  const visibleRuns = column.runs.slice(0, Math.min(visibleCount, totalRuns));
  const configureHref =
    !isDoneLineColumn(column) && column.appId
      ? factoryAppConfigurePath(organizationId, factoryKey, column.appId, { from: "lines", lineId })
      : null;
  const glyph = resolveColumnGlyph(column);
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);

  return (
    <>
      <WorkOrderBoardLane
        title={title}
        label={`${title} phase`}
        count={totalRuns}
        tone={PHASE_LANE_TONE[glyph]}
        surfaceClassName={surfaceClassName}
        emptyDescription="Nothing here."
        canRename={canRename}
        onRename={onRename}
        titleTestId={`lines-column-title-phase-${column.stepIndex}`}
        testId={`lines-phase-column-${column.stepIndex}`}
        actions={
          <ColumnLaneMenu
            title={title}
            testId={`lines-phase-menu-${column.stepIndex}`}
            editHref={configureHref}
            editLabel={configureHref ? "Edit Automation" : undefined}
            onEditAgent={agentEditor.openEditor}
            onSetParallelism={configureHref ? () => setParallelismOpen(true) : undefined}
            parallelism={parallelism}
            colorId={colorId}
            onColorChange={onColorChange}
          />
        }
      >
        <ul
          ref={scrollRef}
          className={workOrderKanbanLaneScrollClassName}
          onScroll={(event) => loadMoreIfNeeded(event.currentTarget)}
          data-testid={`lines-phase-column-scroll-${column.stepIndex}`}
        >
          {visibleRuns.map((run) => (
            <li key={run.executionId}>
              <PhaseRunCard run={run} workOrderCardContext={workOrderCardContext} onOpenWorkOrder={onOpenWorkOrder} />
            </li>
          ))}
        </ul>
      </WorkOrderBoardLane>
      <ParallelismSettingsDialog
        open={parallelismOpen}
        value={parallelism}
        onSave={(value) => {
          onSaveParallelism(value);
          setParallelismOpen(false);
        }}
        onClose={() => setParallelismOpen(false)}
      />
      {agentEditor.editorOpen ? (
        <PlanningReviewPopup
          key={agentEditor.agentNode?.id ?? "agent"}
          onClose={agentEditor.closeEditor}
          organizationId={organizationId}
          automationHref={configureHref ?? undefined}
          initialDraft={agentEditor.draft ?? { title: "Editing Agent", components: [] }}
          isLoading={agentEditor.isLoading || !agentEditor.draft}
          onSave={agentEditor.save}
        />
      ) : null}
    </>
  );
}

function PhaseRunCard({
  run,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  run: LinePhaseRunCard;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
}) {
  const queuedLabel = isQueuedStepRow(run.execution) ? resolvePhaseRunStatus(run.execution).label : null;

  return (
    <div data-testid={`lines-phase-run-${run.executionId}`}>
      <LineBoardWorkOrderCard
        order={run.order}
        workOrderCardContext={workOrderCardContext}
        onOpen={() => {
          if (run.workOrderId) {
            onOpenWorkOrder(run.workOrderId, run.order);
          }
        }}
      />
      {queuedLabel ? (
        <p className="mt-1 flex items-center gap-1 px-0.5 text-[11px] text-muted-foreground">
          <Clock className="size-3 shrink-0" aria-hidden />
          {queuedLabel} — waiting for a free slot
        </p>
      ) : null}
    </div>
  );
}
