import { createContext, useContext } from "react";

/**
 * Surfaces that are built, but hidden from the app until the flow is ready.
 * The Storybook harness turns them on so design review keeps the UI.
 * Shipped flags default to on when no provider is present.
 */
export interface FactoryPreviewFlags {
  /** Show Add intake in the Intake drawer. */
  addIntakeControl: boolean;
  /**
   * Per-column Automations menu. On by default.
   * Set false to keep lane listener banners (legacy Storybook).
   */
  columnAutomations?: boolean;
}

export const FactoryPreviewFlagsContext = createContext<FactoryPreviewFlags | null>(null);

const SHIPPED_PREVIEW_FLAGS: Partial<FactoryPreviewFlags> = {
  columnAutomations: true,
};

/** Returns the shipped default when no preview provider is present. */
export function useFactoryPreviewFlag(flag: keyof FactoryPreviewFlags): boolean {
  return useContext(FactoryPreviewFlagsContext)?.[flag] ?? SHIPPED_PREVIEW_FLAGS[flag] ?? false;
}
