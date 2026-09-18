import { useMemo, useState } from "react";

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";

import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";
import { PopupHeaderActions } from "./PopupHeaderActions";
import { SplitRunPopupTabs } from "./SplitRunPopupTabs";
import { SplitRunReview } from "./SplitRunReview";
import { classicSplitRunFooter, isTaskResultFooter } from "./splitRunFooter";
import { defaultSplitRunPopupTab } from "./splitRunPopupModel";
import { isPullRequestReviewFooter } from "./splitRunPullRequestReview";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import { useWorkOrderFullPagePreference } from "./workOrderFullPagePreference";
import type { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunPopup";
import { draftStartAction, footerMutationHandlers, popupWorkOrderUrl } from "./workOrderPopupActions";

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
  const canPickDraftStartModel = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_CREATE_WITH_AGENT);
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
  const { fullPage, toggleFullPage } = useWorkOrderFullPagePreference();
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const draftStart = draftStartAction(classicFixture.footer.kind, onDispatch, () => setTab("log"), draftModel);
  const showPullRequestReview = isPullRequestReviewFooter(classicFixture.footer);
  const showSidebarNote = showPullRequestReview || isTaskResultFooter(classicFixture.footer);
  const reviewProps = {
    footer: classicFixture.footer,
    organizationId,
    factoryKey,
    orderNumber,
    canAct: canUpdate,
    onStart: draftStart,
    onArchive: mutations.onArchive,
    onReject: mutations.onReject,
    onStop: mutations.onStop,
    startBusy: isDispatching,
    actionBusy: footerActions.busy,
    startDisabled: !canDispatch,
    modelSelect:
      classicFixture.footer.kind === "draft" && canPickDraftStartModel ? (
        <DraftStartModelSelect
          organizationId={organizationId}
          factoryId={factoryId}
          lineName={classicFixture.lineName}
          value={draftModel}
          onChange={setDraftModel}
          disabled={isDispatching || !canDispatch}
        />
      ) : undefined,
  };
  const review = <SplitRunReview {...reviewProps} compact={showSidebarNote} />;
  const reviewActions = showPullRequestReview ? <SplitRunReview {...reviewProps} actionsOnly /> : undefined;

  return (
    <PopupShell testId="work-order-split-run" fixed={fixed} fullPage={fullPage} onDismiss={onClose}>
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
        sidebarNote={showSidebarNote ? review : undefined}
        sessionLookupError={sessionLookupError?.message}
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
                onArchive={classicFixture.footer.kind === "draft" ? mutations.onArchive : undefined}
                archiveBusy={footerActions.busy}
                taskActions={reviewActions}
              />
            }
            accessory={views}
          >
            <OwnerTimeCostRow
              fixture={{ ...classicFixture, owner: edits.owner }}
              assigneeIds={edits.assigneeIds}
              usageByModel={classicFixture.usageByModel}
              usageByMachineType={classicFixture.usageByMachineType}
            />
          </PopupHeader>
        )}
      />
      {showPullRequestReview ? null : review}
    </PopupShell>
  );
}
