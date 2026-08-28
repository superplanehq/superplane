import { createContext, useContext } from "react";

/**
 * Surfaces that are built, but hidden from the app until the flow is ready.
 * The Storybook harness turns them on so design review keeps the UI.
 */
export interface FactoryPreviewFlags {
  /** Show Add intake in the Intake drawer. */
  addIntakeControl: boolean;
}

export const FactoryPreviewFlagsContext = createContext<FactoryPreviewFlags | null>(null);

/** Returns false in the app, where no preview provider is present. */
export function useFactoryPreviewFlag(flag: keyof FactoryPreviewFlags): boolean {
  return useContext(FactoryPreviewFlagsContext)?.[flag] ?? false;
}
