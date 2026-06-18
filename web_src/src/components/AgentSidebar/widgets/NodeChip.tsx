import { createElement, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { cn, resolveIcon } from "@/lib/utils";
import { appPath } from "@/lib/appPaths";
import { useCanvas } from "@/hooks/useCanvasData";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import type { CanvasesCanvas, SuperplaneComponentsNode } from "@/api-client";
import { BUILTIN_COMPONENT_ICON_SLUGS } from "./componentIcons";

const TRIGGER_COMPONENTS = new Set(["start", "schedule", "webhook"]);

/** Canvas node ids are `{block}-{name}-{6-char}`; legacy ids use `component-node-*`. */
function inferComponentFromNodeId(nodeId: string): string | undefined {
  if (nodeId.startsWith("component-node-")) return undefined;
  const parts = nodeId.split("-");
  if (parts.length < 3) return undefined;
  const suffix = parts[parts.length - 1];
  if (!/^[a-z0-9]{6}$/.test(suffix)) return undefined;
  return parts[0];
}

function resolveCanvasNode(
  nodes: SuperplaneComponentsNode[] | undefined,
  nodeId: string,
  label: string,
): SuperplaneComponentsNode | undefined {
  if (!nodes?.length) return undefined;
  const byId = nodes.find((n) => n.id === nodeId);
  if (byId) return byId;
  if (label && label !== nodeId) {
    const byName = nodes.find((n) => n.name === label);
    if (byName) return byName;
  }
  return undefined;
}

function resolveNodeComponent(node: SuperplaneComponentsNode | undefined, nodeId: string): string | undefined {
  if (node?.component) return node.component;
  if (node?.type === "TYPE_TRIGGER" && node.configuration && "templates" in node.configuration) {
    return "start";
  }
  return inferComponentFromNodeId(nodeId);
}

function isTriggerNode(node: SuperplaneComponentsNode | undefined, component?: string): boolean {
  if (node?.type === "TYPE_TRIGGER") return true;
  return component ? TRIGGER_COMPONENTS.has(component) : false;
}

interface NodeChipProps {
  nodeId: string;
  label: string;
  canvasId: string;
  organizationId: string;
}

export function NodeChipFromLink({
  nodeId,
  rawLabel,
  canvasId,
  organizationId,
}: {
  nodeId: string;
  rawLabel?: string;
  canvasId: string;
  organizationId: string;
}) {
  const label = rawLabel && rawLabel !== "node" ? rawLabel : nodeId;
  return <NodeChip nodeId={nodeId} label={label} canvasId={canvasId} organizationId={organizationId} />;
}

function getChipStyle(node: SuperplaneComponentsNode | undefined, component?: string, isTrigger?: boolean) {
  const trigger = isTrigger ?? isTriggerNode(node, component);
  if (!node && !component) return "bg-slate-100 text-slate-600";
  return trigger ? "bg-violet-100 text-violet-700 hover:bg-violet-200" : "bg-blue-100 text-blue-700 hover:bg-blue-200";
}

function NodeIconInline({ component, isTrigger }: { component?: string; isTrigger: boolean }) {
  const iconSrc = component ? getHeaderIconSrc(component) : undefined;
  if (iconSrc) return <img src={iconSrc} alt="" className="size-3 object-contain shrink-0" />;

  const slug = component ? BUILTIN_COMPONENT_ICON_SLUGS[component] : undefined;
  if (slug) return createElement(resolveIcon(slug), { className: "size-3 shrink-0" });

  return <span className={cn("size-2 rounded-full shrink-0", isTrigger ? "bg-violet-500" : "bg-blue-500")} />;
}

function NodeHoverIcon({ component, isTrigger }: { component?: string; isTrigger: boolean }) {
  const iconSrc = component ? getHeaderIconSrc(component) : undefined;
  if (iconSrc) return <img src={iconSrc} alt="" className="size-4 object-contain shrink-0" />;

  const builtinSlug = component ? BUILTIN_COMPONENT_ICON_SLUGS[component] : undefined;
  if (builtinSlug) return createElement(resolveIcon(builtinSlug), { className: "size-4 shrink-0" });

  return <span className={cn("size-3 rounded-full shrink-0", isTrigger ? "bg-violet-400" : "bg-blue-400")} />;
}

function NodeHoverHeader({
  node,
  component,
  isTrigger,
}: {
  node: SuperplaneComponentsNode;
  component?: string;
  isTrigger: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-2 border-b border-slate-100",
        isTrigger ? "bg-violet-50" : "bg-blue-50",
      )}
    >
      <NodeHoverIcon component={component} isTrigger={isTrigger} />
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium text-slate-900 truncate">{node.name || node.id}</p>
        <p className="text-[10px] text-slate-500">
          {component || "unknown"} · {isTrigger ? "Trigger" : "Action"}
        </p>
      </div>
    </div>
  );
}

