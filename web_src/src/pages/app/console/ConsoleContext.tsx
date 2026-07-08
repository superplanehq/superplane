import { createContext, useContext } from "react";

import type { SuperplaneComponentsNode, TriggersTrigger } from "@/api-client";

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
   * Names of triggers (component identifiers, e.g. `start`, `schedule`) that
   * expose a user-invokable `run` hook — the same condition
   * `InvokeNodeTriggerHook` enforces on the backend. When defined, console
   * widgets use it to hide manual-run affordances (Run buttons, table row
   * actions) for event triggers such as `github.pullRequest`. `undefined`
   * strictly means the catalog is still loading — consumers then fall back
   * to the previous `TYPE_TRIGGER`-only heuristic to avoid flicker on first
   * paint. A failed fetch yields an empty set (fail closed) instead.
   */
  manualRunTriggers?: ReadonlySet<string>;
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

const NO_MANUAL_RUN_TRIGGERS: ReadonlySet<string> = new Set();

/**
 * Build the manual-run trigger catalog from the triggers API response.
 *
 * Returns `undefined` only while the catalog is genuinely still loading so
 * {@link isManualRunNode} keeps the permissive first-paint fallback. When the
 * fetch has failed we fail closed with an empty set instead — otherwise
 * event-only triggers would keep showing Run controls that the backend would
 * reject anyway.
 */
export function manualRunTriggersFromCatalog(
  triggers: TriggersTrigger[] | undefined,
  fetchFailed: boolean,
): ReadonlySet<string> | undefined {
  if (!triggers) return fetchFailed ? NO_MANUAL_RUN_TRIGGERS : undefined;
  const names = new Set<string>();
  for (const trigger of triggers) {
    if (trigger.manualRunnable && trigger.name) names.add(trigger.name);
  }
  return names;
}

/**
 * Single source of truth for "this node can be manually run from the
 * console". Combines the structural `TYPE_TRIGGER` check with the trigger
 * catalog's `manualRunnable` bit (populated from the backend's user `run`
 * hook). While the catalog is still loading (`manualRunTriggers` undefined)
 * we intentionally return `true` for any `TYPE_TRIGGER` so first paint keeps
 * the previous behavior — the widgets will re-render once the query settles.
 *
 * Nodes without a `component` (unlikely at runtime, but possible in
 * partially normalized fixtures) are rejected once the catalog is loaded.
 */
export function isManualRunNode(
  ctx: Pick<ConsoleContextValue, "manualRunTriggers"> | undefined,
  node: SuperplaneComponentsNode | undefined,
): boolean {
  if (!node) return false;
  if (node.type !== "TYPE_TRIGGER") return false;
  const catalog = ctx?.manualRunTriggers;
  if (catalog === undefined) return true;
  const component = node.component;
  return Boolean(component) && catalog.has(component!);
}
