import { workOrderDetailPath } from "../../lib/factoryPagePaths";
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
  selectedThinking?: string,
) {
  if (kind !== "draft") {
    return undefined;
  }
  return async () => {
    await onDispatch?.(draftStartModelPayload(selectedModel), startThinkingPayload(selectedThinking));
    openAutomations();
  };
}

function startThinkingPayload(selectedThinking?: string) {
  if (selectedThinking === undefined) {
    return undefined;
  }
  return draftStartThinkingPayload(selectedThinking);
}

/**
 * Strips sp-file:// references from markdown and HTML.
 * Removes pasted-image markdown references and attachment links while preserving
 * surrounding text and ordinary HTTP/HTTPS links.
 */
export function stripFileReferences(description: string): string {
  let result = description;

  // Remove markdown image/link syntax with sp-file:// URLs: ![alt](sp-file://...) or [text](sp-file://...)
  result = result.replace(/(!?\[[^\]]*]\()sp-file:\/\/[a-f0-9-]+(\))/g, "");

  // Remove HTML img src with sp-file:// URLs: src="sp-file://..."
  result = result.replace(/(\bsrc\s*=\s*["'])sp-file:\/\/[a-f0-9-]+(["'])/gi, "$1$2");

  // Remove HTML href with sp-file:// URLs: href="sp-file://..."
  result = result.replace(/(\bhref\s*=\s*["'])sp-file:\/\/[a-f0-9-]+(["'])/gi, "$1$2");

  // Clean up empty markdown links/images left behind by the above replacements: ]() or ![]()
  result = result.replace(/(!?\[\s*\]\s*\(\s*\))/g, "");

  return result;
}
