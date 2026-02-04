import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, OutputPayload, SubtitleContext } from "../types";
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

export const updateReleaseMapper: ComponentBaseMapper = {
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
      const release = outputs.default[0].data as ReleaseOutput;
      Object.assign(details, {
        "Published At": release?.published_at ? new Date(release.published_at).toLocaleString() : "-",
        "Updated By": release?.author?.login || "-",
      });

      details["Release URL"] = release?.html_url || "";
      details["Release ID"] = release?.id?.toString() || "";
      details["Tag Name"] = release?.tag_name || "";

      if (release?.name) {
        details["Name"] = release.name;
      }

      if (release?.draft !== undefined) {
        details["Draft"] = release.draft ? "Yes" : "No";
      }

      if (release?.prerelease !== undefined) {
        details["Prerelease"] = release.prerelease ? "Yes" : "No";
      }
    }

    return details;
  },
};
