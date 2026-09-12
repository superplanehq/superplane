import { useMemo, useState } from "react";

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_DRAFT_START_MODEL } from "@/lib/experimentalFeatures";

import { CopyLinkButton } from "../../CopyLinkButton";
import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";
import { SplitRunPopupTabs } from "./SplitRunPopupTabs";
import { SplitRunReview } from "./SplitRunReview";
import { classicSplitRunFooter } from "./splitRunFooter";
import { defaultSplitRunPopupTab } from "./splitRunPopupModel";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import type { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunPopup";
import {
  draftStartAction,
  footerMutationHandlers,
  popupWorkOrderUrl,
  returnToBacklogAction,
} from "./workOrderPopupActions";

type ClassicWorkOrderPopupProps = WorkOrderSplitRunPopupProps & {
  popupData: ReturnType<typeof useSplitRunPopupData>;
  sessionLookupError: Error | null;
};

export function ClassicWorkOrderPopup({
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
  popupData,
  sessionLookupError,
}: ClassicWorkOrderPopupProps) {
  const classicFixture = useMemo(() => ({ ...fixture, footer: classicSplitRunFooter(fixture.footer) }), [fixture]);
  const canPickDraftStartModel = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_DRAFT_START_MODEL);
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const dismissCurrentPopup = useCurrentPopupDismiss(orderId, onClose);
  const mutations = footerMutationHandlers(canUpdate, footerActions, classicFixture, dismissCurrentPopup);
  const edits = useSplitRunWorkOrderEdits({
    organizationId,
    factoryId,
    orderId,
    canUpdate,
    title: fixture.title,
    description: popupData.sourceDescription,
    owner: fixture.owner,
    assigneeIds: fixture.assigneeIds ?? [],
    footerKind: classicFixture.footer.kind,
  });
  const [tab, setTab] = useState(defaultSplitRunPopupTab(classicFixture));
  const [fullPage, setFullPage] = useState(false);
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const draftStart = draftStartAction(classicFixture.footer.kind, onDispatch, () => setTab("log"), draftModel);
  const backToDraft = returnToBacklogAction(mutations.onBackToDraft, () => setTab("description"));

  return (
    <PopupShell testId="work-order-split-run" fixed={fixed} fullPage={fullPage} onDismiss={onClose}>
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
        <OwnerTimeCostRow fixture={{ ...classicFixture, owner: edits.owner }} assigneeIds={edits.assigneeIds} />
      </PopupHeader>
      <SplitRunPopupTabs
        mode="classic"
        fixture={classicFixture}
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
        sessionLookupError={sessionLookupError?.message}
      />
      <SplitRunReview
        footer={classicFixture.footer}
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
          classicFixture.footer.kind === "draft" && canPickDraftStartModel ? (
            <DraftStartModelSelect
              organizationId={organizationId}
              factoryId={factoryId}
              lineName={classicFixture.lineName}
              value={draftModel}
              onChange={setDraftModel}
              disabled={isDispatching}
            />
          ) : undefined
        }
      />
    </PopupShell>
  );
}
