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
  author?: {
    login?: string;
    html_url?: string;
  };
}

interface ListReleasesConfiguration {
  repository?: string;
  perPage?: number;
}

function getListReleasesMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ListReleasesConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.perPage) {
    metadata.push({ icon: "list", label: `Limit: ${configuration.perPage}` });
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

    if (outputs && outputs.default && outputs.default.length > 0) {
      const releases = outputs.default[0].data as ReleaseOutput[];
      Object.assign(details, {
        "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      });

      if (Array.isArray(releases)) {
        details["Total Releases"] = releases.length.toString();

        if (releases.length > 0) {
          const latestRelease = releases[0];
          if (latestRelease.tag_name) {
            details["Latest Tag"] = latestRelease.tag_name;
          }
          if (latestRelease.html_url) {
            details["Latest Release URL"] = latestRelease.html_url;
          }
          if (latestRelease.published_at) {
            details["Latest Published At"] = new Date(latestRelease.published_at).toLocaleString();
          }
        }

        const draftCount = releases.filter((r) => r.draft).length;
        const prereleaseCount = releases.filter((r) => r.prerelease).length;

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
