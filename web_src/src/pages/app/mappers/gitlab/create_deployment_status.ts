import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import type { Deployment, GitLabNodeMetadata } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface CreateDeploymentStatusConfiguration {
  project?: string;
  deploymentId?: string;
  status?: string;
}

export const createDeploymentStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as CreateDeploymentStatusConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.deploymentId) {
      metadataItems.push({ icon: "rocket", label: `Deployment #${configuration.deploymentId}` });
    }

    if (configuration.status) {
      metadataItems.push({ icon: "activity", label: configuration.status });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const deployment = outputs?.default?.[0]?.data as Deployment | undefined;
    return buildGitlabExecutionSubtitle(context.execution, deployment?.status);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const deployment = outputs.default[0].data as Deployment;
    if (!deployment) {
      return details;
    }

    const updatedAt = deployment.updated_at || deployment.created_at;
    details["Updated At"] = updatedAt ? new Date(updatedAt).toLocaleString() : "-";
    details["Status"] = deployment.status || "-";

    if (deployment.environment?.name) {
      details["Environment"] = deployment.environment.name;
    }

    if (deployment.id) {
      details["Deployment ID"] = deployment.id.toString();
    }

    if (deployment.ref) {
      details["Ref"] = deployment.ref;
    }

    if (deployment.environment?.external_url) {
      details["URL"] = deployment.environment.external_url;
    }

    return details;
  },
};
