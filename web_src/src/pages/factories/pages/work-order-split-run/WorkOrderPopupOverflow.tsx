import { CopyLinkButton } from "../../CopyLinkButton";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { getWorkOrderDetailDerived } from "../../lib/workOrderProgress";
import { buildWorkOrderStatusActions } from "../../lib/workOrderStatusActions";
import { WorkOrderOverflowMenu } from "../../WorkOrderOverflowMenu";
import type { FactoriesWorkOrder } from "@/api-client";
import type { SplitRunFixture } from "./splitRunMocks";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";

function popupWorkOrderUrl(organizationId?: string, factoryKey?: string, orderNumber?: string, lineId?: string) {
  if (!organizationId || !factoryKey || !orderNumber) {
    return window.location.href;
  }
  return window.location.origin + workOrderDetailPath(organizationId, factoryKey, orderNumber, lineId);
}

export function WorkOrderPopupOverflow({
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
  order,
  fixture,
  canCreate,
  canUpdate,
  isDuplicating,
  onDuplicate,
  footerActions,
}: {
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  order?: FactoriesWorkOrder;
  fixture: SplitRunFixture;
  canCreate: boolean;
  canUpdate: boolean;
  isDuplicating: boolean;
  onDuplicate: () => void;
  footerActions: SplitRunFooterActions;
}) {
  const derived = getWorkOrderDetailDerived(order);
  const statusActions = buildWorkOrderStatusActions({
    displayStatus: derived.displayStatus ?? "waiting",
    isOpen: derived.isOpen,
    isDispatchable: derived.isDispatchable,
    isClosed: derived.isClosed,
    canClose: canUpdate,
    canManage: canUpdate,
    isClosing: footerActions.busy,
    isUpdatingStatus: footerActions.busy,
  });
  const stopFooter = {
    ...fixture.footer,
    lineName: fixture.lineName,
    stepIndex: fixture.currentStepIndex,
  };

  return (
    <>
      <CopyLinkButton
        url={popupWorkOrderUrl(organizationId, factoryKey, orderNumber, lineId)}
        className="flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10"
        iconClassName="h-4 w-4"
        testId="popup-work-order-copy-link-button"
      />
      <WorkOrderOverflowMenu
        canCreate={canCreate}
        isDuplicating={isDuplicating}
        onDuplicate={onDuplicate}
        actions={statusActions}
        onClose={(result) => {
          if (result === "RESULT_REJECTED") {
            void footerActions.handleReject();
            return;
          }
          void footerActions.handleStop("completed", stopFooter);
        }}
        onStatusChange={async (state) => {
          if (state === "STATE_DRAFT") {
            await footerActions.handleBackToDraft();
            return;
          }
          if (state === "STATE_OPEN") {
            await footerActions.handleStop("reopen", stopFooter);
          }
        }}
        disabled={footerActions.busy}
        triggerClassName="flex h-6 w-6 items-center justify-center rounded-full text-muted-foreground hover:bg-slate-950/5 hover:text-foreground dark:hover:bg-white/10"
      />
    </>
  );
}
