import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { executionSubtitle, propsWithMetadata, readDefaultOutput, projectMetadata } from "./shared";
import { stringOrDash } from "./common";

function deploymentIdMetadata(label: string): (node: NodeInfo) => MetadataItem[] {
  return (node) => {
    const configuration = node.configuration as { deploymentId?: string } | undefined;
    const metadata: MetadataItem[] = [...projectMetadata(node)];
    if (configuration?.deploymentId) {
      metadata.push({ icon: "hash", label: `${label}: ${configuration.deploymentId}` });
    }
    return metadata;
  };
}

interface RollbackOutput {
  projectId?: string;
  deploymentId?: string;
  description?: string;
  rolledBack?: boolean;
}

export const cancelDeploymentMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, deploymentIdMetadata("Deployment")),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = readDefaultOutput(context);
    const result = output?.data as { readyState?: string; deploymentId?: string } | undefined;

    return {
      "Deployment ID": stringOrDash(result?.deploymentId),
      State: stringOrDash(result?.readyState),
      "Canceled At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };
  },
};

export const rollbackMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, deploymentIdMetadata("Rollback To")),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = readDefaultOutput(context);
    const result = output?.data as RollbackOutput | undefined;

    const details: Record<string, string> = {
      "Rolled Back To": stringOrDash(result?.deploymentId),
      Project: stringOrDash(result?.projectId),
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    if (result?.description) {
      details["Reason"] = result.description;
    }

    return details;
  },
};
