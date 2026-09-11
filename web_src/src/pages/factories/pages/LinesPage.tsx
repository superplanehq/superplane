import type {
  FactoriesFactory,
  FactoriesFactoryIntake,
  FactoriesFactoryLine,
  FactoriesFactoryPrFeedbackHandler,
  FactoriesWorkOrder,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import {
  useFactoryApps,
  useFactoryPullRequests,
  useFactoryWorkOrders,
  useUpdateFactoryLine,
} from "@/hooks/useFactoryData";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCreateFactoryIntake, useFactoryIntakes } from "@/hooks/useFactoryIntakeData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useMe } from "@/hooks/useMe";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { cn } from "@/lib/utils";
import {
  FEATURE_FACTORY_CREATE_WITH_AGENT,
  FEATURE_FACTORY_PRODUCTIVE_INTAKE,
  FEATURE_FACTORY_SENTRY_INTAKE,
} from "@/lib/experimentalFeatures";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { Clock, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Navigate, useLocation, useNavigate, useParams } from "react-router";
import type { BacklogAnalysisRun } from "../lib/backlogAnalysis";
import { ClickToRename } from "../layout/ClickToRename";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { useColumnAutomationViewPreference, type ColumnAutomationView } from "../lib/columnAutomationViewPreference";
import { useHostedCreditChrome } from "../lib/useHostedCreditEmptyBanner";
import { AddIntakePicker } from "./AddIntakePicker";
import { AddPRFeedbackPicker } from "./AddPRFeedbackPicker";
import { NextStepsPanel, WorkspaceNextStepsHeaderBadge } from "./NextStepsPanel";
import { useWorkspaceNextStepDeferral } from "./workspaceNextStepDeferral";
import {
  isWorkspaceNextStepDeferred,
  runWorkspaceNextStepAction,
  shouldForgetDeferredWorkspaceNextStep,
  isWorkspaceNextStepsQueryReady,
  workspaceNextStepBanner,
  workspaceNextSteps,
  workspaceNextStepsProgressCopy,
} from "./workspaceNextStepCatalog";
import { BacklogColumn, type BacklogIntakePanel } from "./BacklogColumn";
import { LineBoardViewMenu } from "./LineBoardViewMenu";
import { columnAutomationRowsSubheader } from "./columnAutomationRowsSubheader";
import { ColumnAutomationsHeaderSlot } from "./ColumnAutomationsIndicator";
import type { ColumnAutomationRowAction } from "./ColumnAutomationsPopup";
import { CreateWithAgentDialog } from "./CreateWithAgentDialog";
import { LineBoardOrderCard, LineBoardWorkOrderCard } from "./LineBoardOrderCard";
import { workspacePlanningRepository } from "./planningSessionView";
import { useCreateWithAgentSession } from "./useCreateWithAgentSession";
import { usePlanningSessionLiveRun } from "./usePlanningSessionLiveRun";
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
  factoryAppConfigurePath,
  factoryAppRunPath,
  factoryHomePath,
  factoryIntakePath,
  factoryPRFeedbackPath,
  factoryPRFeedbackSetupPath,
  columnAutomationViewCanvasIdFromSearch,
  firstFactoryLineId,
  workOrderDetailPath,
  workOrderBoardLineIdFromSearch,
  intakeIdFromSearch,
  intakeSettingsTabFromSearch,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
  prFeedbackHandlerIdFromSearch,
  prFeedbackSettingsTabFromSearch,
  prFeedbackSetupKindFromSourceId,
} from "../lib/factoryPagePaths";
import { humanizeLineName } from "../lib/humanizeLineName";
import {
  factoryKanbanPageClassName,
  factorySectionHeaderClassName,
  factoryWorkOrdersBodyClassName,
} from "./factoryPageLayoutStyles";
import {
  applyColumnAutomationsOverlay,
  buildColumnAutomations,
  columnAutomationOpenPath,
  type ColumnAutomation,
  type ColumnKey,
} from "../lib/columnAutomations";
import { columnAutomationHeaderRowCount } from "../lib/columnAutomationHeadline";
import { replaceLineStepParallelism } from "../lib/factoryLineFormShared";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import { ParallelismSettingsDialog } from "./ParallelismSettingsDialog";
import { ProductiveIntakeSetupDialog } from "./ProductiveIntakeSetupDialog";
import {
  ADD_INTAKE_TEMPLATES,
  apiIntakeSource,
  intakeSourcesFromFactoryIntakes,
  isLineIntakeSourceId,
  type AddIntakeTemplate,
} from "./lineIntakeModel";
import { isIntakeSettingsTab } from "./intakeSourceSettingsModel";
import { useFactoryPreviewFlag } from "./factoryPreviewFlagsContext";
import { ColumnAutomationViewHost } from "./ColumnAutomationViewPopup";
import { IntakeSettingsHost } from "./IntakeSettingsHost";
import { PRFeedbackSettingsHost } from "./PRFeedbackSettingsHost";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  hasAvailablePRFeedbackSource,
  isPRFeedbackSetupAvailable,
  isPRFeedbackSettingsTab,
  prFeedbackListenTitle,
  takenPRFeedbackSourceIds,
  type PRFeedbackSource,
  type PRFeedbackSourceId,
} from "./prFeedbackSettingsModel";
import { isFactoryOnboardingComplete } from "./onboarding/onboardingStatus";
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

