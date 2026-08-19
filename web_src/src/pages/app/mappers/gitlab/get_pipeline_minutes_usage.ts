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
import type { PipelineMinutesUsage } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface GetPipelineMinutesUsageConfiguration {
  projects?: string[];
}

export const getPipelineMinutesUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as GetPipelineMinutesUsageConfiguration | undefined) ?? {};
    const metadataItems: MetadataItem[] = [];

    if (configuration.projects && configuration.projects.length > 0) {
      metadataItems.push({ icon: "book", label: `${configuration.projects.length} project(s)` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const usage = outputs?.default?.[0]?.data as PipelineMinutesUsage | undefined;
    if (usage) {
      return `${usage.minutes} min (${usage.month})`;
    }
    return buildGitlabExecutionSubtitle(context.execution, "Pipeline Minutes Usage Retrieved");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const usage = outputs.default[0].data as PipelineMinutesUsage;
    if (!usage) {
      return details;
    }

    details["Month"] = usage.month || "-";
    details["Minutes Used"] = typeof usage.minutes === "number" ? String(usage.minutes) : "-";
    details["Shared Runners Duration"] =
      typeof usage.sharedRunnersDuration === "number" ? `${usage.sharedRunnersDuration}s` : "-";

    if (usage.projects && usage.projects.length > 0) {
      details["Projects"] = usage.projects
        .map((p) => p.project?.name)
        .filter(Boolean)
        .join(", ");
    }

    return details;
  },
};
