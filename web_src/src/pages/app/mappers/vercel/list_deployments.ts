import type { ComponentBaseMapper, ExecutionDetailsContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { NodeInfo } from "../types";
import { executionSubtitle, propsWithMetadata, readDefaultOutput, projectMetadata } from "./shared";
import { stringOrDash } from "./common";

interface ListDeploymentsConfiguration {
  project?: string;
  target?: string;
  state?: string;
}

interface DeploymentListItem {
  deploymentId?: string;
  name?: string;
  url?: string;
  readyState?: string;
  target?: string;
}

interface ListDeploymentsOutput {
  deployments?: DeploymentListItem[];
  count?: number;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as ListDeploymentsConfiguration | undefined;
  const metadata: MetadataItem[] = [...projectMetadata(node)];

  if (configuration?.target) {
    metadata.push({ icon: "flag", label: `Target: ${configuration.target}` });
  }
  if (configuration?.state) {
    metadata.push({ icon: "activity", label: `State: ${configuration.state}` });
  }

  return metadata;
}

export const listDeploymentsMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, metadataList),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = readDefaultOutput(context);
    const result = output?.data as ListDeploymentsOutput | undefined;

    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Count: stringOrDash(result?.count),
    };

    const latest = result?.deployments?.[0];
    if (latest) {
      details["Latest Deployment"] = `${latest.name || latest.deploymentId} · ${latest.readyState}`;
      if (latest.url) {
        details["Latest URL"] = latest.url;
      }
    }

    return details;
  },
};
