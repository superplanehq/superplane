import { useMemo, useRef } from "react";
import { useParams } from "react-router-dom";

import { BlueprintsBlueprint, ComponentsComponent, ComponentsEdge, ComponentsNode } from "@/api-client";
import { useBlueprint, useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useChildExecutions, useWorkflow } from "@/hooks/useWorkflowData";
import { getTriggerRenderer } from "@/pages/workflowv2/renderers";
import { CanvasEdge, CanvasNode, CanvasPage } from "@/ui/CanvasPage";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";

export function NodeRunPage() {
  const { organizationId, workflowId, nodeId, executionId } = useParams();
  const { data: workflow } = useWorkflow(organizationId!, workflowId!);

  usePageTitle([workflow?.metadata?.name]);

  // Node details within the workflow
  const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

  // If this is a blueprint node, load the blueprint definition
  const { data: blueprint } = useBlueprint(organizationId || "", node?.blueprint?.id || "");

  // Fetch child executions based on the executionId from the URL
  const { data: childExecsResp } = useChildExecutions(workflowId || "", executionId || null);
  const childExecs = childExecsResp?.executions || [];

  // Components metadata for icon/color/channel information
  const { data: components = [] } = useComponents(organizationId || "");

  // Compute nodes/edges for CanvasPage from the blueprint spec filtered to executed nodes
  const { nodes, edges } = useMemo(() => {
    if (!blueprint) return { nodes: [] as CanvasNode[], edges: [] as CanvasEdge[] };

    // Map of executed child node IDs (strip any namespace prefixes like parent:child)
    const executedIds = new Set(
      childExecs
        .map((e) => (e.nodeId ? e.nodeId.split(":").pop() || e.nodeId : undefined))
        .filter((v): v is string => !!v),
    );

    // If we have no child executions yet, show all nodes so the user can see the blueprint
    const nodesToShow = (blueprint.nodes || []).filter((n) => executedIds.size === 0 || executedIds.has(n.id!));

    // Map child executions by node id suffix so we can attach last run info
    const execByNodeId = new Map<string, any>();
    for (const ce of childExecs) {
      const suffix = ce.nodeId ? ce.nodeId.split(":").pop() || ce.nodeId : undefined;
      if (suffix) execByNodeId.set(suffix, ce);
    }

    const canvasNodes: CanvasNode[] = nodesToShow.map((n) =>
      buildCanvasNode(n, components, blueprint, execByNodeId.get(n.id!)),
    );

    // Only keep edges where both ends are visible
    const visibleIds = new Set(canvasNodes.map((n) => n.id));
    const canvasEdges: CanvasEdge[] = (blueprint.edges || [])
      .filter((e) => visibleIds.has(e.sourceId!) && visibleIds.has(e.targetId!))
      .map((e) => buildCanvasEdge(e));

    return { nodes: canvasNodes, edges: canvasEdges };
  }, [blueprint, components, childExecs]);

  const isSidebarOpenRef = useRef<boolean | null>(false);
  const breadcrumbs = useBreadcrumbs();
  const workflowName = workflow?.metadata?.name || "Workflow";

  return (
    <div className="h-screen w-screen bg-slate-50">
      <CanvasPage
        title={workflowName}
        breadcrumbs={breadcrumbs}
        organizationId={organizationId}
        nodes={nodes}
        edges={edges}
        buildingBlocks={[]}
        isSidebarOpenRef={isSidebarOpenRef}
      />
    </div>
  );
}

function buildCanvasEdge(edge: ComponentsEdge): CanvasEdge {
  const idParts = [edge.sourceId, edge.targetId, edge.channel].filter(Boolean).join("-");
  return {
    id: idParts,
    source: edge.sourceId!,
    target: edge.targetId!,
    sourceHandle: edge.channel || undefined,
    data: {},
  } as CanvasEdge;
}

