import { Loader2 } from "lucide-react";
import type { ReactNode } from "react";

import { PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { LiveOwnerTimeCostRow } from "./LiveOwnerTimeCostRow";
import { PlanningHeaderSpendCollector } from "./PlanningHeaderSpendCollector";
import { PopupHeaderActions } from "./PopupHeaderActions";
import { showsArchive } from "./splitRunFooter";
import type { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";
import { popupWorkOrderUrl } from "./workOrderPopupActions";

export function LoadingWorkOrderPopup({
  title,
  titlePrefix,
  fixed,
  onClose,
}: {
  title: string;
  titlePrefix?: string;
  fixed: boolean;
  onClose?: () => void;
}) {
  return (
    <PopupShell testId="work-order-split-run-loading" fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={title} titlePrefix={titlePrefix} onClose={onClose} />
      <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground" role="status">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        Loading task…
      </div>
    </PopupShell>
  );
}

type AnalysisHeaderEdits = {
  title: string;
  canEdit: boolean;
  titleBusy: boolean;
  saveTitle: (next: string) => void;
  owner: WorkOrderSplitRunPopupProps["fixture"]["owner"];
  assigneeIds: string[];
};

export function AnalysisPopupHeader({
  edits,
  fixture,
  titlePrefix,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
  onClose,
  fullPage,
  toggleFullPage,
  onArchive,
  footerBusy,
  reviewActions,
  showOwnerRow,
  planningSpend,
  views,
}: {
  edits: AnalysisHeaderEdits;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  titlePrefix?: string;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  onClose: WorkOrderSplitRunPopupProps["onClose"];
  fullPage: boolean;
  toggleFullPage: () => void;
  onArchive?: () => void;
  footerBusy: boolean;
  reviewActions: ReactNode;
  /** The unified view moves owner and spend into the summary panel. */
  showOwnerRow: boolean;
  planningSpend?: {
    view: ReturnType<typeof useAnalysisPlanningSession>["view"];
    savedTokens: number;
    savedCostCents: number;
  };
  views: ReactNode;
}) {
  return (
    <PopupHeader
      title={edits.title}
      titlePrefix={titlePrefix}
      onClose={onClose}
      canEditTitle={edits.canEdit}
      titleBusy={edits.titleBusy}
      onTitleSave={(next) => void edits.saveTitle(next)}
      expanded={fullPage}
      onToggleExpanded={toggleFullPage}
      actions={
        <PopupHeaderActions
          copyUrl={popupWorkOrderUrl(organizationId, factoryKey, orderNumber, lineId)}
          onArchive={showsArchive(fixture.footer) ? onArchive : undefined}
          archiveBusy={footerBusy}
          taskActions={reviewActions}
        />
      }
      accessory={views}
    >
      {planningSpend ? (
        <PlanningHeaderSpendCollector
          organizationId={organizationId}
          view={planningSpend.view}
          saved={{ tokens: planningSpend.savedTokens, cents: planningSpend.savedCostCents }}
        />
      ) : null}
      {showOwnerRow ? (
        <LiveOwnerTimeCostRow
          fixture={{ ...fixture, owner: edits.owner }}
          organizationId={organizationId}
          canEditOwner={edits.canEdit}
          assigneeIds={edits.assigneeIds}
          ownerBusy={edits.ownerBusy}
          onOwnerSave={edits.saveOwner}
          usageByModel={fixture.usageByModel}
          usageByMachineType={fixture.usageByMachineType}
        />
      ) : null}
    </PopupHeader>
  );
}
