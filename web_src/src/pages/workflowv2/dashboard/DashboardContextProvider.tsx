import { useMemo, type ReactNode } from "react";

import { DashboardContext, type DashboardContextValue } from "./DashboardContext";

export interface DashboardContextProviderProps extends DashboardContextValue {
  children: ReactNode;
}

export function DashboardContextProvider({
  children,
  canvasId,
  organizationId,
  nodes,
  nodeStatuses,
  canRunNodes,
  onTriggerNode,
  onOpenNode,
}: DashboardContextProviderProps) {
  const value = useMemo<DashboardContextValue>(
    () => ({ canvasId, organizationId, nodes, nodeStatuses, canRunNodes, onTriggerNode, onOpenNode }),
    [canvasId, organizationId, nodes, nodeStatuses, canRunNodes, onTriggerNode, onOpenNode],
  );

  return <DashboardContext.Provider value={value}>{children}</DashboardContext.Provider>;
}
