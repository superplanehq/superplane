import { createElement, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { cn, resolveIcon } from "@/lib/utils";
import { appPath } from "@/lib/appPaths";
import { useCanvas } from "@/hooks/useCanvasData";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import type { CanvasesCanvas, SuperplaneComponentsNode } from "@/api-client";
import { MetadataList } from "@/ui/metadataList";
import { BUILTIN_COMPONENT_ICON_SLUGS } from "./componentIcons";
import { getNodeHoverMetadataItems, listNodeNeighbors, type NodeNeighbor } from "./nodeChipHover";

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
  if (!node && !component) return "bg-slate-100 text-slate-600 dark:bg-gray-700 dark:text-gray-300";
  return trigger
    ? "bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-950 dark:text-violet-300 dark:hover:bg-violet-900"
    : "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-950 dark:text-blue-300 dark:hover:bg-blue-900";
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

function NeighborChip({ neighbor }: { neighbor: NodeNeighbor }) {
  const label = neighbor.direction === "upstream" ? `← ${neighbor.label}` : `${neighbor.label} →`;
  return (
    <span className="max-w-full truncate rounded-full bg-slate-100 px-2 py-0.5 text-[11px] text-slate-600 dark:bg-gray-800 dark:text-gray-300">
      {label}
    </span>
  );
}

function HoverSectionLabel({ children }: { children: string }) {
  return (
    <div className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-slate-400 dark:text-gray-500">
      {children}
    </div>
  );
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
        "flex items-center gap-2.5 border-b px-3.5 py-3",
        isTrigger
          ? "border-violet-200 bg-violet-100 dark:border-violet-800/60 dark:bg-violet-950/55"
          : "border-blue-200 bg-blue-100 dark:border-blue-800/60 dark:bg-blue-950/55",
      )}
    >
      <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-white/80 shadow-sm dark:bg-gray-900/70">
        <NodeHoverIcon component={component} isTrigger={isTrigger} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-semibold text-slate-900 dark:text-gray-100">{node.name || node.id}</p>
        <p className="text-[11px] text-slate-500 dark:text-gray-400">
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
        <HoverCardContent className="w-[280px] p-0" side="top" align="start">
          <NodeHoverContent node={node} nodes={nodes} edges={edges} component={component} />
        </HoverCardContent>
      )}
    </HoverCard>
  );
}

function NodeHoverContent({
  node,
  nodes,
  edges,
  component: resolvedComponent,
}: {
  node: SuperplaneComponentsNode;
  nodes: SuperplaneComponentsNode[] | undefined;
  edges: NonNullable<CanvasesCanvas["spec"]>["edges"];
  component?: string;
}) {
  const isTrigger = node.type === "TYPE_TRIGGER" || isTriggerNode(node, resolvedComponent);
  const component = resolvedComponent ?? resolveNodeComponent(node, node.id ?? "");
  const metadataItems = getNodeHoverMetadataItems(node, component);
  const neighbors = listNodeNeighbors(node.id, edges, nodes);
  const hasBody = Boolean(metadataItems.length > 0 || neighbors.items.length > 0 || node.errorMessage);

  return (
    <div>
      <NodeHoverHeader node={node} component={component} isTrigger={isTrigger} />

      {hasBody && (
        <div className="space-y-2.5 px-3.5 pt-3 pb-2.5">
          {metadataItems.length > 0 && (
            <div>
              <HoverSectionLabel>Details</HoverSectionLabel>
              <MetadataList
                items={metadataItems}
                maxVisibleItems={3}
                iconSize={14}
                className="flex flex-col gap-1 text-slate-600 dark:text-gray-300"
              />
            </div>
          )}

          {neighbors.items.length > 0 && (
            <div>
              <HoverSectionLabel>Connected to</HoverSectionLabel>
              <div className="flex flex-wrap gap-1">
                {neighbors.items.map((neighbor) => (
                  <NeighborChip key={`${neighbor.direction}:${neighbor.id}`} neighbor={neighbor} />
                ))}
                {neighbors.overflow > 0 && (
                  <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[11px] text-slate-500 dark:bg-gray-800 dark:text-gray-400">
                    +{neighbors.overflow} more
                  </span>
                )}
              </div>
            </div>
          )}

          {node.errorMessage && (
            <div className="flex items-start gap-1.5 rounded-lg bg-red-50 px-2.5 py-2 text-[11px] text-red-700 dark:bg-red-950/40 dark:text-red-300">
              <span className="shrink-0" aria-hidden>
                ⚠
              </span>
              <span className="min-w-0 truncate">{node.errorMessage}</span>
            </div>
          )}
        </div>
      )}

      <div className="border-t border-slate-100 px-3.5 py-2 text-[11px] text-slate-400 dark:border-gray-800 dark:text-gray-500">
        Click to open on canvas
      </div>
    </div>
  );
}
