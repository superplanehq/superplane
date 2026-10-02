import { useState, type ReactNode } from "react";

import type { FactoriesFactory, FactoriesFactoryPullRequest } from "@/api-client";
import { useFactory } from "@/hooks/useFactoryData";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";

import { analysisFirstResultDelivered, hasAnalysisPlan, hasAnalysisScore } from "../../lib/analysisOutcome";
import { formatWorkOrderIdentifier } from "../../lib/workspaceKey";
import { LiveHeaderSpendProvider } from "./liveHeaderSpendContext";
import { planningHeaderSpendActive } from "./planningHeaderSpend";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";
import { THINKING_LEVEL_MEDIUM } from "@/lib/thinkingLevel";
import { SplitRunPopupTabs } from "./SplitRunPopupTabs";
import { SplitRunReview } from "./SplitRunReview";
import {
  classicSplitRunFooter,
  composerCreditVerdict,
  creditBillingHref,
  isTaskResultFooter,
  SPLIT_RUN_ANALYZING_NOTE,
} from "./splitRunFooter";
import { refinePopupShowsAutomations, SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";
import { isPullRequestReviewFooter } from "./splitRunPullRequestReview";
import { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import { useWorkOrderFullPagePreference } from "./workOrderFullPagePreference";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";
import { draftStartAction, footerMutationHandlers } from "./workOrderPopupActions";
import { AnalysisPopupHeader, LoadingWorkOrderPopup } from "./workOrderPopupHeader";
import { workOrderPopupMode } from "./workOrderPopupMode";
import { factoryPlanningEnabled, factoryShowsClarity, factoryShowsConfidence } from "../planningSettingsModel";
import { PopupShell } from "../work-order-popup-redesign/popupShared";

export type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";

/**
 * Work-order popup from a line-board card. Description and a phase log.
 * The automation canvas lives on the full run page, not here.
 */
export function WorkOrderSplitRunPopup(props: WorkOrderSplitRunPopupProps) {
  const {
    organizationId,
    factoryId,
    factoryKey,
    orderNumber,
    orderId,
    fixture,
    fixed = false,
    onClose,
    canUpdate = true,
  } = props;
  const titlePrefix = popupTitlePrefix(fixture, factoryKey, orderNumber);
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
    return <LoadingWorkOrderPopup title={fixture.title} titlePrefix={titlePrefix} fixed={fixed} onClose={onClose} />;
  }
  return <AnalysisWorkOrderPopup {...props} titlePrefix={titlePrefix} analysis={analysis} popupData={popupData} />;
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
  analysis,
  popupData,
  titlePrefix,
}: WorkOrderSplitRunPopupProps & {
  analysis: ReturnType<typeof useAnalysisPlanningSession>;
  popupData: ReturnType<typeof useSplitRunPopupData>;
  titlePrefix?: string;
}) {
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const dismissCurrentPopup = useCurrentPopupDismiss(orderId, onClose);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, dismissCurrentPopup);
  const edits = useAnalysisPopupEdits({ organizationId, factoryId, orderId, canUpdate, fixture, popupData });
  const { fullPage, toggleFullPage } = useWorkOrderFullPagePreference();
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const [draftThinking, setDraftThinking] = useState(THINKING_LEVEL_MEDIUM);
  const factory = useFactory(organizationId ?? "", factoryId ?? "").data;
  const draftStart = draftStartAction(
    fixture.footer.kind,
    onDispatch,
    draftModel,
    factoryPlanningEnabled(factory) ? draftThinking : undefined,
  );
  const showPullRequestReview = isPullRequestReviewFooter(fixture.footer);
  const showSidebarNote = showPullRequestReview || isTaskResultFooter(fixture.footer);
  const { sourceOnly, viewFixture } = analysisPopupView(fixture, factory);
  const unified = refinePopupShowsAutomations({ footerKind: viewFixture.footer.kind, sourceOnly });
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
  const panelReview = unified ? analysisPopupReview({ ...reviewArgs, compact: "stacked" }) : undefined;
  const stripAnalysis = draftChrome.stripAnalysis;
  const descriptionReview = !unified && !showSidebarNote ? review : undefined;

  return (
    <PopupShell
      testId="work-order-split-run"
      fixed={fixed}
      fullPage={fullPage}
      className={analysisPopupClassName(fullPage, unified, sourceOnly)}
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
          canUpdate={canUpdate}
          footerActions={footerActions}
          resultFooter={descriptionReview}
          sidebarNote={!unified && showSidebarNote ? review : undefined}
          analysis={stripAnalysis}
          sourceOnly={sourceOnly}
          sessionLookupError={analysis.queryError?.message}
          panelReview={panelReview}
          header={(views) => (
            <AnalysisPopupHeader
              edits={edits}
              fixture={fixture}
              titlePrefix={titlePrefix}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              lineId={lineId}
              onClose={onClose}
              fullPage={fullPage}
              toggleFullPage={toggleFullPage}
              onArchive={mutations.onArchive}
              footerBusy={footerActions.busy}
              reviewActions={reviewActions}
              showOwnerRow={!unified}
              planningSpend={draftPlanningHeaderSpend(fixture, analysis.view)}
              views={views}
            />
          )}
        />
      </LiveHeaderSpendProvider>
    </PopupShell>
  );
}

