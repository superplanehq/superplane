import { draftStartModelPayload } from "./draftStartModel";
import type { SplitRunFixture } from "./splitRunMocks";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";

export function footerMutationHandlers(
  canUpdate: boolean,
  footerActions: SplitRunFooterActions,
  fixture: SplitRunFixture,
) {
  if (!canUpdate) {
    return {};
  }
  return {
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
