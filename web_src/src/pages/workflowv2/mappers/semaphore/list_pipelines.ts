import type {
  ComponentBaseMapper,
  ComponentBaseProps,
  OutputPayload,
} from "../../component-props";
import type { ExecutionWithNode } from "../../types";
import type { NodeData } from "../../workflow-types";

interface PipelineItem {
  ppl_id: string;
  name: string;
  wf_id: string;
  state: string;
  result: string;
}

export const listPipelinesMapper: ComponentBaseMapper = {
  props(
    _data: NodeData,
    _execution: ExecutionWithNode | null
  ): ComponentBaseProps {
    return {};
  },

  subtitle(_node: NodeData, execution: ExecutionWithNode | null): string {
    const outputs = execution?.outputs as
      | { default?: OutputPayload[] }
      | undefined;
    if (outputs?.default?.[0]?.data) {
      const pipelines = outputs.default[0].data as PipelineItem[];
      if (Array.isArray(pipelines) && pipelines.length > 0) {
        return `${pipelines.length} pipeline${pipelines.length === 1 ? "" : "s"}`;
      }
    }
    return "";
  },

  getExecutionDetails(
    execution: ExecutionWithNode | null,
    _node: NodeData
  ): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = execution?.outputs as
      | { default?: OutputPayload[] }
      | undefined;

    if (outputs?.default?.[0]?.data) {
      const pipelines = outputs.default[0].data as PipelineItem[];
      if (Array.isArray(pipelines)) {
        details["Total Pipelines"] = String(pipelines.length);

        const passed = pipelines.filter((p) => p.result === "passed").length;
        const failed = pipelines.filter((p) => p.result === "failed").length;
        const running = pipelines.filter((p) => p.state === "running").length;

        if (passed > 0) details["Passed"] = String(passed);
        if (failed > 0) details["Failed"] = String(failed);
        if (running > 0) details["Running"] = String(running);
      }
    }

    return details;
  },
};
