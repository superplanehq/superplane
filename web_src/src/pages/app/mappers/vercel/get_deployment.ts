import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { executionSubtitle, propsWithMetadata, readDefaultOutput } from "./shared";
import { deploymentDetails, type VercelDeploymentOutput } from "./deploy";

interface GetDeploymentConfiguration {
  deploymentId?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetDeploymentConfiguration | undefined;

  if (configuration?.deploymentId) {
    metadata.push({ icon: "hash", label: `Deployment: ${configuration.deploymentId}` });
  }

  return metadata;
}

export const getDeploymentMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, metadataList),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const result = readDefaultOutput(context)?.data as VercelDeploymentOutput | undefined;

    return {
      ...deploymentDetails(result),
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };
  },
};
