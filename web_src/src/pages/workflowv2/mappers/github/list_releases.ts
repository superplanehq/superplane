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

interface ReleaseItem {
  id?: number;
  tag_name?: string;
  name?: string;
  html_url?: string;
  body?: string;
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
    browser_download_url?: string;
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

  if (configuration?.perPage) {
    metadata.push({ icon: "list", label: `${configuration.perPage} per page` });
  }

  if (configuration?.page && configuration.page > 1) {
    metadata.push({ icon: "hash", label: `Page ${configuration.page}` });
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
      const releases = outputs.default.map(o => o.data) as ReleaseItem[];

      Object.assign(details, {
        "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
        "Total Releases": releases.length.toString(),
      });

      // Show summary of first few releases
      const preview = releases.slice(0, 5);
      preview.forEach((release, index) => {
        const tag = release?.tag_name || "unknown";
        const name = release?.name || tag;
        details[`Release ${index + 1}`] = `${name} (${tag})`;
      });

      if (releases.length > 5) {
        details["...and more"] = `${releases.length - 5} additional releases`;
      }

      // Show latest release details
      if (releases.length > 0 && releases[0]) {
        const latest = releases[0];
        if (latest.published_at) {
          details["Latest Published"] = new Date(latest.published_at).toLocaleString();
        }
        if (latest.author?.login) {
          details["Latest Author"] = latest.author.html_url || latest.author.login;
        }
        if (latest.assets && latest.assets.length > 0) {
          details["Latest Assets"] = latest.assets.length.toString();
        }
      }
    }

    return details;
  },
};
