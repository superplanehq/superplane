import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../utils";
import { baseProps } from "./base";

interface GetBuildConfiguration {
  jobName?: string;
  buildNumber?: number;
}

interface GetBuildOutput {
  building?: boolean;
  result?: string | null;
  number?: number;
  url?: string;
  durationMs?: number;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetBuildConfiguration | undefined;

  if (configuration?.jobName) {
    metadata.push({ icon: "hammer", label: configuration.jobName });
  }

  if (configuration?.buildNumber !== undefined) {
    metadata.push({ icon: "hash", label: `Build: ${configuration.buildNumber}` });
  }

  return metadata;
}

export const getBuildMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetBuildOutput | undefined;

    return {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Build Number": stringOrDash(result?.number),
      Building: result?.building === undefined ? "-" : result.building ? "Yes" : "No",
      Result: stringOrDash(result?.result),
      "Duration (ms)": stringOrDash(result?.durationMs),
      URL: stringOrDash(result?.url),
    };
  },
};