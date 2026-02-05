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

interface ReleaseOutput {
  id?: number;
  tag_name?: string;
  name?: string;
  html_url?: string;
  draft?: boolean;
  prerelease?: boolean;
  created_at?: string;
  published_at?: string;
  tarball_url?: string;
  zipball_url?: string;
  author?: {
    login?: string;
    html_url?: string;
  };
  assets?: Array<{
    name?: string;
    size?: number;
    download_count?: number;
  }>;
}

interface ListReleasesConfiguration {
  repository?: string;
  perPage?: number;
  page?: number;
}

function getListReleasesMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ListReleasesConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.perPage && configuration.perPage !== 30) {
    metadata.push({ icon: "list", label: `${configuration.perPage} per page` });
  }

  if (configuration?.page && configuration.page > 1) {
    metadata.push({ icon: "file", label: `Page ${configuration.page}` });
  }

  return metadata;
}

export const listReleasesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getListReleasesMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    });

    if (outputs && outputs.default && outputs.default.length > 0) {
      const releases = outputs.default.map((o) => o.data as ReleaseOutput);

      details["Releases Found"] = releases.length.toString();

      if (releases.length > 0) {
        // Show info about the first release
        const firstRelease = releases[0];
        if (firstRelease?.tag_name) {
          details["Latest Tag"] = firstRelease.tag_name;
        }
        if (firstRelease?.name) {
          details["Latest Name"] = firstRelease.name;
        }
        if (firstRelease?.html_url) {
          details["Latest URL"] = firstRelease.html_url;
        }
        if (firstRelease?.published_at) {
          details["Latest Published"] = new Date(firstRelease.published_at).toLocaleString();
        }

        // Count drafts and prereleases
        const draftCount = releases.filter((r) => r?.draft).length;
        const prereleaseCount = releases.filter((r) => r?.prerelease).length;

        if (draftCount > 0) {
          details["Drafts"] = draftCount.toString();
        }
        if (prereleaseCount > 0) {
          details["Prereleases"] = prereleaseCount.toString();
        }
      }
    }

    return details;
  },
};
