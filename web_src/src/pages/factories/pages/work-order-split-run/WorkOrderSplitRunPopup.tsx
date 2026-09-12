import { useState } from "react";

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CREATE_WITH_AGENT, FEATURE_FACTORY_DRAFT_START_MODEL } from "@/lib/experimentalFeatures";

import { CopyLinkButton } from "../../CopyLinkButton";
import { analysisFirstResultDelivered } from "../../lib/analysisOutcome";
import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { ClassicWorkOrderPopup } from "./ClassicWorkOrderPopup";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";
import { SplitRunPopupTabs } from "./SplitRunPopupTabs";
import { SplitRunReview } from "./SplitRunReview";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { defaultSplitRunPopupTab, SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";
import { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";
import {
  draftStartAction,
  footerMutationHandlers,
  popupWorkOrderUrl,
  returnToBacklogAction,
} from "./workOrderPopupActions";
import { workOrderPopupMode } from "./workOrderPopupMode";

export type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";

/**
 * Work-order popup from a line-board card. Description and a phase log.
 * The automation canvas lives on the full run page, not here.
 */
export function WorkOrderSplitRunPopup(props: WorkOrderSplitRunPopupProps) {
  const { organizationId, factoryId, orderId, fixture, canUpdate = true } = props;
  const refinementEnabled = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_CREATE_WITH_AGENT);
  const isDraft = fixture.footer.kind === "draft";
  const isAnalyzing = fixture.footer.note?.headline === SPLIT_RUN_ANALYZING_NOTE.headline;
  const canLookupSession = isDraft && Boolean(organizationId && factoryId && orderId);
  const hasLookupIdentity = Boolean(factoryId && orderId);
  const popupData = useSplitRunPopupData({ organizationId, factoryId, orderId, fixture });
  const analysis = useAnalysisPlanningSession({
    organizationId,
    factoryId,
    workOrderId: orderId,
    enabled: canLookupSession,
    pollForSession: refinementEnabled && isAnalyzing,
    canUpdate,
    analysisDelivered: analysisFirstResultDelivered({
      checks: fixture.checks,
      artifacts: popupData.artifacts,
    }),
  });
  const mode = workOrderPopupMode({
    hasPlanningSession: Boolean(analysis.session),
    refinementEnabled,
    analysisActive: isAnalyzing,
    hasLookupIdentity,
  });

  if (mode === "analysis") {
    return <AnalysisWorkOrderPopup {...props} analysis={analysis} popupData={popupData} />;
  }
  return <ClassicWorkOrderPopup {...props} popupData={popupData} sessionLookupError={analysis.queryError} />;
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
  const canPickDraftStartModel = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_DRAFT_START_MODEL);
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const dismissCurrentPopup = useCurrentPopupDismiss(orderId, onClose);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, dismissCurrentPopup);
  const edits = useSplitRunWorkOrderEdits({
    organizationId,
    factoryId,
    orderId,
    canUpdate,
    title: fixture.title,
    description: popupData.sourceDescription,
    owner: fixture.owner,
    assigneeIds: fixture.assigneeIds ?? [],
    footerKind: fixture.footer.kind,
  });
  const initialTab = defaultSplitRunPopupTab(fixture);
  const [tab, setTab] = useState(initialTab);
  const [fullPage, setFullPage] = useState(false);
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const draftStart = draftStartAction(fixture.footer.kind, onDispatch, () => setTab("log"), draftModel);
  const backToDraft = returnToBacklogAction(mutations.onBackToDraft, () => setTab("description"));
  const review = (
    <SplitRunReview
      footer={fixture.footer}
      organizationId={organizationId}
      factoryKey={factoryKey}
      orderNumber={orderNumber}
      canAct={canUpdate}
      onStart={draftStart}
      onArchive={mutations.onArchive}
      onReject={mutations.onReject}
      onBackToDraft={backToDraft}
      onStop={mutations.onStop}
      startBusy={isDispatching}
      actionBusy={footerActions.busy}
      startDisabled={!canDispatch}
      modelSelect={
        fixture.footer.kind === "draft" &&
        canPickDraftStartModel &&
        fixture.footer.actions.some((action) => action.kind === "start") ? (
          <DraftStartModelSelect
            organizationId={organizationId}
            factoryId={factoryId}
            lineName={fixture.lineName}
            value={draftModel}
            onChange={setDraftModel}
            disabled={isDispatching}
          />
        ) : undefined
      }
    />
  );

  return (
    <PopupShell
      testId="work-order-split-run"
      fixed={fixed}
      fullPage={fullPage}
      className={fullPage ? undefined : SPLIT_RUN_POPUP_DIALOG_CLASSNAME}
      onDismiss={onClose}
    >
      <PopupHeader
        title={edits.title}
        onClose={onClose}
        canEditTitle={edits.canEdit}
        titleBusy={edits.titleBusy}
        onTitleSave={(next) => void edits.saveTitle(next)}
        expanded={fullPage}
        onToggleExpanded={() => setFullPage((current) => !current)}
        actions={
          <CopyLinkButton
            url={popupWorkOrderUrl(organizationId, factoryKey, orderNumber, lineId)}
            className="flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10"
            iconClassName="h-4 w-4"
            testId="popup-work-order-copy-link-button"
          />
        }
      >
        <OwnerTimeCostRow fixture={{ ...fixture, owner: edits.owner }} assigneeIds={edits.assigneeIds} />
      </PopupHeader>
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
        resultFooter={tab === "description" ? review : undefined}
        analysis={fixture.footer.kind === "draft" ? analysis : undefined}
      />
      {tab !== "description" ? review : null}
    </PopupShell>
  );
}
