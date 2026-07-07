import { useMemo, type ReactNode } from "react";

import { ConsoleContext, type ConsoleContextValue } from "./ConsoleContext";

export interface ConsoleContextProviderProps extends ConsoleContextValue {
  children: ReactNode;
}

export function ConsoleContextProvider({
  children,
  canvasId,
  organizationId,
  nodes,
  nodeStatuses,
  canRunNodes,
  manualRunTriggers,
  onTriggerNode,
  onOpenNode,
}: ConsoleContextProviderProps) {
  const value = useMemo<ConsoleContextValue>(
    () => ({
      canvasId,
      organizationId,
      nodes,
      nodeStatuses,
      canRunNodes,
      manualRunTriggers,
      onTriggerNode,
      onOpenNode,
    }),
    [canvasId, organizationId, nodes, nodeStatuses, canRunNodes, manualRunTriggers, onTriggerNode, onOpenNode],
  );

  return <ConsoleContext.Provider value={value}>{children}</ConsoleContext.Provider>;
}
