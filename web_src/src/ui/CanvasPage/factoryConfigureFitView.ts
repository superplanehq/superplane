import { FACTORY_LAYOUT_ANIMATION_DURATION_MS } from "./nodePositionAnimation";

/** Wait for configure layout + rank expansion to finish, then fit the graph. */
export const FACTORY_CONFIGURE_FIT_SETTLE_MS = FACTORY_LAYOUT_ANIMATION_DURATION_MS + 50;

/** One-shot fit when Factory Configure edit becomes ready. Not for later draft edits. */
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
