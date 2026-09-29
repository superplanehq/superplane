import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import type { CreatedTaskHref } from "./CreatedTaskCard";
import { draftStartModelPayload } from "./draftStartModel";
import { draftStartThinkingPayload } from "@/lib/thinkingLevel";
import type { SplitRunFixture } from "./splitRunMocks";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";

export function popupWorkOrderUrl(organizationId?: string, factoryKey?: string, orderNumber?: string, lineId?: string) {
  if (!organizationId || !factoryKey || !orderNumber) {
    return window.location.href;
  }
  return window.location.origin + workOrderDetailPath(organizationId, factoryKey, orderNumber, lineId);
}

/**
 * Permalink builder for tasks the agent splits off the open draft. It keeps
 * the board line so the new task opens on the same board. Returns nothing
 * when the popup has no factory context or the task has no number yet.
 */
export function createdTaskHref(organizationId?: string, factoryKey?: string, lineId?: string): CreatedTaskHref {
  return (task) => {
    if (!organizationId || !factoryKey || !task.number) {
      return undefined;
    }
    return workOrderDetailPath(organizationId, factoryKey, task.number, lineId);
  };
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
  onDispatch: ((model?: string, thinkingLevel?: string) => Promise<void>) | undefined,
  openAutomations: () => void,
  selectedModel: string,
  selectedThinking: string,
) {
  if (kind !== "draft") {
    return undefined;
  }
  return async () => {
    await onDispatch?.(draftStartModelPayload(selectedModel), draftStartThinkingPayload(selectedThinking));
    openAutomations();
  };
}
