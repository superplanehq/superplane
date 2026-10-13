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
  nodesLoading,
  nodeStatuses,
  canRunNodes,
  runNodesDisabledReason,
  onTriggerNode,
  onOpenNode,
  onSelectRun,
}: ConsoleContextProviderProps) {
  const value = useMemo<ConsoleContextValue>(
    () => ({
      canvasId,
      organizationId,
      nodes,
      nodesLoading,
      nodeStatuses,
      canRunNodes,
      runNodesDisabledReason,
      onTriggerNode,
      onOpenNode,
      onSelectRun,
    }),
    [
      canvasId,
      organizationId,
      nodes,
      nodesLoading,
      nodeStatuses,
      canRunNodes,
      runNodesDisabledReason,
      onTriggerNode,
      onOpenNode,
      onSelectRun,
    ],
  );

  return <ConsoleContext.Provider value={value}>{children}</ConsoleContext.Provider>;
}
