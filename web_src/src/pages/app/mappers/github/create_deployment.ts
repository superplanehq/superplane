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

interface Deployment {
  ref?: string;
  environment?: string;
  description?: string;
  created_at?: string;
  url?: string;
}

export const createDeploymentMapper: ComponentBaseMapper = {
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
      const deployment = outputs.default[0].data as Deployment;
      Object.assign(details, {
        "Created At": deployment.created_at ? new Date(deployment.created_at).toLocaleString() : "-",
      });

      details["Deployment URL"] = deployment.url || "-";
      details["Environment"] = deployment.environment || "-";
      details["Ref"] = deployment.ref || "-";
      details["Description"] = deployment.description || "-";
    }

    return details;
  },
};
