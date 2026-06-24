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
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";
import type { ImageNodeMetadata } from "./image_helpers";
import computeIcon from "@/assets/icons/integrations/gcp.compute.svg";

interface CreateImageConfiguration {
  name?: string;
  sourceType?: string;
  sourceDisk?: string;
  sourceSnapshot?: string;
  sourceImage?: string;
  family?: string;
}

interface CreateImageOutputData {
  name?: string;
  family?: string;
  status?: string;
  diskSizeGb?: number;
  sourceDisk?: string;
  deprecationState?: string;
}

const sourceLabels: Record<string, string> = {
  disk: "Disk",
  snapshot: "Snapshot",
  image: "Image",
};

export const createImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: computeIcon,
      iconSlug: context.componentDefinition?.icon ?? "image",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Create Image",
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
    const result = outputs?.default?.[0]?.data as CreateImageOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Image Name"] = result.name;
    if (result.family) details["Family"] = result.family;
    if (result.status) details["Status"] = result.status;
    if (result.diskSizeGb !== undefined) details["Disk Size"] = `${result.diskSizeGb} GB`;
    if (result.sourceDisk) details["Source Disk"] = result.sourceDisk;

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
  const configuration = node.configuration as CreateImageConfiguration | undefined;

  const imageName = nodeMetadata?.imageName || configuration?.name;
  if (imageName) {
    metadata.push({ icon: "image", label: imageName });
  }
  if (configuration?.sourceType) {
    metadata.push({
      icon: "git-branch",
      label: `Source: ${sourceLabels[configuration.sourceType] || configuration.sourceType}`,
    });
  }
  if (configuration?.family) {
    metadata.push({ icon: "layers", label: configuration.family });
  }

  return metadata;
}
