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
  draft?: boolean;
  prerelease?: boolean;
  published_at?: string;
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
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const releases = outputs.default[0].data as ReleaseOutput[];
      
      if (Array.isArray(releases)) {
        details["Releases Found"] = releases.length.toString();
        
        if (releases.length > 0) {
          // Show first release info
          const firstRelease = releases[0];
          details["Latest Tag"] = firstRelease?.tag_name || "-";
          details["Latest Name"] = firstRelease?.name || "-";
          
          if (firstRelease?.published_at) {
            details["Latest Published"] = new Date(firstRelease.published_at).toLocaleString();
          }
        }
      }
    }

    return details;
  },
};
