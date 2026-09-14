import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { draftStartModelPayload } from "./draftStartModel";
import type { SplitRunFixture } from "./splitRunMocks";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";

export function popupWorkOrderUrl(organizationId?: string, factoryKey?: string, orderNumber?: string, lineId?: string) {
  if (!organizationId || !factoryKey || !orderNumber) {
    return window.location.href;
  }
  return window.location.origin + workOrderDetailPath(organizationId, factoryKey, orderNumber, lineId);
}

export function footerMutationHandlers(
  canUpdate: boolean,
  footerActions: SplitRunFooterActions,
  fixture: SplitRunFixture,
  onDismiss?: () => void,
) {
  if (!canUpdate) {
    return {};
  }
  return {
    onArchive: async () => {
      const archived = await footerActions.handleArchive();
      if (archived) {
        onDismiss?.();
      }
    },
    onReject: () => void footerActions.handleReject(),
    onBackToDraft: () => footerActions.handleBackToDraft(),
    onStop: (choice: Parameters<typeof footerActions.handleStop>[0]) =>
      void footerActions.handleStop(choice, {
        ...fixture.footer,
        lineName: fixture.lineName,
        stepIndex: fixture.currentStepIndex,
      }),
  };
}

export function draftStartAction(
  kind: SplitRunFixture["footer"]["kind"],
  onDispatch: ((model?: string) => Promise<void>) | undefined,
  openAutomations: () => void,
  selectedModel: string,
) {
  if (kind !== "draft") {
    return undefined;
  }
  return async () => {
    await onDispatch?.(draftStartModelPayload(selectedModel));
    openAutomations();
  };
}

export function returnToBacklogAction(
  onBackToDraft: (() => void | Promise<boolean | void>) | undefined,
  openDescription: () => void,
) {
  if (!onBackToDraft) {
    return undefined;
  }
  return async () => {
    const returned = await onBackToDraft();
    if (returned === false) {
      return;
    }
    openDescription();
  };
}
