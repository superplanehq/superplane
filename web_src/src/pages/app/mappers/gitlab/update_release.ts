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
import type { GitLabNodeMetadata, Release } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface UpdateReleaseConfiguration {
  project?: string;
  releaseStrategy?: string;
  tagName?: string;
}

export const updateReleaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as UpdateReleaseConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.releaseStrategy === "latest") {
      metadataItems.push({ icon: "tag", label: "Latest release" });
    } else if (configuration.tagName) {
      metadataItems.push({ icon: "tag", label: configuration.tagName });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const release = outputs?.default?.[0]?.data as Release | undefined;
    if (release) {
      return release.name || release.tag_name;
    }
    return buildGitlabExecutionSubtitle(context.execution, "Release Updated");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const release = outputs.default[0].data as Release;
    if (!release) {
      return details;
    }

    details["Tag"] = release.tag_name || "-";
    details["Name"] = release.name || "-";
    details["Released At"] = release.released_at ? new Date(release.released_at).toLocaleString() : "-";

    if (release.milestones && release.milestones.length > 0) {
      details["Milestones"] = release.milestones.map((m) => m.title).join(", ");
    }

    if (release._links?.self) {
      details["URL"] = release._links.self;
    }

    return details;
  },
};
