import { useState, type ReactNode } from "react";
import { Loader2 } from "lucide-react";

import type { FactoriesFactoryPullRequest } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";

import { analysisFirstResultDelivered, hasAnalysisPlan, hasAnalysisScore } from "../../lib/analysisOutcome";
import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { ClassicWorkOrderPopup } from "./ClassicWorkOrderPopup";
import type { CreatedTaskHref } from "./CreatedTaskCard";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";
import { PopupHeaderActions } from "./PopupHeaderActions";
import { SplitRunPopupTabs } from "./SplitRunPopupTabs";
import { SplitRunReview } from "./SplitRunReview";
import { isTaskResultFooter, SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { defaultSplitRunPopupTab, SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";
import { isPullRequestReviewFooter } from "./splitRunPullRequestReview";
import { useImplementationRunnerModel } from "./useImplementationRunnerModel";
import { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import { useWorkOrderFullPagePreference } from "./workOrderFullPagePreference";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";
import { createdTaskHref, draftStartAction, footerMutationHandlers, popupWorkOrderUrl } from "./workOrderPopupActions";
import { workOrderPopupMode } from "./workOrderPopupMode";

export type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";

/**
 * Work-order popup from a line-board card. Description and a phase log.
 * The automation canvas lives on the full run page, not here.
 */
export function WorkOrderSplitRunPopup(props: WorkOrderSplitRunPopupProps) {
  const { organizationId, factoryId, orderId, fixture, fixed = false, onClose, canUpdate = true } = props;
  const refinementFeature = useExperimentalFeature(organizationId);
  const refinementEnabled = refinementFeature.has(FEATURE_FACTORY_CREATE_WITH_AGENT);
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
    pollForSession: refinementEnabled && isAnalyzing,
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
  if (mode === "analysis") {
    return <AnalysisWorkOrderPopup {...props} analysis={analysis} popupData={popupData} />;
  }
  return <ClassicWorkOrderPopup {...props} popupData={popupData} sessionLookupError={analysis.queryError} />;
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
  analysis,
  popupData,
}: WorkOrderSplitRunPopupProps & {
  analysis: ReturnType<typeof useAnalysisPlanningSession>;
  popupData: ReturnType<typeof useSplitRunPopupData>;
}) {
  const modelLabel = useImplementationRunnerModel(organizationId, fixture.phases);
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const dismissCurrentPopup = useCurrentPopupDismiss(orderId, onClose);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, dismissCurrentPopup);
  const edits = useAnalysisPopupEdits({ organizationId, factoryId, orderId, canUpdate, fixture, popupData });
  const initialTab = defaultSplitRunPopupTab(fixture);
  const [tab, setTab] = useState(initialTab);
  const { fullPage, toggleFullPage } = useWorkOrderFullPagePreference();
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const draftStart = draftStartAction(fixture.footer.kind, onDispatch, () => setTab("log"), draftModel);
  const showPullRequestReview = isPullRequestReviewFooter(fixture.footer);
  const showSidebarNote = showPullRequestReview || isTaskResultFooter(fixture.footer);
  const modelSelects = draftModelSelects({
    organizationId,
    factoryId,
    fixture,
    value: draftModel,
    onChange: setDraftModel,
    disabled: isDispatching || !canDispatch,
  });
  const reviewArgs = {
    fixture,
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
    compact: showSidebarNote || fixture.footer.kind === "draft",
    modelSelect: modelSelects.footer,
  };
  const review = analysisPopupReview(reviewArgs);
  const reviewActions = showPullRequestReview ? analysisPopupReview({ ...reviewArgs, actionsOnly: true }) : undefined;
  const taskHref = createdTaskHref(organizationId, factoryKey, lineId);
  const stripAnalysis = draftStripAnalysis(fixture.footer.kind, analysis, modelSelects.strip, taskHref);

  return (
    <PopupShell
      testId="work-order-split-run"
      fixed={fixed}
      fullPage={fullPage}
      className={fullPage ? undefined : SPLIT_RUN_POPUP_DIALOG_CLASSNAME}
      onDismiss={onClose}
    >
      <SplitRunPopupTabs
        fixture={fixture}
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
        resultFooter={!showSidebarNote && tab === "description" ? review : undefined}
        sidebarNote={showSidebarNote ? review : undefined}
        analysis={stripAnalysis}
        header={(views) => (
          <PopupHeader
            title={edits.title}
            onClose={onClose}
            canEditTitle={edits.canEdit}
            titleBusy={edits.titleBusy}
            onTitleSave={(next) => void edits.saveTitle(next)}
            expanded={fullPage}
            onToggleExpanded={toggleFullPage}
            actions={
              <PopupHeaderActions
                copyUrl={popupWorkOrderUrl(organizationId, factoryKey, orderNumber, lineId)}
                onArchive={fixture.footer.kind === "draft" ? mutations.onArchive : undefined}
                archiveBusy={footerActions.busy}
                taskActions={reviewActions}
              />
            }
            accessory={views}
          >
            <OwnerTimeCostRow
              fixture={{ ...fixture, owner: edits.owner }}
              modelLabel={modelLabel}
              assigneeIds={edits.assigneeIds}
            />
          </PopupHeader>
        )}
      />
      {!showSidebarNote && tab !== "description" ? review : null}
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
      confirmUnclearStart
      modelSelect={args.modelSelect}
    />
  );
}

/**
 * The refine strip only shows for a draft. It gets the ghost model select
 * and a permalink builder for tasks the agent splits off this one.
 */
function draftStripAnalysis(
  footerKind: WorkOrderSplitRunPopupProps["fixture"]["footer"]["kind"],
  analysis: ReturnType<typeof useAnalysisPlanningSession>,
  modelSelect: ReactNode | undefined,
  taskHref: CreatedTaskHref,
) {
  if (footerKind !== "draft") {
    return undefined;
  }
  return { ...analysis, modelSelect, taskHref };
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
  value: string;
  onChange: (value: string) => void;
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
