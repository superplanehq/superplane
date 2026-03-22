import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp, stringOrDash } from "./common";
import { baseProps } from "./base";
import { DEPLOY_STATE_MAP } from "./deploy";

interface CancelDeployConfiguration {
  service?: string;
  deployId?: string;
}

interface CancelDeployOutput {
  deployId?: string;
  status?: string;
  createdAt?: string;
  finishedAt?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CancelDeployConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }
  if (configuration?.deployId) {
    metadata.push({ icon: "circle-slash-2", label: `Deploy: ${configuration.deployId}` });
  }

  return metadata;
}

export const cancelDeployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node), eventStateMap: DEPLOY_STATE_MAP };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { success?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
    const result =
      (outputs?.success?.[0]?.data as CancelDeployOutput | undefined) ??
      (outputs?.failed?.[0]?.data as CancelDeployOutput | undefined);

    return {
      "Triggered At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Deploy ID": stringOrDash(result?.deployId),
      Status: stringOrDash(result?.status),
      "Finished At": formatTimestamp(result?.finishedAt),
    };
  },
};
