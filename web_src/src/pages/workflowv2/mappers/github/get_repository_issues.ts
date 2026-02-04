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
import { MetadataItem } from "@/ui/metadataList";

interface IssueOutput {
  id?: number;
  number?: number;
  title?: string;
  state?: string;
  html_url?: string;
  comments_count?: number;
  created_at?: string;
  updated_at?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
  labels?: Array<{
    name?: string;
    color?: string;
  }>;
  assignees?: Array<{
    login?: string;
  }>;
}

interface GetRepositoryIssuesConfiguration {
  repository?: string;
  state?: string;
  labels?: string;
  sort?: string;
  direction?: string;
  perPage?: number;
}

function getRepositoryIssuesMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetRepositoryIssuesConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.state) {
    metadata.push({ icon: "filter", label: `State: ${configuration.state}` });
  }

  if (configuration?.labels) {
    metadata.push({ icon: "tag", label: `Labels: ${configuration.labels}` });
  }

  return metadata;
}

export const getRepositoryIssuesMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    queueItems: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const base = baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);

    return {
      ...base,
      metadata: getRepositoryIssuesMetadataList(node),
    };
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default && Array.isArray(outputs.default[0]?.data)) {
      const issues = outputs.default[0].data as IssueOutput[];
      const count = issues.length;
      return buildGithubExecutionSubtitle(execution, `${count} issue${count !== 1 ? "s" : ""}`);
    }
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Retrieved At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
    });

    if (outputs?.default && Array.isArray(outputs.default[0]?.data)) {
      const issues = outputs.default[0].data as IssueOutput[];
      details["Issues Found"] = issues.length.toString();

      if (issues.length > 0) {
        const openCount = issues.filter((i) => i.state === "open").length;
        const closedCount = issues.filter((i) => i.state === "closed").length;
        if (openCount > 0) details["Open"] = openCount.toString();
        if (closedCount > 0) details["Closed"] = closedCount.toString();

        // Show first issue as preview
        const first = issues[0];
        if (first.html_url) {
          details["First Issue"] = first.html_url;
        }
      }
    }

    return details;
  },
};
