import { FACTORY_CONFIGURE_FIT_VIEW_OPTIONS } from "./canvasFitOptions";

/** Wait one frame for node measure after Configure layout snaps, then frame at 100% zoom. */
export const FACTORY_CONFIGURE_FIT_SETTLE_MS = 50;

/** Frame the Configure enter fit. Center a deep-linked node when one is present. */
export function factoryConfigureEnterFitViewOptions(focusNode?: { id: string } | null) {
  if (focusNode) {
    return {
      ...FACTORY_CONFIGURE_FIT_VIEW_OPTIONS,
      nodes: [focusNode],
      duration: 0,
    };
  }

  return {
    ...FACTORY_CONFIGURE_FIT_VIEW_OPTIONS,
    duration: 0,
  };
}

/** One-shot frame at 100% zoom when Factory Configure edit becomes ready. Not for later draft edits. */
export function shouldFitFactoryConfigureEnter(input: {
  factoryConfigure: boolean;
  isEditing: boolean;
  hasReactFlowInitialized: boolean;
  hasFittedThisVisit: boolean;
  nodeCount: number;
  layoutReady: boolean;
}): boolean {
  return (
    input.factoryConfigure &&
    input.isEditing &&
    input.hasReactFlowInitialized &&
    input.layoutReady &&
    !input.hasFittedThisVisit &&
    input.nodeCount > 0
  );
}
