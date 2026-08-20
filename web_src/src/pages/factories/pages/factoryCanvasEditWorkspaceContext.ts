import { createContext, useContext } from "react";

export const FactoryCanvasEditWorkspaceContext = createContext(false);

/** True only inside the Storybook factories harness. Live app stays on the current canvas chrome. */
export function useFactoryCanvasEditWorkspace() {
  return useContext(FactoryCanvasEditWorkspaceContext);
}
