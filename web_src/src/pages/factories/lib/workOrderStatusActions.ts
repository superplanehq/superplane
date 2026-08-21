import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";

import type { WorkOrderDisplayStatus } from "./workOrderProgress";

export type WorkOrderStatusActionKind = "complete" | "reject" | "reject-draft" | "back-to-draft" | "reopen";

export interface WorkOrderStatusAction {
  kind: WorkOrderStatusActionKind;
  label: string;
  disabled: boolean;
  separatorBefore?: boolean;
}

export interface WorkOrderStatusActionInput {
  displayStatus: WorkOrderDisplayStatus;
  isOpen: boolean;
  isDispatchable: boolean;
  isClosed: boolean;
  canClose: boolean;
  canManage: boolean;
  isClosing: boolean;
  isUpdatingStatus: boolean;
}

/** Lifecycle items for the work-order overflow menu and the status-note menu. */
export function buildWorkOrderStatusActions(input: WorkOrderStatusActionInput): WorkOrderStatusAction[] {
  const actions: WorkOrderStatusAction[] = [];
  const isDraft = input.isDispatchable && !input.isOpen && !input.isClosed;
  const closeDisabled = !input.canClose || input.isClosing;
  const manageDisabled = !input.canManage || input.isUpdatingStatus;

  if (input.isOpen) {
    actions.push(
      { kind: "complete", label: "Complete", disabled: closeDisabled },
      { kind: "reject", label: "Reject", disabled: closeDisabled },
    );
  }

  if (input.isOpen && input.displayStatus !== "running") {
    actions.push({
      kind: "back-to-draft",
      label: "Back to draft",
      disabled: manageDisabled,
      separatorBefore: true,
    });
  }

  if (isDraft) {
    actions.push({ kind: "reject-draft", label: "Reject", disabled: closeDisabled });
  }

  if (input.isClosed) {
    actions.push({ kind: "reopen", label: "Reopen", disabled: manageDisabled });
  }

  return actions;
}

export function applyWorkOrderStatusAction(
  kind: WorkOrderStatusActionKind,
  handlers: {
    onClose: (result: FactoriesWorkOrderResult) => void;
    onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  },
): void {
  switch (kind) {
    case "complete":
      handlers.onClose("RESULT_COMPLETED");
      return;
    case "reject":
    case "reject-draft":
      handlers.onClose("RESULT_REJECTED");
      return;
    case "back-to-draft":
      void handlers.onStatusChange("STATE_DRAFT");
      return;
    case "reopen":
      void handlers.onStatusChange("STATE_OPEN");
  }
}
