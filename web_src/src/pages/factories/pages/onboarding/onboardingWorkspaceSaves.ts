import { createContext, useContext } from "react";

export const OnboardingWorkspaceSavesContext = createContext(true);

export function useOnboardingWorkspaceSavesAllowed(): boolean {
  return useContext(OnboardingWorkspaceSavesContext);
}

export const WORKSPACE_STILL_OPENING = "This workspace is still opening. Save again when it is ready.";

export function guardWorkspaceSave<Args extends unknown[], Result>(
  savesAllowed: () => boolean,
  save: (...args: Args) => Promise<Result>,
): (...args: Args) => Promise<Result> {
  return (...args) => {
    if (!savesAllowed()) {
      return Promise.reject(new Error(WORKSPACE_STILL_OPENING));
    }
    return save(...args);
  };
}
