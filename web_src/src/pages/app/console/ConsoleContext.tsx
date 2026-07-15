import { createContext, useContext } from "react";

import type { SuperplaneComponentsNode } from "@/api-client";

/**
 * Public status shape used by status chips. Mirrors the categories already
 * surfaced by the canvas UI (passed / failed / running / etc.) but normalized
 * to a small enum so the dashboard does not have to know about every API
 * variant.
 */
export type ConsoleNodeStatus = "passed" | "failed" | "running" | "pending" | "cancelled" | "skipped" | "unknown";

export interface ConsoleTriggerOptions {
  /** Trigger hook name (default `run`). */
  hookName?: string;
  /** Start template name when applicable. */
  templateName?: string;
  /** @deprecated Alias for `templateName`. */
  triggerName?: string;
  /** Pre-built hook parameters (merged by the caller). */
  parameters?: Record<string, unknown>;
  /** Toast label after success (defaults to "Triggered node"). */
  successLabel?: string;
}

export interface ConsoleContextValue {
  canvasId: string;
  organizationId: string;
  /** All canvas nodes available for chip resolution. */
  nodes: SuperplaneComponentsNode[];
  /** Optional latest-status map keyed by node id. */
  nodeStatuses?: Record<string, ConsoleNodeStatus | undefined>;
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
  onTriggerNode?: (nodeId: string, options?: ConsoleTriggerOptions) => void | Promise<void>;
  /**
   * Optional callback when the user opens a node chip (e.g. to focus / scroll
   * the corresponding canvas node into view). Falls back to navigation.
   */
  onOpenNode?: (nodeId: string) => void;
}

export const ConsoleContext = createContext<ConsoleContextValue | undefined>(undefined);

export function useConsoleContext(): ConsoleContextValue | undefined {
  return useContext(ConsoleContext);
}

/**
 * Resolve a textual node reference to its concrete node id and friendly label.
 * Accepts either the canvas node id (UUID) or the node name (e.g. `deploy-prod`).
 *
 * Returns `undefined` when the reference doesn't match any current node.
 */
export function resolveConsoleNode(
  ctx: Pick<ConsoleContextValue, "nodes"> | undefined,
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

/**
 * Resolve a trigger-filter reference (id or name) to a `TYPE_TRIGGER` node.
 * Unlike {@link resolveConsoleNode}, this ignores actions and other node
 * types so a shared name cannot silently bind a run filter to the wrong id.
 */
export function resolveConsoleTrigger(
  ctx: Pick<ConsoleContextValue, "nodes"> | undefined,
  reference: string,
): { node: SuperplaneComponentsNode; label: string } | undefined {
  if (!ctx) return undefined;
  const trimmed = reference.trim();
  if (!trimmed) return undefined;
  const triggers = ctx.nodes.filter((n) => n.type === "TYPE_TRIGGER");
  const byId = triggers.find((n) => n.id === trimmed);
  if (byId) return { node: byId, label: byId.name || byId.id || trimmed };
  const byName = triggers.find((n) => n.name === trimmed);
  if (byName) return { node: byName, label: byName.name || byName.id || trimmed };
  return undefined;
}
