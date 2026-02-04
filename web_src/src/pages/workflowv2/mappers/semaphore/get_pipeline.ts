import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, ComponentBaseState } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";

interface PipelineOutput {
  name?: string;
  ppl_id?: string;
  wf_id?: string;
  state?: string;
  result?: string;
}

function baseProps(
  _nodes: ComponentsNode[],
  node: ComponentsNode,
  componentDefinition: ComponentsComponent,
  lastExecutions: CanvasesCanvasNodeExecution[],
  queueItems: CanvasesCanvasNodeQueueItem[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : undefined;

  let state: ComponentBaseState = "idle";
  if (queueItems.length > 0) {
    state = "queued";
  } else if (lastExecution?.state === "running") {
    state = "running";
  } else if (lastExecution?.state === "finished") {
    state = lastExecution.result === "success" ? "success" : "error";
  }

  return {
    id: node.id ?? "",
    state,
    label: componentDefinition.label ?? "Get Pipeline",
    icon: componentDefinition.icon ?? "workflow",
    color: componentDefinition.color ?? "gray",
  };
}

export const getPipelineMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    queueItems: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const pipeline = outputs.default[0].data as PipelineOutput;
      if (pipeline.name && pipeline.result) {
        return `${pipeline.name} - ${pipeline.result}`;
      }
      if (pipeline.ppl_id) {
        return pipeline.ppl_id.slice(0, 8);
      }
    }
    return "";
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs?.default?.[0]?.data) {
      const pipeline = outputs.default[0].data as PipelineOutput;

      if (pipeline.ppl_id) {
        details["Pipeline ID"] = pipeline.ppl_id;
      }
      if (pipeline.name) {
        details["Name"] = pipeline.name;
      }
      if (pipeline.wf_id) {
        details["Workflow ID"] = pipeline.wf_id;
      }
      if (pipeline.state) {
        details["State"] = pipeline.state;
      }
      if (pipeline.result) {
        details["Result"] = pipeline.result;
      }
    }

    return details;
  },
};
