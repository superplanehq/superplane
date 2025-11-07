import { useParams } from "react-router-dom";

import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { useNodeExecutions, useWorkflow } from "@/hooks/useWorkflowData";
import { getTriggerRenderer } from "@/pages/workflowv2/renderers";
import { CanvasPage } from "@/ui/CanvasPage";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { useRef } from "react";

export function NodeRunPage() {
  const { organizationId, workflowId } = useParams();
  const { data: workflow } = useWorkflow(organizationId!, workflowId!);

  const isSidebarOpenRef = useRef<boolean | null>(false);
  const breadcrumbs = useBreadcrumbs();
  const workflowName = workflow?.metadata?.name || "Workflow";

  return (
    <div className="h-screen w-screen bg-slate-50">
      <CanvasPage
        title={workflowName}
        breadcrumbs={breadcrumbs}
        organizationId={organizationId}
        nodes={[]}
        edges={[]}
        buildingBlocks={[]}
        isSidebarOpenRef={isSidebarOpenRef}
      />
    </div>
  );
}

function useBreadcrumbs() {
  const { organizationId, workflowId, nodeId } = useParams();
  const { data: workflow } = useWorkflow(organizationId || "", workflowId || "");
  const { data: nodeExecs } = useNodeExecutions(workflowId || "", nodeId || "");
  const { data: blueprints = [] } = useBlueprints(organizationId || "");
  const { data: components = [] } = useComponents(organizationId || "");

  const nodeName = workflow?.spec?.nodes?.find((n) => n.id === nodeId)?.name || "Component";
  const latestExecution = nodeExecs?.executions?.[0];
  const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

  const latestRunTitle = (() => {
    if (!latestExecution) return undefined;
    const rootNode = workflow?.spec?.nodes?.find((n) => n.id === latestExecution.rootEvent?.nodeId);
    const renderer = getTriggerRenderer(rootNode?.trigger?.name || "");
    if (latestExecution.rootEvent) {
      return renderer.getTitleAndSubtitle(latestExecution.rootEvent).title;
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