export function NodeChip({ nodeId, label, canvasId, organizationId }: NodeChipProps) {
  const navigate = useNavigate();
  const { data: canvas } = useCanvas(organizationId, canvasId, {
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });

  const nodes = canvas?.spec?.nodes;
  const node = resolveCanvasNode(nodes, nodeId, label);
  const component = resolveNodeComponent(node, nodeId);
  const isTrigger = isTriggerNode(node, component);
  const edges = canvas?.spec?.edges ?? [];

  const handleClick = useCallback(() => {
    navigate(appPath(organizationId, canvasId, `?sidebar=1&node=${node?.id ?? nodeId}`));
    window.dispatchEvent(new CustomEvent("agent:focus-node", { detail: { nodeId: node?.id ?? nodeId } }));
  }, [navigate, organizationId, canvasId, node?.id, nodeId]);

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          onClick={handleClick}
          className={cn(
            "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors cursor-pointer align-middle whitespace-nowrap",
            getChipStyle(node, component, isTrigger),
          )}
          title={`Node: ${node?.id ?? nodeId}`}
        >
          <NodeIconInline component={component} isTrigger={isTrigger} />
          {label}
        </button>
      </HoverCardTrigger>
      {node && (
        <HoverCardContent className="w-64 p-0" side="top" align="start">
          <NodeHoverContent node={node} edges={edges} component={component} />
        </HoverCardContent>
      )}
    </HoverCard>
  );
}

function NodeHoverContent({
  node,
  edges,
  component: resolvedComponent,
}: {
  node: SuperplaneComponentsNode;
  edges: NonNullable<CanvasesCanvas["spec"]>["edges"];
  component?: string;
}) {
  const isTrigger = node.type === "TYPE_TRIGGER";
  const component = resolvedComponent ?? resolveNodeComponent(node, node.id ?? "");
  const config = node.configuration ?? {};

  // Count connections
  const incoming = (edges ?? []).filter((e) => e.targetId === node.id).length;
  const outgoing = (edges ?? []).filter((e) => e.sourceId === node.id).length;

  // Extract key config summary
  const summary = getConfigSummary(component, config);

  return (
    <div>
      {/* Header */}
      <NodeHoverHeader node={node} component={component} isTrigger={isTrigger} />

      {/* Config summary */}
      {summary && (
        <div className="px-3 py-2 border-b border-slate-100">
          <p className="text-[10px] text-slate-500 font-mono truncate">{summary}</p>
        </div>
      )}

      {/* Connections */}
      <div className="px-3 py-2 flex items-center gap-3 text-[10px] text-slate-500">
        <span>{incoming} incoming</span>
        <span>·</span>
        <span>{outgoing} outgoing</span>
      </div>

      {/* Error/warning */}
      {node.errorMessage && (
        <div className="px-3 py-1.5 bg-red-50 border-t border-red-100 text-[10px] text-red-600 truncate">
          ⚠ {node.errorMessage}
        </div>
      )}
    </div>
  );
}

const CONFIG_SUMMARIZERS: Record<string, (c: Record<string, unknown>) => string> = {
  http: (c) => `${c.method || "GET"} ${c.url || ""}`,
  ssh: (c) => `${c.username || "root"}@${c.host || ""}`,
  if: (c) => String(c.expression || ""),
  filter: (c) => String(c.expression || ""),
  wait: (c) => `Wait: ${c.duration || c.waitFor || ""}`,
  webhook: (c) => `Auth: ${c.authentication || "none"}`,
  schedule: (c) => `Cron: ${c.cron || ""}`,
  approval: (c) => String(c.message || "Approval required"),
};

function getConfigSummary(component?: string, config?: Record<string, unknown>): string | null {
  if (!component || !config) return null;
  const summarizer = CONFIG_SUMMARIZERS[component];
  return summarizer ? summarizer(config) : null;
}
// already at end of file