function popupTitlePrefix(
  fixture: WorkOrderSplitRunPopupProps["fixture"],
  factoryKey?: string,
  orderNumber?: string,
): string | undefined {
  const stored = fixture.identifier?.trim();
  if (stored) {
    return stored;
  }
  return formatWorkOrderIdentifier(factoryKey, orderNumber) || undefined;
}

function draftPlanningHeaderSpend(
  fixture: WorkOrderSplitRunPopupProps["fixture"],
  view: ReturnType<typeof useAnalysisPlanningSession>["view"],
) {
  if (!planningHeaderSpendActive(fixture.footer.kind, view)) {
    return undefined;
  }
  return {
    view,
    savedTokens: fixture.savedTokens ?? 0,
    savedCostCents: fixture.savedCostCents ?? 0,
  };
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
  compact: boolean | "stacked";
  actionsOnly?: boolean;
  ctaOnly?: boolean;
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
      ctaOnly={args.ctaOnly}
      confirmUnclearStart={args.confirmUnclearStart}
      modelSelect={args.modelSelect}
    />
  );
}

type AnalysisDraftChromeArgs = {
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
};

function analysisDraftChrome(args: AnalysisDraftChromeArgs) {
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
    stripAnalysis: draftStripAnalysis(args, modelSelects.strip),
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

/** The console is wide. A Planning draft and a source-only draft keep the refine widths. */
function analysisPopupClassName(fullPage: boolean, unified: boolean, sourceOnly: boolean) {
  if (fullPage || sourceOnly) {
    return undefined;
  }
  if (unified) {
    return "h-[min(52rem,calc(100vh-5rem))] w-[min(72rem,calc(100vw-5rem))]";
  }
  return SPLIT_RUN_POPUP_DIALOG_CLASSNAME;
}

/**
 * The refine strip only shows for a draft.
 */
function analysisPopupView(fixture: WorkOrderSplitRunPopupProps["fixture"], factory: FactoriesFactory | undefined) {
  const sourceOnly = fixture.footer.kind === "draft" && !factoryPlanningEnabled(factory);
  return {
    sourceOnly,
    viewFixture: sourceOnly ? { ...fixture, footer: classicSplitRunFooter(fixture.footer) } : fixture,
  };
}

function draftStripAnalysis(args: AnalysisDraftChromeArgs, modelSelect: ReactNode | undefined) {
  if (args.fixture.footer.kind !== "draft") {
    return undefined;
  }
  const billingHref = creditBillingHref(args.organizationId, args.factoryKey);
  return {
    ...args.analysis,
    modelSelect,
    showClarity: factoryShowsClarity(args.factory),
    showConfidence: factoryShowsConfidence(args.factory),
    creditVerdict: composerCreditVerdict(args.fixture.footer.note, billingHref),
  };
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
