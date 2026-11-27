import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

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
  };
}

function getRequiredReleaseDetails(release: ReleaseOutput): Record<string, string> {
  return {
    "Published At": release.published_at ? new Date(release.published_at).toLocaleString() : "-",
    "Updated By": release.author?.login || "-",
    "Release URL": release.html_url || "",
    "Release ID": release.id?.toString() || "",
    "Tag Name": release.tag_name || "",
  };
}

function getOptionalReleaseDetails(release: ReleaseOutput): Record<string, string> {
  const details: Record<string, string> = {};

  if (release.name) {
    details["Name"] = release.name;
  }

  if (release.draft !== undefined) {
    details["Draft"] = release.draft ? "Yes" : "No";
  }

  if (release.prerelease !== undefined) {
    details["Prerelease"] = release.prerelease ? "Yes" : "No";
  }

  return details;
}

export const updateReleaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return {};
    }

    const release = (outputs.default[0].data ?? {}) as ReleaseOutput;
    return {
      ...getRequiredReleaseDetails(release),
      ...getOptionalReleaseDetails(release),
    };
  },
};
