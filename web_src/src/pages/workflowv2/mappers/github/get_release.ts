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
  assets?: Array<{
    name?: string;
    size?: number;
    download_count?: number;
  }>;
}

interface GetReleaseConfiguration {
  repository?: string;
  releaseStrategy?: string;
  tagName?: string;
  releaseId?: string;
}

function getReleaseMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetReleaseConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.releaseStrategy) {
    switch (configuration.releaseStrategy) {
      case "latest":
        metadata.push({ icon: "tag", label: "Latest release" });
        break;
      case "latestDraft":
        metadata.push({ icon: "tag", label: "Latest draft" });
        break;
      case "latestPrerelease":
        metadata.push({ icon: "tag", label: "Latest prerelease" });
        break;
      case "specific":
        if (configuration.tagName) {
          metadata.push({ icon: "tag", label: `Tag: ${configuration.tagName}` });
        }
        break;
      case "byId":
        if (configuration.releaseId) {
          metadata.push({ icon: "hash", label: `ID: ${configuration.releaseId}` });
        }
        break;
    }
  }

  return metadata;
}

export const getReleaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getReleaseMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const release = outputs.default[0].data as ReleaseOutput;
      Object.assign(details, {
        "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      });

      details["Release URL"] = release?.html_url || "";
      details["Tag Name"] = release?.tag_name || "";

      if (release?.name) {
        details["Release Name"] = release.name;
      }

      if (release?.author?.login) {
        details["Author"] = release.author.html_url || release.author.login;
      }

      if (release?.draft !== undefined) {
        details["Draft"] = release.draft ? "Yes" : "No";
      }

      if (release?.prerelease !== undefined) {
        details["Prerelease"] = release.prerelease ? "Yes" : "No";
      }

      if (release?.published_at) {
        details["Published At"] = new Date(release.published_at).toLocaleString();
      }

      if (release?.assets && release.assets.length > 0) {
        details["Assets"] = release.assets.length.toString();
      }
    }

    return details;
  },
};
