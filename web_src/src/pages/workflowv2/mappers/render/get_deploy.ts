import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestamp, stringOrDash } from "./common";
import { baseProps } from "./base";

interface GetDeployConfiguration {
  service?: string;
  deployId?: string;
}

interface DeployCommitOutput {
  id?: string;
  message?: string;
  createdAt?: string;
}

interface DeployImageOutput {
  ref?: string;
  sha?: string;
}

interface GetDeployOutput {
  serviceId?: string;
  deployId?: string;
  status?: string;
  trigger?: string;
  createdAt?: string;
  updatedAt?: string;
  startedAt?: string;
  finishedAt?: string;
  rollbackToDeployId?: string;
  commit?: DeployCommitOutput;
  image?: DeployImageOutput;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetDeployConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }
  if (configuration?.deployId) {
    metadata.push({ icon: "hash", label: `Deploy: ${configuration.deployId}` });
  }

  return metadata;
}

export const getDeployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetDeployOutput | undefined;

    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Service ID": stringOrDash(result?.serviceId),
      "Deploy ID": stringOrDash(result?.deployId),
      Status: stringOrDash(result?.status),
      Trigger: stringOrDash(result?.trigger),
      "Created At": formatTimestamp(result?.createdAt),
      "Started At": formatTimestamp(result?.startedAt),
      "Finished At": formatTimestamp(result?.finishedAt),
    };

    if (result?.rollbackToDeployId) {
      details["Rollback To"] = result.rollbackToDeployId;
    }

    if (result?.commit?.id) {
      details["Commit ID"] = result.commit.id;
    }
    if (result?.commit?.message) {
      details["Commit Message"] = result.commit.message;
    }
    if (result?.image?.ref) {
      details["Image Ref"] = result.image.ref;
    }
    if (result?.image?.sha) {
      details["Image SHA"] = result.image.sha;
    }

    if (result?.updatedAt) {
      details["Updated At"] = formatTimestamp(result.updatedAt);
    }

    return details;
  },
};
