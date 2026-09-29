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

interface DeletedReleaseOutput {
  id?: number;
  tag_name?: string;
  name?: string;
  html_url?: string;
  draft?: boolean;
  prerelease?: boolean;
  deleted_at?: string;
  tag_deleted?: boolean;
}

export const deleteReleaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs?.default?.length) {
      return {};
    }

    return detailsFromDeletedRelease(outputs.default[0].data as DeletedReleaseOutput);
  },
};

function detailsFromDeletedRelease(deletedRelease: DeletedReleaseOutput): Record<string, string> {
  const details = alwaysSetDeletedReleaseDetails(deletedRelease);

  if (deletedRelease?.name) {
    details["Release Name"] = deletedRelease.name;
  }

  if (deletedRelease?.draft) {
    details["Was Draft"] = "Yes";
  }

  if (deletedRelease?.prerelease) {
    details["Was Prerelease"] = "Yes";
  }

  return details;
}

function alwaysSetDeletedReleaseDetails(deletedRelease: DeletedReleaseOutput): Record<string, string> {
  return {
    "Deleted At": deletedRelease?.deleted_at ? new Date(deletedRelease.deleted_at).toLocaleString() : "-",
    "Tag Deleted": deletedRelease?.tag_deleted ? "Yes" : "No",
    "Release ID": deletedRelease?.id?.toString() || "",
    "Tag Name": deletedRelease?.tag_name || "",
  };
}
