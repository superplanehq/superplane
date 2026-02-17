import { ComponentBaseProps } from "@/ui/componentBase";
import {
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
  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const deletedRelease = outputs.default[0].data as DeletedReleaseOutput;
      Object.assign(details, {
        "Deleted At": deletedRelease?.deleted_at ? new Date(deletedRelease.deleted_at).toLocaleString() : "-",
        "Tag Deleted": deletedRelease?.tag_deleted ? "Yes" : "No",
      });

      details["Release ID"] = deletedRelease?.id?.toString() || "";
      details["Tag Name"] = deletedRelease?.tag_name || "";

      if (deletedRelease?.name) {
        details["Release Name"] = deletedRelease.name;
      }

      if (deletedRelease?.draft) {
        details["Was Draft"] = "Yes";
      }

      if (deletedRelease?.prerelease) {
        details["Was Prerelease"] = "Yes";
      }
    }

    return details;
  },
};
