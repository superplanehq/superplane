import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

interface ReleaseOutput {
  id?: number;
  tag_name?: string;
  name?: string;
  html_url?: string;
  created_at?: string;
  published_at?: string;
  author?: {
    login?: string;
  };
}

export const listReleasesMapper: ComponentBaseMapper = {
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
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { releases?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.releases) {
      const releases = outputs.releases[0].data as ReleaseOutput[];
      details["Total Releases"] = releases.length.toString();

      if (releases.length > 0) {
        const latest = releases[0];
        details["Latest Release Tag"] = latest.tag_name || "-";
        details["Latest Release Name"] = latest.name || "-";
        details["Latest Release URL"] = latest.html_url || "";
        details["Latest Release Date"] = latest.published_at 
          ? new Date(latest.published_at).toLocaleString() 
          : (latest.created_at ? new Date(latest.created_at).toLocaleString() : "-");
      }
    }

    return details;
  },
};
