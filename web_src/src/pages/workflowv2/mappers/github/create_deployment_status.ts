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

interface DeploymentStatus {
  state?: string;
  description?: string;
  environment?: string;
  created_at?: string;
  url?: string;
}

export const createDeploymentStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs?.default && outputs.default.length > 0) {
      const status = outputs.default[0].data as DeploymentStatus;
      Object.assign(details, {
        "Created At": status.created_at ? new Date(status.created_at).toLocaleString() : "-",
      });

      details["Deployment Status URL"] = status.url || "-";
      details["State"] = status.state || "-";
      details["Environment"] = status.environment || "-";
      details["Description"] = status.description || "-";
    }

    return details;
  },
};
