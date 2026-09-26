import { useState, type ReactNode } from "react";
import { Loader2 } from "lucide-react";

import type { FactoriesFactory, FactoriesFactoryPullRequest } from "@/api-client";
import { useFactory } from "@/hooks/useFactoryData";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";

import { analysisFirstResultDelivered, hasAnalysisPlan, hasAnalysisScore } from "../../lib/analysisOutcome";
import { PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { LiveOwnerTimeCostRow } from "./LiveOwnerTimeCostRow";
import { LiveHeaderSpendProvider } from "./liveHeaderSpendContext";
import type { CreatedTaskHref } from "./CreatedTaskCard";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";
import { DRAFT_START_THINKING_AUTO } from "@/lib/thinkingLevel";
import { PopupHeaderActions } from "./PopupHeaderActions";
import { SplitRunPopupTabs } from "./SplitRunPopupTabs";
import { SplitRunReview } from "./SplitRunReview";
import { classicSplitRunFooter, isTaskResultFooter, SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { defaultSplitRunPopupTab, SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";
import { isPullRequestReviewFooter } from "./splitRunPullRequestReview";
import { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import { useWorkOrderFullPagePreference } from "./workOrderFullPagePreference";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";
import { createdTaskHref, draftStartAction, footerMutationHandlers, popupWorkOrderUrl } from "./workOrderPopupActions";
import { workOrderPopupMode } from "./workOrderPopupMode";
import { factoryPlanningEnabled, factoryShowsClarity, factoryShowsConfidence } from "../planningSettingsModel";

export type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";

/**
 * Work-order popup from a line-board card. Description and a phase log.
 * The automation canvas lives on the full run page, not here.
 */
export function WorkOrderSplitRunPopup(props: WorkOrderSplitRunPopupProps) {
  const { organizationId, factoryId, orderId, fixture, fixed = false, onClose, canUpdate = true } = props;
  const factoryQuery = useFactory(organizationId ?? "", factoryId ?? "");
  const refinementEnabled = factoryPlanningEnabled(factoryQuery.data);
  const refinementFeature = { isLoading: factoryQuery.isPending };
  const isAnalyzing = fixture.footer.note?.headline === SPLIT_RUN_ANALYZING_NOTE.headline;
  const canLookupSession = Boolean(organizationId && factoryId && orderId);
  const hasLookupIdentity = Boolean(factoryId && orderId);
  const popupData = useSplitRunPopupData({ organizationId, factoryId, orderId, fixture });
  const fileUpload = useWorkOrderFileUpload({
    organizationId: organizationId ?? "",
    factoryId: factoryId ?? "",
    orderId,
  });
  const analysis = useAnalysisPlanningSession({
    organizationId,
    factoryId,
    workOrderId: orderId,
    enabled: canLookupSession,
    canUpdate,
    isUploading: fileUpload.isUploading,
    uploadFiles: fileUpload.uploadFiles,
    analysisDelivered: analysisFirstResultDelivered({
      checks: fixture.checks,
      artifacts: popupData.artifacts,
    }),
  });
  const mode = workOrderPopupMode({
    hasPlanningSession: Boolean(analysis.session),
    hasAnalysisResult: hasAnalysisScore(fixture.checks) || hasAnalysisPlan(popupData.artifacts),
    refinementEnabled,
    refinementLoading: refinementFeature.isLoading,
    sessionLoading: analysis.isLoading,
    artifactsLoading: popupData.artifactsLoading,
    artifactsFailed: Boolean(popupData.artifactsError),
    analysisActive: isAnalyzing,
    hasLookupIdentity,
    isDraft: fixture.footer.kind === "draft",
  });

  if (mode === "loading") {
    return <LoadingWorkOrderPopup title={fixture.title} fixed={fixed} onClose={onClose} />;
  }
  return <AnalysisWorkOrderPopup {...props} analysis={analysis} popupData={popupData} />;
}

function LoadingWorkOrderPopup({ title, fixed, onClose }: { title: string; fixed: boolean; onClose?: () => void }) {
  return (
    <PopupShell testId="work-order-split-run-loading" fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={title} onClose={onClose} />
      <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground" role="status">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        Loading task…
      </div>
    </PopupShell>
  );
}

function AnalysisWorkOrderPopup({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  fixture,
  onClose,
  fixed = false,
  onDispatch,
  isDispatching = false,
  canDispatch = false,
  canUpdate = true,
  initialTab,
  analysis,
  popupData,
}: WorkOrderSplitRunPopupProps & {
  analysis: ReturnType<typeof useAnalysisPlanningSession>;
  popupData: ReturnType<typeof useSplitRunPopupData>;
}) {
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const dismissCurrentPopup = useCurrentPopupDismiss(orderId, onClose);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, dismissCurrentPopup);
  const edits = useAnalysisPopupEdits({ organizationId, factoryId, orderId, canUpdate, fixture, popupData });
  const [tab, setTab] = useState(() => initialTab ?? defaultSplitRunPopupTab(fixture));
  const { fullPage, toggleFullPage } = useWorkOrderFullPagePreference();
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const [draftThinking, setDraftThinking] = useState(DRAFT_START_THINKING_AUTO);
  const draftStart = draftStartAction(fixture.footer.kind, onDispatch, () => setTab("log"), draftModel, draftThinking);
  const showPullRequestReview = isPullRequestReviewFooter(fixture.footer);
  const showSidebarNote = showPullRequestReview || isTaskResultFooter(fixture.footer);
  const factory = useFactory(organizationId ?? "", factoryId ?? "").data;
  const { sourceOnly, viewFixture } = analysisPopupView(fixture, factory);
  const draftChrome = analysisDraftChrome({
    factory,
    organizationId,
    factoryId,
    factoryKey,
    lineId,
    fixture: viewFixture,
    analysis,
    draftModel,
    draftThinking,
    onDraftStartChange: ({ model, thinkingLevel }) => {
      setDraftModel(model);
      setDraftThinking(thinkingLevel);
    },
    disabled: isDispatching || !canDispatch,
  });
  const reviewArgs = analysisReviewArgs({
    viewFixture,
    organizationId,
    factoryId,
    factoryKey,
    orderId,
    orderNumber,
    pullRequests: popupData.pullRequests,
    canUpdate,
    draftStart,
    mutations,
    isDispatching,
    footerBusy: footerActions.busy,
    canDispatch,
    compact: analysisReviewCompact(showSidebarNote, viewFixture.footer.kind, sourceOnly),
    modelSelect: draftChrome.footerModelSelect,
    factory,
  });
  const review = analysisPopupReview(reviewArgs);
  const reviewActions = showPullRequestReview ? analysisPopupReview({ ...reviewArgs, actionsOnly: true }) : undefined;
  const stripAnalysis = draftChrome.stripAnalysis;
  const descriptionReview = showsDescriptionReview(sourceOnly, showSidebarNote, tab) ? review : undefined;

  return (
    <PopupShell
      testId="work-order-split-run"
      fixed={fixed}
      fullPage={fullPage}
      className={analysisPopupClassName(fullPage, sourceOnly)}
      onDismiss={onClose}
    >
      <LiveHeaderSpendProvider>
        <SplitRunPopupTabs
          fixture={viewFixture}
          edits={edits}
          popupData={popupData}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          orderId={orderId}
          orderNumber={orderNumber}
          lineId={lineId}
          tab={tab}
          onTabChange={setTab}
          canUpdate={canUpdate}
          footerActions={footerActions}
          resultFooter={descriptionReview}
          sidebarNote={showSidebarNote ? review : undefined}
          analysis={stripAnalysis}
          sourceOnly={sourceOnly}
          sessionLookupError={analysis.queryError?.message}
          header={analysisPopupHeader({
            edits,
            fixture,
            organizationId,
            factoryKey,
            orderNumber,
            lineId,
            onClose,
            fullPage,
            toggleFullPage,
            mutations,
            footerBusy: footerActions.busy,
            reviewActions,
          })}
        />
        {analysisShellReview(sourceOnly, showSidebarNote, tab, review)}
      </LiveHeaderSpendProvider>
    </PopupShell>
  );
}

/** Title, owner, and assignee edits for the popup header, fed from the fixture and loaded description. */
function useAnalysisPopupEdits(args: {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canUpdate: boolean;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  popupData: ReturnType<typeof useSplitRunPopupData>;
}) {
  const { fixture, popupData, ...ids } = args;
  return useSplitRunWorkOrderEdits({
    ...ids,
    title: fixture.title,
    description: popupData.sourceDescription,
    owner: fixture.owner,
    assigneeIds: fixture.assigneeIds ?? [],
    footerKind: fixture.footer.kind,
  });
}

function analysisPopupHeader(args: {
  edits: ReturnType<typeof useAnalysisPopupEdits>;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  onClose: WorkOrderSplitRunPopupProps["onClose"];
  fullPage: boolean;
  toggleFullPage: () => void;
  mutations: ReturnType<typeof footerMutationHandlers>;
  footerBusy: boolean;
  reviewActions: ReactNode;
}) {
  return (views: ReactNode) => (
    <PopupHeader
      title={args.edits.title}
      onClose={args.onClose}
      canEditTitle={args.edits.canEdit}
      titleBusy={args.edits.titleBusy}
      onTitleSave={(next) => void args.edits.saveTitle(next)}
      expanded={args.fullPage}
      onToggleExpanded={args.toggleFullPage}
      actions={
        <PopupHeaderActions
          copyUrl={popupWorkOrderUrl(args.organizationId, args.factoryKey, args.orderNumber, args.lineId)}
          onArchive={args.fixture.footer.kind === "draft" ? args.mutations.onArchive : undefined}
          archiveBusy={args.footerBusy}
          taskActions={args.reviewActions}
        />
      }
      accessory={views}
    >
      <LiveOwnerTimeCostRow
        fixture={{ ...args.fixture, owner: args.edits.owner }}
        assigneeIds={args.edits.assigneeIds}
        usageByModel={args.fixture.usageByModel}
        usageByMachineType={args.fixture.usageByMachineType}
      />
    </PopupHeader>
  );
}

function analysisReviewArgs(args: {
  viewFixture: WorkOrderSplitRunPopupProps["fixture"];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  pullRequests?: FactoriesFactoryPullRequest[];
  canUpdate: boolean;
  draftStart: ReturnType<typeof draftStartAction>;
  mutations: ReturnType<typeof footerMutationHandlers>;
  isDispatching: boolean;
  footerBusy: boolean;
  canDispatch: boolean;
  compact: boolean;
  modelSelect?: ReactNode;
  factory?: FactoriesFactory;
}) {
  return {
    fixture: args.viewFixture,
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    factoryKey: args.factoryKey,
    orderId: args.orderId,
    orderNumber: args.orderNumber,
    pullRequests: args.pullRequests,
    canUpdate: args.canUpdate,
    draftStart: args.draftStart,
    mutations: args.mutations,
    isDispatching: args.isDispatching,
    footerBusy: args.footerBusy,
    canDispatch: args.canDispatch,
    compact: args.compact,
    modelSelect: args.modelSelect,
    confirmUnclearStart: factoryPlanningEnabled(args.factory),
  };
}

function analysisPopupReview(args: {
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  pullRequests?: FactoriesFactoryPullRequest[];
  canUpdate: boolean;
  draftStart: ReturnType<typeof draftStartAction>;
  mutations: ReturnType<typeof footerMutationHandlers>;
  isDispatching: boolean;
  footerBusy: boolean;
  canDispatch: boolean;
  compact: boolean;
  actionsOnly?: boolean;
  modelSelect?: ReactNode;
  confirmUnclearStart?: boolean;
}) {
  return (
    <SplitRunReview
      footer={args.fixture.footer}
      organizationId={args.organizationId}
      factoryId={args.factoryId}
      factoryKey={args.factoryKey}
      orderId={args.orderId}
      orderNumber={args.orderNumber}
      pullRequests={args.pullRequests}
      canAct={args.canUpdate}
      onStart={args.draftStart}
      onArchive={args.mutations.onArchive}
      onReject={args.mutations.onReject}
      onStop={args.mutations.onStop}
      startBusy={args.isDispatching}
      actionBusy={args.footerBusy}
      startDisabled={!args.canDispatch}
      compact={args.compact}
      actionsOnly={args.actionsOnly}
      confirmUnclearStart={args.confirmUnclearStart}
      modelSelect={args.modelSelect}
    />
  );
}

function analysisDraftChrome(args: {
  factory?: FactoriesFactory;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  lineId?: string;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  analysis: ReturnType<typeof useAnalysisPlanningSession>;
  draftModel: string;
  draftThinking: string;
  onDraftStartChange: (next: { model: string; thinkingLevel: string }) => void;
  disabled: boolean;
}) {
  if (!factoryPlanningEnabled(args.factory)) {
    return { footerModelSelect: undefined, stripAnalysis: undefined };
  }
  const modelSelects = draftModelSelects({
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    fixture: args.fixture,
    model: args.draftModel,
    thinkingLevel: args.draftThinking,
    onChange: args.onDraftStartChange,
    disabled: args.disabled,
  });
  return {
    footerModelSelect: modelSelects.footer,
    stripAnalysis: draftStripAnalysis(
      args.fixture.footer.kind,
      args.analysis,
      modelSelects.strip,
      createdTaskHref(args.organizationId, args.factoryKey, args.lineId),
      {
        showClarity: factoryShowsClarity(args.factory),
        showConfidence: factoryShowsConfidence(args.factory),
      },
    ),
  };
}

/** A draft with Planning on uses the compact review; a source-only draft keeps the classic one. */
function analysisReviewCompact(
  showSidebarNote: boolean,
  footerKind: WorkOrderSplitRunPopupProps["fixture"]["footer"]["kind"],
  sourceOnly: boolean,
) {
  return showSidebarNote || (footerKind === "draft" && !sourceOnly);
}

function showsDescriptionReview(sourceOnly: boolean, showSidebarNote: boolean, tab: string) {
  return !sourceOnly && !showSidebarNote && tab === "description";
}

function analysisShellReview(sourceOnly: boolean, showSidebarNote: boolean, tab: string, review: ReactNode) {
  if (sourceOnly || (!showSidebarNote && tab !== "description")) {
    return review;
  }
  return null;
}

/** The split-run dialog width applies only to the Planning layout in a fixed popup. */
function analysisPopupClassName(fullPage: boolean, sourceOnly: boolean) {
  return fullPage || sourceOnly ? undefined : SPLIT_RUN_POPUP_DIALOG_CLASSNAME;
}

/**
 * The refine strip only shows for a draft. It gets the ghost model select
 * and a permalink builder for tasks the agent splits off this one.
 */
function analysisPopupView(fixture: WorkOrderSplitRunPopupProps["fixture"], factory: FactoriesFactory | undefined) {
  const sourceOnly = fixture.footer.kind === "draft" && !factoryPlanningEnabled(factory);
  return {
    sourceOnly,
    viewFixture: sourceOnly ? { ...fixture, footer: classicSplitRunFooter(fixture.footer) } : fixture,
  };
}

function draftStripAnalysis(
  footerKind: WorkOrderSplitRunPopupProps["fixture"]["footer"]["kind"],
  analysis: ReturnType<typeof useAnalysisPlanningSession>,
  modelSelect: ReactNode | undefined,
  taskHref: CreatedTaskHref,
  scores: { showClarity: boolean; showConfidence: boolean },
) {
  if (footerKind !== "draft") {
    return undefined;
  }
  return { ...analysis, modelSelect, taskHref, ...scores };
}

/**
 * One model select per surface: `labeled` for the footer capsule under the
 * plan, `ghost` for the refine strip settings row. Only a draft with Start
 * gets one.
 */
function draftModelSelects(args: {
  organizationId?: string;
  factoryId?: string;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  model: string;
  thinkingLevel: string;
  onChange: (next: { model: string; thinkingLevel: string }) => void;
  disabled: boolean;
}): { footer?: ReactNode; strip?: ReactNode } {
  const { fixture, ...select } = args;
  if (fixture.footer.kind !== "draft" || !fixture.footer.actions.some((action) => action.kind === "start")) {
    return {};
  }
  return {
    footer: <DraftStartModelSelect {...select} lineName={fixture.lineName} appearance="labeled" />,
    strip: <DraftStartModelSelect {...select} lineName={fixture.lineName} appearance="ghost" />,
  };
}
