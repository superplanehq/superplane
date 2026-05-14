import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Clock, Filter, Globe, Hand, Merge, Play, Split, Terminal, Webhook, type LucideIcon } from "lucide-react";
import type { CanvasesCanvas, SuperplaneComponentsNode } from "@/api-client";

const COMPONENT_ICONS: Record<string, LucideIcon> = {
  http: Globe,
  wait: Clock,
  webhook: Webhook,
  start: Play,
  if: Split,
  filter: Filter,
  ssh: Terminal,
  approval: Hand,
  merge: Merge,
  schedule: Clock,
};

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

export function NodeChip({ nodeId, label, canvasId, organizationId }: NodeChipProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const canvas = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
  const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
  const edges = canvas?.spec?.edges ?? [];

  const isTrigger = node?.type === "TYPE_TRIGGER";
  const iconSrc = node?.component ? getHeaderIconSrc(node.component) : undefined;
  const LucideIcon = node?.component ? COMPONENT_ICONS[node.component] : undefined;

  const handleClick = useCallback(() => {
    navigate(`/${organizationId}/canvases/${canvasId}?sidebar=1&node=${nodeId}`);
    // Dispatch a custom event for the canvas page to zoom to the node
    window.dispatchEvent(new CustomEvent("agent:focus-node", { detail: { nodeId } }));
  }, [navigate, organizationId, canvasId, nodeId]);

  const chipClassName = cn(
    "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ring-1 ring-inset transition-colors cursor-pointer align-middle whitespace-nowrap",
    node
      ? isTrigger
        ? "bg-purple-100 text-purple-700 ring-purple-300 hover:bg-purple-200"
        : "bg-blue-100 text-blue-700 ring-blue-300 hover:bg-blue-200"
      : "bg-slate-100 text-slate-600 ring-slate-300",
  );

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button type="button" onClick={handleClick} className={chipClassName} title={`Node: ${nodeId}`}>
          {iconSrc ? (
            <img src={iconSrc} alt="" className="size-3 object-contain shrink-0" />
          ) : LucideIcon ? (
            <LucideIcon className="size-3 shrink-0" />
          ) : (
            <span className={cn("size-2 rounded-full shrink-0", isTrigger ? "bg-purple-500" : "bg-blue-500")} />
          )}
          {label}
        </button>
      </HoverCardTrigger>
      {node && (
        <HoverCardContent className="w-64 p-0" side="top" align="start">
          <NodeHoverContent node={node} edges={edges} />
        </HoverCardContent>
      )}
    </HoverCard>
  );
}

function NodeHoverContent({
  node,
  edges,
}: {
  node: SuperplaneComponentsNode;
  edges: NonNullable<CanvasesCanvas["spec"]>["edges"];
}) {
  const isTrigger = node.type === "TYPE_TRIGGER";
  const iconSrc = node.component ? getHeaderIconSrc(node.component) : undefined;
  const NodeIcon = node.component ? COMPONENT_ICONS[node.component] : undefined;
  const config = node.configuration ?? {};

  // Count connections
  const incoming = (edges ?? []).filter((e) => e.targetId === node.id).length;
  const outgoing = (edges ?? []).filter((e) => e.sourceId === node.id).length;

  // Extract key config summary
  const summary = getConfigSummary(node.component, config);

  return (
    <div>
      {/* Header */}
      <div
        className={cn(
          "flex items-center gap-2 px-3 py-2 border-b border-slate-100",
          isTrigger ? "bg-purple-50" : "bg-blue-50",
        )}
      >
        {iconSrc ? (
          <img src={iconSrc} alt="" className="size-4 object-contain shrink-0" />
        ) : NodeIcon ? (
          <NodeIcon className="size-4 shrink-0" />
        ) : (
          <span className={cn("size-3 rounded-full shrink-0", isTrigger ? "bg-purple-400" : "bg-blue-400")} />
        )}
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium text-slate-900 truncate">{node.name || node.id}</p>
          <p className="text-[10px] text-slate-500">
            {node.component} · {isTrigger ? "Trigger" : "Action"}
          </p>
        </div>
      </div>

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

function getConfigSummary(component?: string, config?: Record<string, unknown>): string | null {
  if (!component || !config) return null;

  switch (component) {
    case "http":
      return `${config.method || "GET"} ${config.url || ""}`;
    case "ssh":
      return `${config.username || "root"}@${config.host || ""}`;
    case "if":
    case "filter":
      return String(config.expression || "");
    case "wait":
      return `Wait: ${config.duration || config.waitFor || ""}`;
    case "webhook":
      return `Auth: ${config.authentication || "none"}`;
    case "schedule":
      return `Cron: ${config.cron || ""}`;
    case "approval":
      return String(config.message || "Approval required");
    default:
      return null;
  }
}
