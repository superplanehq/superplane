import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import gcpComputeIcon from "@/assets/icons/integrations/gcp.compute.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";
import { type ImageNodeMetadata, imageNameFromValue } from "./image_helpers";

interface DeleteImageConfiguration {
  image?: string;
}

interface DeleteImageOutputData {
  imageName?: string;
}

export const deleteImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpComputeIcon,
      iconSlug: context.componentDefinition?.icon ?? "trash-2",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Delete Image",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as DeleteImageOutputData | undefined;
    if (!result) return details;

    if (result.imageName) details["Image Name"] = result.imageName;
    details["Status"] = "Deleted";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as ImageNodeMetadata | undefined;
  const configuration = node.configuration as DeleteImageConfiguration | undefined;

  const imageName = nodeMetadata?.imageName || imageNameFromValue(configuration?.image);
  if (imageName) {
    metadata.push({ icon: "trash-2", label: imageName });
  }

  return metadata;
}
