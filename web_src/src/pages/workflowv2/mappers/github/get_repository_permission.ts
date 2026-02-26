import { ComponentBaseProps } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

interface Output {
  permission?: string;
  role_name?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
}

interface Configuration {
  username?: string;
}

function getRepositoryPermissionMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;
  const configuration = node.configuration as Configuration | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.username) {
    metadata.push({ icon: "user", label: configuration.username });
  }

  return metadata;
}

export const getRepositoryPermissionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getRepositoryPermissionMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;
    const details: Record<string, string> = {};

    if (output?.permission) {
      details["Permission"] = output.permission;
    }

    if (output?.role_name) {
      details["Role"] = output.role_name;
    }

    if (output?.user?.login) {
      details["User"] = output.user.login;
    }

    if (output?.user?.html_url) {
      details["URL"] = output.user.html_url;
    }

    return details;
  },
};
