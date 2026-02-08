import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { MetadataItem } from "@/ui/metadataList";
import { Release } from "./types";

interface ListReleasesConfiguration {
  repository?: string;
  perPage?: string;
  page?: string;
}

function getMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ListReleasesConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.perPage) {
    metadata.push({ icon: "list", label: `Per Page: ${configuration.perPage}` });
  }

  if (configuration?.page) {
    metadata.push({ icon: "hash", label: `Page: ${configuration.page}` });
  }

  return metadata;
}

export const listReleasesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const list = outputs.default[0].data as Release[] | undefined;
      details["Retrieved At"] = context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";
      details["Releases Count"] = list ? String(list.length) : "0";
    }

    return details;
  },
};
