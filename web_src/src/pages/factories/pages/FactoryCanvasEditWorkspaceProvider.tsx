import type { ReactNode } from "react";

import { FactoryCanvasEditWorkspaceContext } from "./factoryCanvasEditWorkspaceContext";

export function FactoryCanvasEditWorkspaceProvider({ children }: { children: ReactNode }) {
  return (
    <FactoryCanvasEditWorkspaceContext.Provider value={true}>{children}</FactoryCanvasEditWorkspaceContext.Provider>
  );
}
