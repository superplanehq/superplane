import { FACTORY_CONFIGURE_FIT_VIEW_OPTIONS } from "./canvasFitOptions";
import { FACTORY_LAYOUT_ANIMATION_DURATION_MS } from "./nodePositionAnimation";

/** Wait for configure layout + rank expansion to finish, then frame at 100% zoom. */
export const FACTORY_CONFIGURE_FIT_SETTLE_MS = FACTORY_LAYOUT_ANIMATION_DURATION_MS + 50;

/** Frame the Configure enter fit. Center a deep-linked node when one is present. */
export function factoryConfigureEnterFitViewOptions(focusNode?: { id: string } | null) {
  if (focusNode) {
    return {
      ...FACTORY_CONFIGURE_FIT_VIEW_OPTIONS,
      nodes: [focusNode],
      duration: 500,
    };
  }

  return {
    ...FACTORY_CONFIGURE_FIT_VIEW_OPTIONS,
    duration: 500,
  };
}

/** One-shot frame at 100% zoom when Factory Configure edit becomes ready. Not for later draft edits. */
export function shouldFitFactoryConfigureEnter(input: {
  factoryConfigure: boolean;
  isEditing: boolean;
  hasReactFlowInitialized: boolean;
  hasFittedThisVisit: boolean;
  nodeCount: number;
}): boolean {
  return (
    input.factoryConfigure &&
    input.isEditing &&
    input.hasReactFlowInitialized &&
    !input.hasFittedThisVisit &&
    input.nodeCount > 0
  );
}
