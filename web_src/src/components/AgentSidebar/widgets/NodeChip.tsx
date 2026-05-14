import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useReactFlow } from "@xyflow/react";
import { useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { getSafeComponentProps, getSafeTriggerProps } from "@/ui/CanvasPage/Block/data";
import { ComponentBase } from "@/ui/componentBase";
import { Trigger } from "@/ui/trigger";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import type { CanvasesCanvas } from "@/api-client";
import type { BlockData } from "@/ui/CanvasPage/Block/types";

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
  const reactFlow = useReactFlow();

  const canvas = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
  const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);

  const handleClick = useCallback(() => {
    // Navigate to open sidebar
    navigate(`/${organizationId}/canvases/${canvasId}?sidebar=1&node=${nodeId}`);

    // Select the node and zoom to it
    if (reactFlow && node) {
      try {
        const allNodes = reactFlow.getNodes();
        const rfNode = allNodes.find((n) => n.id === nodeId);

        if (rfNode) {
          // Select only this node
          reactFlow.setNodes((nodes) =>
            nodes.map((n) => ({
              ...n,
              selected: n.id === nodeId,
            })),
          );

          // Zoom to the node
          reactFlow.fitView({
            nodes: [rfNode],
            duration: 500,
            maxZoom: 1.2,
            padding: 0.2,
          });
        }
      } catch (error) {
        console.error("Failed to focus node:", error);
      }
    }
  }, [navigate, organizationId, canvasId, nodeId, reactFlow, node]);

  if (!node) {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ring-1 ring-inset bg-slate-100 text-slate-600 ring-slate-300 align-middle">
        {label}
      </span>
    );
  }

  const nodeType = node.type;
  const isTrigger = nodeType === "TYPE_TRIGGER";
  const blockName = node.action?.name || node.trigger?.name;
  const iconSrc = blockName ? getHeaderIconSrc(blockName) : undefined;

  // Color based on node type
  const chipClassName = cn(
    "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ring-1 ring-inset transition-colors cursor-pointer align-middle",
    isTrigger
      ? "bg-purple-100 text-purple-700 ring-purple-300 hover:bg-purple-200"
      : "bg-blue-100 text-blue-700 ring-blue-300 hover:bg-blue-200",
  );

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button type="button" onClick={handleClick} className={chipClassName} title={`Node ${nodeId}`}>
          {iconSrc ? (
            <img src={iconSrc} alt="" className="size-3 object-contain flex-shrink-0" />
          ) : (
            <span className="size-3 rounded-full bg-current opacity-50 flex-shrink-0" />
          )}
          {label}
        </button>
      </HoverCardTrigger>
      <HoverCardContent className="w-[420px] p-0" side="right" align="start">
        <NodePreview node={node} />
      </HoverCardContent>
    </HoverCard>
  );
}

function NodePreview({ node }: { node: NonNullable<CanvasesCanvas["spec"]>["nodes"][number] }) {
  const blockData: BlockData = {
    type: node.type === "TYPE_TRIGGER" ? "trigger" : "component",
    label: node.label || node.id || "Node",
    outputChannels: node.outputChannels,
    trigger: node.trigger,
    component: node.action,
    composite: undefined,
    annotation: undefined,
  };

  if (blockData.type === "trigger" && blockData.trigger) {
    const triggerProps = getSafeTriggerProps(blockData);
    return (
      <div className="pointer-events-none">
        <Trigger {...triggerProps} canvasMode="live" selected={false} showHeader={true} />
      </div>
    );
  }

  if (blockData.type === "component" && blockData.component) {
    const componentProps = getSafeComponentProps(blockData);
    return (
      <div className="pointer-events-none">
        <ComponentBase {...componentProps} canvasMode="live" selected={false} showHeader={true} />
      </div>
    );
  }

  return (
    <div className="p-4 text-sm text-muted-foreground">
      <p>Preview unavailable</p>
    </div>
  );
}
