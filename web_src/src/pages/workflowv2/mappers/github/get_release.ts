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
  created_at?: string;
  published_at?: string;
  author?: {
    login?: string;
    html_url?: string;
  };
  assets?: Array<{
    name?: string;
    size?: number;
    download_count?: number;
  }>;
}

export const getReleaseMapper: ComponentBaseMapper = {
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
      const release = outputs.default[0].data as ReleaseOutput;
      Object.assign(details, {
        "Retrieved At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
      });

      details["Release URL"] = release?.html_url || "";
      details["Tag Name"] = release?.tag_name || "";

      if (release?.name) {
        details["Release Name"] = release.name;
      }

      if (release?.author?.login) {
        details["Author"] = release.author.html_url || release.author.login;
      }

      if (release?.draft !== undefined) {
        details["Draft"] = release.draft ? "Yes" : "No";
      }

      if (release?.prerelease !== undefined) {
        details["Prerelease"] = release.prerelease ? "Yes" : "No";
      }

      if (release?.published_at) {
        details["Published At"] = new Date(release.published_at).toLocaleString();
      }

      if (release?.assets && release.assets.length > 0) {
        details["Assets"] = release.assets.length.toString();
      }
    }

    return details;
  },
};