function prFeedbackSetupHref(
  organizationId: string,
  factoryKey: string,
  lineId: string | undefined,
  sourceId: PRFeedbackSourceId,
): string | undefined {
  if (!lineId) {
    return undefined;
  }
  return factoryPRFeedbackSetupPath(organizationId, factoryKey, lineId, prFeedbackSetupKindFromSourceId(sourceId));
}

export function LinesPage() {
  const { organizationId, factoryId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { lineId: routeLineId, orderNumber: routeOrderNumber } = useParams<{ lineId?: string; orderNumber?: string }>();
  const { search, state: locationState } = useLocation();
  const navigate = useNavigate();
  const showColumnAutomations = useFactoryPreviewFlag("columnAutomations");
  const canChooseAutomationView = useFactoryPreviewFlag("columnAutomationRows") && showColumnAutomations;
  const { view: columnAutomationView, setView: setColumnAutomationView } = useColumnAutomationViewPreference();
  const showAutomationRows = canChooseAutomationView && columnAutomationView === "names";
  const intakeOpen = isIntakeSearchOpen(search);
  const intakeId = intakeIdFromSearch(search);
  const intakeSettingsTab = intakeSettingsTabFromSearch(search);
  const automationViewCanvasId = columnAutomationViewCanvasIdFromSearch(search);
  const prFeedbackOpen = isPRFeedbackSearchOpen(search);
  const prFeedbackSettingsTab = prFeedbackSettingsTabFromSearch(search);
  const prFeedbackHandlerId = prFeedbackHandlerIdFromSearch(search);
  const { data: workOrders = [], isLoading: workOrdersLoading } = useFactoryWorkOrders(organizationId, factoryId);
  const { data: pullRequests = [] } = useFactoryPullRequests(organizationId, factoryId);
  const { data: factoryApps = [] } = useFactoryApps(organizationId, factoryId);
  const { data: me } = useMe(false);
  const prFeedbackHandlersQuery = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const prFeedbackHandlers = prFeedbackHandlersQuery.data ?? [];
  const listState = useWorkOrderListState(factoryId);
  const { data: factoryIntakes = [] } = useFactoryIntakes(organizationId, factoryId);
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const configuredIntakes = useMemo(() => intakeSourcesFromFactoryIntakes(factoryIntakes), [factoryIntakes]);
  const showAddIntakeControl = useFactoryPreviewFlag("addIntakeControl");
  const { has: hasExperimentalFeature } = useExperimentalFeature(organizationId);
  const canCreateWithAgent = hasExperimentalFeature(FEATURE_FACTORY_CREATE_WITH_AGENT);
  const canAddSentryIntake = hasExperimentalFeature(FEATURE_FACTORY_SENTRY_INTAKE);
  const canAddProductiveIntake = hasExperimentalFeature(FEATURE_FACTORY_PRODUCTIVE_INTAKE);
  const addIntakeTemplates = useMemo(() => {
    const allowedIds = new Set(["github-issues"]);
    if (canAddSentryIntake) {
      allowedIds.add("sentry-exceptions");
    }
    if (canAddProductiveIntake) {
      allowedIds.add("productive-tasks");
    }
    return ADD_INTAKE_TEMPLATES.filter((template) => allowedIds.has(template.id));
  }, [canAddSentryIntake, canAddProductiveIntake]);
  // The menu entry only pays off once a source beyond the default GitHub issues is available.
  const canAddIntakeFromMenu = canAddSentryIntake || canAddProductiveIntake;
  const [addIntakeOpen, setAddIntakeOpen] = useState(false);
  const [productiveIntakeSetupOpen, setProductiveIntakeSetupOpen] = useState(false);
  const [addPRFeedbackOpen, setAddPRFeedbackOpen] = useState(false);
  const appRepository = factory?.onboarding?.appRepository?.trim() ?? "";
  const githubIntegrationId = factory?.onboarding?.vcsIntegrationId?.trim() ?? "";
  const catalogParameters = appRepository ? { repository: appRepository } : undefined;
  useIntegrationResources(organizationId, githubIntegrationId, "status_check", catalogParameters, {
    enabled: addPRFeedbackOpen && Boolean(githubIntegrationId),
  });
  useIntegrationResources(organizationId, githubIntegrationId, "review_bot", catalogParameters, {
    enabled: addPRFeedbackOpen && Boolean(githubIntegrationId),
  });
  const [peekHint, setPeekHint] = useState<FactoriesWorkOrder | null>(null);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const {
    addressingFeedbackOrderIds,
    addressingFeedbackLabels,
    waitingOnChecksOrderIds,
    checksPassedOrderIds,
    fixesPausedOrderIds,
  } = usePRFeedbackWorkOrderAttention(pullRequests);

  const { headerKicker: hostedCreditHeaderKicker, banner: hostedCreditEmptyBanner } = useHostedCreditChrome(
    organizationId,
    factoryKey,
  );
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

  const intakePanel: BacklogIntakePanel | undefined = showColumnAutomations
    ? undefined
    : {
        sources: configuredIntakes,
        showAddIntake: showAddIntakeControl,
        onOpenSettings: (intake) =>
          navigate(factoryIntakePath(organizationId, factoryKey, selectedLine.id, intake.intakeId)),
        onAddIntake: () => setAddIntakeOpen(true),
      };

  const takenPRFeedbackSources = takenPRFeedbackSourceIds(prFeedbackHandlers);
  const canAddPRFeedback = canUpdate && hasAvailablePRFeedbackSource(takenPRFeedbackSources);
  const nextSteps = workspaceNextSteps({
    onboardingComplete: isFactoryOnboardingComplete(factory),
    canConfigure: canUpdate,
    takenPRFeedbackSources,
    prFeedbackHandlersReady: isWorkspaceNextStepsQueryReady(prFeedbackHandlersQuery),
  });
  const nextStepBanner = workspaceNextStepBanner(nextSteps);
  const nextStepDeferral = useWorkspaceNextStepDeferral(factoryId);
  const nextStepsCollapsed = isWorkspaceNextStepDeferred(nextStepBanner, nextStepDeferral.deferredStepId);

  useEffect(() => {
    if (shouldForgetDeferredWorkspaceNextStep(nextStepDeferral.deferredStepId, takenPRFeedbackSources)) {
      nextStepDeferral.forget();
    }
  }, [nextStepDeferral.deferredStepId, nextStepDeferral.forget, takenPRFeedbackSources]);

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
    if (!isPRFeedbackSetupAvailable(source.id) || takenPRFeedbackSources.includes(source.id)) {
      return;
    }
    const href = prFeedbackSetupHref(organizationId, factoryKey, selectedLine.id, source.id);
    if (!href) {
      return;
    }
    setAddPRFeedbackOpen(false);
    navigate(href);
  };

  const createIntakeFromTemplate = (template: AddIntakeTemplate) => {
    setAddIntakeOpen(false);
    if (template.id === "productive-tasks") {
      setProductiveIntakeSetupOpen(true);
      return;
    }
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
          repository={factory?.onboarding?.backlogRepository}
          vcsIntegrationId={factory?.onboarding?.vcsIntegrationId}
          initialTab={isIntakeSettingsTab(intakeSettingsTab) ? intakeSettingsTab : "general"}
          onClose={() => navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id))}
        />
      ) : null}
      <AddIntakePicker
        open={addIntakeOpen}
        onClose={() => setAddIntakeOpen(false)}
        onSelect={createIntakeFromTemplate}
        templates={addIntakeTemplates}
      />
      <ProductiveIntakeSetupDialog
        open={productiveIntakeSetupOpen}
        organizationId={organizationId}
        factoryId={factoryId}
        onClose={() => setProductiveIntakeSetupOpen(false)}
      />
      <AddPRFeedbackPicker
        open={addPRFeedbackOpen}
        onClose={() => setAddPRFeedbackOpen(false)}
        onSelect={createPRFeedbackFromSource}
        takenSourceIds={takenPRFeedbackSources}
      />
      {automationViewCanvasId ? (
        <ColumnAutomationViewHost
          organizationId={organizationId}
          factoryKey={factoryKey}
          lineId={selectedLine.id}
          canvasId={automationViewCanvasId}
          title={factoryApps.find((app) => app.id === automationViewCanvasId)?.name?.trim() || "Automation"}
          onClose={() => navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id))}
        />
      ) : null}
      {prFeedbackOpen ? (
        <PRFeedbackSettingsHost
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          githubIntegrationId={githubIntegrationId}
          repository={appRepository}
          lineId={selectedLine.id}
          canUpdate={canUpdate}
          handlerId={prFeedbackHandlerId}
          initialTab={isPRFeedbackSettingsTab(prFeedbackSettingsTab) ? prFeedbackSettingsTab : "general"}
          onClose={() => navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id))}
        />
      ) : null}
      <div className={factoryKanbanPageClassName}>
        <div className="shrink-0">
          <LineDetailHeader
            organizationId={organizationId}
            factoryId={factoryId}
            line={selectedLine}
            workOrders={workOrders}
            factory={factory}
            state={listState}
            canUpdate={canUpdate}
            hostedCreditHeaderKicker={hostedCreditHeaderKicker}
            nextStepsRestore={
              nextStepsCollapsed && nextStepBanner ? (
                <WorkspaceNextStepsHeaderBadge
                  progress={workspaceNextStepsProgressCopy(nextStepBanner.doneCount, nextStepBanner.totalCount)}
                  title={nextStepBanner.badgeLabel}
                  onOpen={nextStepDeferral.restore}
                />
              ) : undefined
            }
            hostedCreditEmptyBanner={hostedCreditEmptyBanner}
            automationView={canChooseAutomationView ? columnAutomationView : undefined}
            onAutomationViewChange={canChooseAutomationView ? setColumnAutomationView : undefined}
          />
          <NextStepsPanel
            steps={nextSteps}
            collapsed={nextStepsCollapsed}
            onContinue={(step) =>
              runWorkspaceNextStepAction(step.action, {
                openPRFeedbackSetup: (sourceId) => {
                  const href = prFeedbackSetupHref(organizationId, factoryKey, selectedLine.id, sourceId);
                  if (href) {
                    navigate(href);
                  }
                },
              })
            }
            onDefer={() => {
              if (nextStepBanner) {
                nextStepDeferral.defer(nextStepBanner.activeStep.id);
              }
            }}
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
            canCreateWithAgent={canCreateWithAgent}
            canUpdate={canUpdate}
            onCreateWorkOrder={openCreateWorkOrder}
            intakePanel={intakePanel}
            onAddIntake={
              showColumnAutomations ? undefined : canAddIntakeFromMenu ? () => setAddIntakeOpen(true) : undefined
            }
            verifyListeners={showColumnAutomations ? [] : verifyListeners}
            onAddPRFeedback={
              showColumnAutomations ? undefined : canAddPRFeedback ? () => setAddPRFeedbackOpen(true) : undefined
            }
            factoryIntakes={factoryIntakes}
            prFeedbackHandlers={prFeedbackHandlers}
            showColumnAutomations={showColumnAutomations}
            showAutomationRows={showAutomationRows}
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
              pullRequests,
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
  line,
  workOrders,
  factory,
  state,
  canUpdate,
  hostedCreditHeaderKicker,
  nextStepsRestore,
  hostedCreditEmptyBanner,
  automationView,
  onAutomationViewChange,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  workOrders: FactoriesWorkOrder[];
  factory: FactoriesFactory | null;
  state: WorkOrderListState;
  canUpdate: boolean;
  hostedCreditHeaderKicker?: ReactNode;
  nextStepsRestore?: ReactNode;
  hostedCreditEmptyBanner?: ReactNode;
  automationView?: ColumnAutomationView;
  onAutomationViewChange?: (view: ColumnAutomationView) => void;
}) {
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const searchRef = useWorkOrdersHeaderShortcuts(state);
  const entries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [factory, workOrders]);
  const assigneeOptions = buildAssigneeFilterOptions(entries);
  const title = humanizeLineName(line.name);

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
        nextStepsRestore || hostedCreditHeaderKicker ? (
          <>
            {hostedCreditHeaderKicker}
            {nextStepsRestore}
          </>
        ) : undefined
      }
      actions={
        <>
          <ScopePills
            value={state.scope}
            onChange={state.setScope}
            options={WORK_ORDER_SCOPES}
            testIdPrefix="work-orders-scope"
          />
          <FilterMenu state={state} assigneeOptions={assigneeOptions} />
          <SearchField
            inputRef={searchRef}
            open={state.searchOpen}
            value={state.search}
            onOpen={state.openSearch}
            onChange={state.setSearch}
            onClose={state.closeSearch}
          />
          {automationView && onAutomationViewChange ? (
            <LineBoardViewMenu view={automationView} onViewChange={onAutomationViewChange} />
          ) : null}
        </>
      }
      belowRow={
        hostedCreditEmptyBanner || state.filterCount > 0 ? (
          <>
            {hostedCreditEmptyBanner}
            {state.filterCount > 0 ? <FilterChips state={state} assigneeOptions={assigneeOptions} /> : null}
          </>
        ) : undefined
      }
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
  canCreateWithAgent,
  canUpdate,
  onCreateWorkOrder,
  intakePanel,
  onAddIntake,
  verifyListeners,
  onAddPRFeedback,
  factoryIntakes,
  prFeedbackHandlers,
  showColumnAutomations,
  showAutomationRows,
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
  canCreateWithAgent: boolean;
  canUpdate: boolean;
  onCreateWorkOrder: () => void;
  intakePanel?: BacklogIntakePanel;
  onAddIntake?: () => void;
  verifyListeners: LaneListener[];
  onAddPRFeedback?: () => void;
  factoryIntakes: FactoriesFactoryIntake[];
  prFeedbackHandlers: FactoriesFactoryPrFeedbackHandler[];
  showColumnAutomations: boolean;
  showAutomationRows: boolean;
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
  const { factory } = useFactoriesLayout();
  const agentSession = useCreateWithAgentSession(workspacePlanningRepository(factory), organizationId, factoryId);
  const navigate = useNavigate();
  const [overlay, setOverlay] = useState<{
    disabledIds: string[];
    removedIds: string[];
  }>({ disabledIds: [], removedIds: [] });

  const automationsFor = useCallback(
    (key: ColumnKey, columnTitle: string) => {
      const base = buildColumnAutomations(key, {
        columnTitle,
        columns: board,
        intakes: factoryIntakes,
        prFeedbackHandlers,
        apps,
        workOrders,
      });
      return applyColumnAutomationsOverlay(base, undefined, overlay.disabledIds, overlay.removedIds);
    },
    [apps, board, factoryIntakes, overlay, prFeedbackHandlers, workOrders],
  );

  const handleRowAction = (automation: ColumnAutomation, action: ColumnAutomationRowAction) => {
    if (action === "settings") {
      const href = columnAutomationOpenPath(automation, { organizationId, factoryKey, lineId: line.id });
      if (href) {
        navigate(href);
      }
      return;
    }
    if (action === "disable") {
      setOverlay((current) => ({ ...current, disabledIds: [...current.disabledIds, automation.id] }));
      return;
    }
    if (action === "enable") {
      setOverlay((current) => ({
        ...current,
        disabledIds: current.disabledIds.filter((id) => id !== automation.id),
      }));
      return;
    }
    if (action === "remove") {
      setOverlay((current) => ({ ...current, removedIds: [...current.removedIds, automation.id] }));
    }
  };

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
          backlogOrders={backlogOrders}
          verifyOrders={verifyOrders}
          doneOrders={doneOrders}
          columns={board}
          canCreateWorkOrder={canCreateWorkOrder}
          canRename={canUpdate}
          onCreateWorkOrder={onCreateWorkOrder}
          onCreateWithAgent={canCreateWithAgent ? agentSession.start : undefined}
          intakePanel={intakePanel}
          onAddIntake={onAddIntake}
          verifyListeners={verifyListeners}
          onAddPRFeedback={onAddPRFeedback}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          analyzingOrderIds={backlogAnalysis.analyzingOrderIds}
          showColumnAutomations={showColumnAutomations}
          showAutomationRows={showAutomationRows}
          automationsFor={automationsFor}
          onAutomationRowAction={handleRowAction}
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
          isAnalyzing={backlogAnalysis.analyzingOrderIds.has(peekOrderId)}
          canRefine={canCreateWithAgent && peekOrder.state !== "STATE_DRAFT"}
          onClose={onClosePeek}
          onRefine={() => {
            const id = peekOrder.id?.trim();
            const title = peekOrder.title?.trim();
            if (!id || !title) {
              return;
            }
            agentSession.start({
              id,
              title,
              description: peekOrder.description ?? "",
            });
          }}
        />
      ) : null}
      {canCreateWithAgent ? (
        <LineCreateWithAgentDialog
          factoryKey={factoryKey}
          factoryId={factoryId}
          organizationId={organizationId}
          session={agentSession}
        />
      ) : null}
    </div>
  );
}

