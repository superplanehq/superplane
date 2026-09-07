import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { executionSubtitle, propsWithMetadata, readDefaultOutput } from "./shared";
import { formatTimestamp, stringOrDash } from "./common";

export interface DeployConfiguration {
  project?: string;
  gitRef?: string;
  target?: string;
}

export interface VercelDeploymentOutput {
  deploymentId?: string;
  name?: string;
  url?: string;
  readyState?: string;
  target?: string;
  projectId?: string;
  createdAt?: number;
  ref?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeployConfiguration | undefined;

  if (configuration?.project) {
    metadata.push({ icon: "database", label: `Project: ${configuration.project}` });
  }
  if (configuration?.gitRef) {
    metadata.push({ icon: "gitBranch", label: `Ref: ${configuration.gitRef}` });
  }
  if (configuration?.target) {
    metadata.push({ icon: "flag", label: `Target: ${configuration.target}` });
  }

  return metadata;
}

/** Shared execution details for components that emit a vercel.deployment payload. */
export function deploymentDetails(result: VercelDeploymentOutput | undefined): Record<string, string> {
  return {
    "Deployment ID": stringOrDash(result?.deploymentId),
    Project: stringOrDash(result?.name || result?.projectId),
    URL: stringOrDash(result?.url),
    State: stringOrDash(result?.readyState),
    Target: stringOrDash(result?.target),
    Ref: stringOrDash(result?.ref),
    "Created At": formatTimestamp(result?.createdAt ? new Date(result.createdAt).toISOString() : undefined),
  };
}

function readOutput(context: ExecutionDetailsContext): VercelDeploymentOutput | undefined {
  return readDefaultOutput(context)?.data as VercelDeploymentOutput | undefined;
}

export const deployMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, metadataList),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return {
      ...deploymentDetails(readOutput(context)),
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };
  },
};