function buildCanvasNode(
  node: ComponentsNode,
  components: ComponentsComponent[],
  blueprint?: BlueprintsBlueprint,
  execution?: any,
): CanvasNode {
  // Resolve metadata for icons/colors/output channels
  const comp = components.find((c) => c.name === node.component?.name);
  const color = comp?.color || blueprint?.color || "indigo";
  const label = node.name || comp?.label || comp?.name || node.id || "Component";

  const outputChannels = comp?.outputChannels?.map((c) => c.name!) || ["default"];

  const canvasNode: CanvasNode = {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "composite",
      label,
      state: "pending",
      outputChannels,
      composite: {
        iconSlug: comp?.icon || blueprint?.icon || "box",
        iconColor: getColorClass(color),
        iconBackground: getBackgroundColorClass(color),
        headerColor: getBackgroundColorClass(color),
        collapsedBackground: getBackgroundColorClass(color),
        collapsed: node.isCollapsed || false,
        title: label,
        description: comp?.description || blueprint?.description,
        parameters: [],
      },
    },
  } as CanvasNode;

  if (execution) {
    const state = getRunItemState(execution);
    const title = friendlyChildLabel(execution, blueprint?.nodes || []);
    (canvasNode.data.composite as any).lastRunItem = {
      title,
      subtitle: execution.id,
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : new Date(),
      state,
      values: {},
    };
  }

  return canvasNode;
}

function getRunItemState(execution: any): "success" | "failed" | "running" {
  if (execution?.state === "STATE_PENDING" || execution?.state === "STATE_STARTED") return "running";
  if (execution?.state === "STATE_FINISHED" && execution?.result === "RESULT_PASSED") return "success";
  return "failed";
}

function friendlyChildLabel(ce: any, nodes: ComponentsNode[]) {
  const meta: any = ce?.metadata || {};
  const metaLabel =
    meta.title || meta.nodeTitle || meta.nodeName || meta.nodeLabel || meta.displayName || meta.name || meta.label;
  if (metaLabel && typeof metaLabel === "string" && metaLabel.trim().length > 0) return metaLabel as string;

  const fromGraph = nodes.find((n) => n.id === (ce?.nodeId?.split(":").pop() || ce?.nodeId))?.name;
  if (fromGraph) return fromGraph;

  const raw = (ce?.nodeId || "").toString();
  const afterColon = raw.includes(":") ? raw.split(":").pop()! : raw;
  const parts = afterColon.split("-");
  if (parts.length > 1 && /^[a-z0-9]{5,}$/.test(parts[parts.length - 1])) {
    parts.pop();
  }
  const deduped: string[] = [];
  for (const p of parts) {
    if (deduped.length === 0 || deduped[deduped.length - 1] !== p) deduped.push(p);
  }
  const label = deduped.join(" ");
  return label.replace(/\b\w/g, (c) => c.toUpperCase());
}

function useBreadcrumbs() {
  const { organizationId, workflowId, nodeId, executionId } = useParams();
  const { data: workflow } = useWorkflow(organizationId || "", workflowId || "");
  const { data: childExecsResp } = useChildExecutions(workflowId || "", executionId || null);
  const { data: blueprints = [] } = useBlueprints(organizationId || "");
  const { data: components = [] } = useComponents(organizationId || "");

  const nodeName = workflow?.spec?.nodes?.find((n) => n.id === nodeId)?.name || "Component";
  const selectedExecution = childExecsResp?.executions?.[0];
  const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

  const latestRunTitle = (() => {
    if (!selectedExecution) return undefined;
    const rootNode = workflow?.spec?.nodes?.find((n) => n.id === selectedExecution.rootEvent?.nodeId);
    const renderer = getTriggerRenderer(rootNode?.trigger?.name || "");
    if (selectedExecution.rootEvent) {
      return renderer.getTitleAndSubtitle(selectedExecution.rootEvent).title;
    }
    return "Execution";
  })();

  let iconSlug: string | undefined;
  let color: string | undefined;
  if (node?.blueprint?.id) {
    const bp = blueprints.find((b) => b.id === node.blueprint?.id);
    iconSlug = bp?.icon || undefined;
    color = bp?.color || undefined;
  } else if (node?.component?.name) {
    const comp = components.find((c) => c.name === node.component?.name);
    iconSlug = comp?.icon || undefined;
    color = comp?.color || undefined;
  } else if (node?.trigger?.name) {
    // triggers not fetched here; fall back to default
    iconSlug = "bolt";
    color = "blue";
  }

  return [
    { label: "Canvases", href: `/${organizationId}` },
    { label: workflow?.metadata?.name || "Workflow", href: `/${organizationId}/workflows/${workflowId}` },
    {
      label: nodeName,
      iconSlug: iconSlug || "boxes",
      iconColor: getColorClass(color || "indigo"),
      iconBackground: getBackgroundColorClass(color || "indigo"),
    },
    ...(latestRunTitle ? [{ label: latestRunTitle }] : []),
  ];
}

export default NodeRunPage;