function LineCreateWithAgentDialog({
  factoryKey,
  factoryId,
  organizationId,
  session,
}: {
  factoryKey: string;
  factoryId: string;
  organizationId: string;
  session: ReturnType<typeof useCreateWithAgentSession>;
}) {
  const view = usePlanningSessionLiveRun(organizationId, session.view);
  return (
    <CreateWithAgentDialog
      open={session.open}
      workspaceName={factoryKey}
      organizationId={organizationId}
      factoryId={factoryId}
      view={view}
      onComposerChange={session.onComposerChange}
      onSend={session.onSend}
      onSubmitSurvey={session.onSubmitSurvey}
      onDraftTitleChange={session.onDraftTitleChange}
      onCreateDraft={session.onCreateDraft}
      onSkipDraft={session.onSkipDraft}
      onSelectCreated={session.onSelectCreated}
      onRefineCreated={session.onRefineCreated}
      onRequestClose={session.onRequestClose}
      onCancelEnd={session.onCancelEnd}
      onConfirmEnd={session.onConfirmEnd}
      onSelectModel={session.onSelectModel}
    />
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
  isAnalyzing,
  canRefine,
  onClose,
  onRefine,
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
  onDispatch: (orderId: string, input: { lineName: string; model?: string }) => Promise<void>;
  analysisRuns: BacklogAnalysisRun[];
  isAnalyzing: boolean;
  canRefine: boolean;
  onClose: () => void;
  onRefine: () => void;
}) {
  const { data: peekChecks = [] } = useWorkOrderChecks(organizationId, factoryId, peekOrderId);
  const { data: peekPullRequests = [] } = useFactoryPullRequests(organizationId, factoryId, {
    workOrderIds: [peekOrderId],
  });
  const { data: peekHandlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const prFeedbackRuns = useWorkOrderPRFeedbackLog(peekPullRequests, peekHandlers);
  const closer = useSplitRunFooterCloser(organizationId, factoryId, peekOrder);
  const { resolveUser } = useOrgUserLookup(organizationId);
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
        lineName: resolvedLineName,
        demoArtifacts: false,
        prFeedbackRuns,
        analysisRuns,
        isAnalyzing,
        stoppedBy: closer.actor,
        closer,
        resolveUser,
      })}
      canDispatch={canDispatch && Boolean(resolvedLineName)}
      canUpdate={canUpdate}
      canRefine={canRefine}
      isDispatching={isDispatching}
      onDispatch={
        resolvedLineName ? (model) => onDispatch(peekOrderId, { lineName: resolvedLineName, model }) : undefined
      }
      onClose={onClose}
      onRefine={onRefine}
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
    if (key === "implementation" || key === "risk" || key === "closure") {
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
  backlogOrders,
  verifyOrders,
  doneOrders,
  columns,
  canCreateWorkOrder,
  canRename,
  onCreateWorkOrder,
  onCreateWithAgent,
  intakePanel,
  onAddIntake,
  verifyListeners,
  onAddPRFeedback,
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
  showColumnAutomations,
  showAutomationRows,
  automationsFor,
  onAutomationRowAction,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  backlogOrders: FactoriesWorkOrder[];
  verifyOrders: FactoriesWorkOrder[];
  doneOrders: FactoriesWorkOrder[];
  columns: LinePhaseColumn[];
  canCreateWorkOrder: boolean;
  canRename: boolean;
  onCreateWorkOrder: () => void;
  onCreateWithAgent?: () => void;
  intakePanel?: BacklogIntakePanel;
  onAddIntake?: () => void;
  verifyListeners: LaneListener[];
  onAddPRFeedback?: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  analyzingOrderIds: ReadonlySet<string>;
  showColumnAutomations: boolean;
  showAutomationRows: boolean;
  automationsFor: (key: ColumnKey, columnTitle: string) => ColumnAutomation[];
  onAutomationRowAction: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
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

  // The line query is the source of truth for persisted colors. Resync when
  // it changes, but skip while a color save is in flight so a stale refetch
  // cannot wipe the optimistic lane color. Do not depend on isPending here:
  // when a save finishes, isPending flips before the factory query refetch
  // lands and would briefly restore the previous color.
  useEffect(() => {
    if (updateLine.isPending) {
      return;
    }
    const next = normalizeColumnColors(line.columnColors);
    if (serializeColumnColors(next) === serializeColumnColors(columnColorsRef.current)) {
      return;
    }
    columnColorsRef.current = next;
    setColumnColors(next);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- isPending must not trigger resync; see comment above
  }, [line.columnColors]);

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
        const updatedLine = await updateLine.mutateAsync({
          lineId,
          columnColors: serializeColumnColors(nextColors),
        });
        const persisted = normalizeColumnColors(updatedLine.columnColors);
        columnColorsRef.current = persisted;
        setColumnColors(persisted);
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

  const backlogTitle = columnTitles.backlog ?? "Backlog";
  const verifyTitle = columnTitles.verify ?? "Verify";
  const doneTitle = columnTitles.done ?? "Done";
  const phaseTitle = (column: LinePhaseColumn) => columnTitles[`phase-${column.stepIndex}`] ?? column.stepName;
  const backlogAutomations = showColumnAutomations ? automationsFor("backlog", backlogTitle) : undefined;
  const phaseAutomations = columns.map((column) =>
    showColumnAutomations ? automationsFor(`phase-${column.stepIndex}`, phaseTitle(column)) : undefined,
  );
  const verifyAutomations = showColumnAutomations ? automationsFor("verify", verifyTitle) : undefined;
  const doneAutomations = showColumnAutomations ? automationsFor("done", doneTitle) : undefined;
  const automationRowCount = showAutomationRows
    ? columnAutomationHeaderRowCount(
        [backlogAutomations, ...phaseAutomations, verifyAutomations, doneAutomations].map((list) => list ?? []),
      )
    : undefined;

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
          title={backlogTitle}
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
          onCreateWithAgent={onCreateWithAgent}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          analyzingOrderIds={analyzingOrderIds}
          intakePanel={intakePanel}
          onAddIntake={onAddIntake}
          automations={backlogAutomations}
          automationRowCount={automationRowCount}
          onAutomationRowAction={onAutomationRowAction}
        />
      </div>
      {columns.map((column, index) => {
        const columnKey: ColumnKey = `phase-${column.stepIndex}`;
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
              title={phaseTitle(column)}
              parallelism={parallelismByStep[column.stepIndex] ?? column.maxParallelism}
              onSaveParallelism={(value) => void saveParallelism(column.stepIndex, value)}
              colorId={columnColors[columnKey] ?? null}
              onColorChange={(colorId) => void setColumnColor(columnKey, colorId)}
              canRename={canRename}
              onRename={(title) => setColumnTitle(columnKey, title)}
              workOrderCardContext={workOrderCardContext}
              onOpenWorkOrder={onOpenWorkOrder}
              automations={phaseAutomations[index]}
              automationRowCount={automationRowCount}
              onAutomationRowAction={onAutomationRowAction}
            />
          </div>
        );
      })}
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <span className="absolute top-[21px] left-0 z-[1] h-px w-3 -translate-x-full bg-border" aria-hidden />
        <VerifyColumn
          orders={verifyOrders}
          title={verifyTitle}
          listeners={verifyListeners}
          onAdd={onAddPRFeedback}
          colorId={columnColors.verify ?? null}
          onColorChange={(colorId) => void setColumnColor("verify", colorId)}
          canRename={canRename}
          onRename={(title) => setColumnTitle("verify", title)}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          automations={verifyAutomations}
          automationRowCount={automationRowCount}
          onAutomationRowAction={onAutomationRowAction}
        />
      </div>
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <span className="absolute top-[21px] left-0 z-[1] h-px w-3 -translate-x-full bg-border" aria-hidden />
        <DoneColumn
          orders={doneOrders}
          title={doneTitle}
          colorId={columnColors.done ?? null}
          onColorChange={(colorId) => void setColumnColor("done", colorId)}
          canRename={canRename}
          onRename={(title) => setColumnTitle("done", title)}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          automations={doneAutomations}
          automationRowCount={automationRowCount}
          onAutomationRowAction={onAutomationRowAction}
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
  automations,
  automationRowCount,
  onAutomationRowAction,
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
  automations?: ColumnAutomation[];
  automationRowCount?: number;
  onAutomationRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
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
          {automationRowCount ? null : (
            <ColumnAutomationsHeaderSlot
              title={title}
              automations={automations}
              onRowAction={onAutomationRowAction}
              testId="lines-verify-automations"
            />
          )}
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
      subheader={columnAutomationRowsSubheader({
        title,
        automations,
        rowCount: automationRowCount,
        onRowAction: onAutomationRowAction,
        testId: "lines-verify-automation-rows",
      })}
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
  automations,
  automationRowCount,
  onAutomationRowAction,
}: {
  orders: FactoriesWorkOrder[];
  title: string;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canRename: boolean;
  onRename: (title: string) => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  automations?: ColumnAutomation[];
  automationRowCount?: number;
  onAutomationRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
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
        <div className="flex shrink-0 items-center gap-0.5">
          {automationRowCount ? null : (
            <ColumnAutomationsHeaderSlot
              title={title}
              automations={automations}
              onRowAction={onAutomationRowAction}
              testId="lines-done-automations"
            />
          )}
          <ColumnLaneMenu title={title} testId="lines-done-menu" colorId={colorId} onColorChange={onColorChange} />
        </div>
      }
      subheader={columnAutomationRowsSubheader({
        title,
        automations,
        rowCount: automationRowCount,
        onRowAction: onAutomationRowAction,
        testId: "lines-done-automation-rows",
      })}
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
  automations,
  automationRowCount,
  onAutomationRowAction,
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
  automations?: ColumnAutomation[];
  automationRowCount?: number;
  onAutomationRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
}) {
  const scrollRef = useRef<HTMLUListElement>(null);
  const [visibleCount, setVisibleCount] = useState(LINE_PHASE_RUNS_PAGE_SIZE);
  const [parallelismOpen, setParallelismOpen] = useState(false);
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
        subheader={columnAutomationRowsSubheader({
          title,
          automations,
          rowCount: automationRowCount,
          onRowAction: onAutomationRowAction,
          testId: `lines-phase-${column.stepIndex}-automation-rows`,
        })}
        actions={
          <div className="flex shrink-0 items-center gap-0.5">
            {automationRowCount ? null : (
              <ColumnAutomationsHeaderSlot
                title={title}
                automations={automations}
                onRowAction={onAutomationRowAction}
                testId={`lines-phase-${column.stepIndex}-automations`}
              />
            )}
            <ColumnLaneMenu
              title={title}
              testId={`lines-phase-menu-${column.stepIndex}`}
              onSetParallelism={configureHref ? () => setParallelismOpen(true) : undefined}
              parallelism={parallelism}
              colorId={colorId}
              onColorChange={onColorChange}
            />
          </div>
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
