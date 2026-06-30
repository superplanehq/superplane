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

interface UpdateImageConfiguration {
  image?: string;
  deprecationState?: string;
}

interface UpdateImageOutputData {
  name?: string;
  family?: string;
  status?: string;
  deprecationState?: string;
  replacement?: string;
}

const deprecationLabels: Record<string, string> = {
  ACTIVE: "Active",
  DEPRECATED: "Deprecated",
  OBSOLETE: "Obsolete",
  DELETED: "Deleted",
};

export const updateImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpComputeIcon,
      iconSlug: context.componentDefinition?.icon ?? "image",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Update Image",
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
    const result = outputs?.default?.[0]?.data as UpdateImageOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Image Name"] = result.name;
    if (result.family) details["Family"] = result.family;
    if (result.deprecationState) {
      details["Deprecation State"] = deprecationLabels[result.deprecationState] || result.deprecationState;
    }
    if (result.replacement) details["Replacement"] = result.replacement;

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
  const configuration = node.configuration as UpdateImageConfiguration | undefined;

  const imageName = nodeMetadata?.imageName || imageNameFromValue(configuration?.image);
  if (imageName) {
    metadata.push({ icon: "image", label: imageName });
  }
  if (configuration?.deprecationState && configuration.deprecationState !== "NO_CHANGE") {
    metadata.push({
      icon: "archive",
      label: deprecationLabels[configuration.deprecationState] || configuration.deprecationState,
    });
  }

  return metadata;
}
