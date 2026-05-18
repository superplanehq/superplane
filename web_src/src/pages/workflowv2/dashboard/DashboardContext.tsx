import { createContext, useContext, useMemo, type ReactNode } from "react";

import type { SuperplaneComponentsNode } from "@/api-client";

/**
 * Public status shape used by status chips. Mirrors the categories already
 * surfaced by the canvas UI (passed / failed / running / etc.) but normalized
 * to a small enum so the dashboard does not have to know about every API
 * variant.
 */
export type DashboardNodeStatus = "passed" | "failed" | "running" | "pending" | "cancelled" | "skipped" | "unknown";

export interface DashboardContextValue {
  canvasId: string;
  organizationId: string;
  /** All canvas nodes available for chip resolution. */
  nodes: SuperplaneComponentsNode[];
  /** Optional latest-status map keyed by node id. */
  nodeStatuses?: Record<string, DashboardNodeStatus | undefined>;
  /**
   * Runtime authorization flag — true when the viewer is allowed to invoke
   * manual triggers, approvals, cancellations, and push-through actions on
   * the underlying canvas. Maps to the same `canvases:update` permission the
   * gRPC interceptor enforces on `InvokeNodeTriggerHook` /
   * `InvokeNodeExecutionHook`; the UI mirrors that so users without the
   * permission see disabled controls instead of clicks that silently fail.
   */
  canRunNodes: boolean;
  /**
   * Open the manual-trigger flow for the given node. Resolution is by node id;
   * if undefined the chip falls back to dispatching the
   * `dashboard:trigger-node` window event so a host can react when wired.
   */
  onTriggerNode?: (nodeId: string, options?: { templateName?: string; triggerName?: string }) => void;
  /**
   * Optional callback when the user opens a node chip (e.g. to focus / scroll
   * the corresponding canvas node into view). Falls back to navigation.
   */
  onOpenNode?: (nodeId: string) => void;
}

const DashboardContext = createContext<DashboardContextValue | undefined>(undefined);

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

export function useDashboardContext(): DashboardContextValue | undefined {
  return useContext(DashboardContext);
}

/**
 * Resolve a textual node reference to its concrete node id and friendly label.
 * Accepts either the canvas node id (UUID) or the node name (e.g. `deploy-prod`).
 *
 * Returns `undefined` when the reference doesn't match any current node.
 */
export function resolveDashboardNode(
  ctx: Pick<DashboardContextValue, "nodes"> | undefined,
  reference: string,
): { node: SuperplaneComponentsNode; label: string } | undefined {
  if (!ctx) return undefined;
  const trimmed = reference.trim();
  if (!trimmed) return undefined;
  const byId = ctx.nodes.find((n) => n.id === trimmed);
  if (byId) return { node: byId, label: byId.name || byId.id || trimmed };
  const byName = ctx.nodes.find((n) => n.name === trimmed);
  if (byName) return { node: byName, label: byName.name || byName.id || trimmed };
  return undefined;
}
