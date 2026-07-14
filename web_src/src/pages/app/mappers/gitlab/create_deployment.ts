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

interface CreateDeploymentConfiguration {
  project?: string;
  environment?: string;
  ref?: string;
}

export const createDeploymentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as CreateDeploymentConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.environment) {
      metadataItems.push({ icon: "rocket", label: configuration.environment });
    }

    if (configuration.ref) {
      metadataItems.push({ icon: "git-branch", label: configuration.ref });
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

    details["Created At"] = deployment.created_at ? new Date(deployment.created_at).toLocaleString() : "-";
    details["Status"] = deployment.status || "-";

    if (deployment.environment?.name) {
      details["Environment"] = deployment.environment.name;
    }

    if (deployment.ref) {
      details["Ref"] = deployment.ref;
    }

    if (deployment.sha) {
      details["SHA"] = deployment.sha;
    }

    if (deployment.environment?.external_url) {
      details["URL"] = deployment.environment.external_url;
    }

    return details;
  },
};
