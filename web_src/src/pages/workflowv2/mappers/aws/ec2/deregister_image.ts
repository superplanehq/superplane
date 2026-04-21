import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/pages/workflowv2/mappers/types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";

interface Configuration {
  region?: string;
  imageId?: string;
  deleteSnapshots?: boolean;
}

interface Output {
  requestId?: string;
  image?: {
    imageId?: string;
  };
  deletedSnapshots?: string[];
}

export const deregisterImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";
    const configuration = context.node.configuration as Configuration | undefined;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? deregisterImageEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: deregisterImageMetadata(configuration),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;

    if (!output) {
      return {};
    }

    const details: Record<string, string> = {
      "Request ID": stringOrDash(output.requestId),
      "Image ID": stringOrDash(output.image?.imageId),
    };

    if (output.deletedSnapshots && output.deletedSnapshots.length > 0) {
      details["Deleted Snapshots"] = output.deletedSnapshots.join(", ");
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function deregisterImageMetadata(configuration?: Configuration): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.imageId) {
    items.push({ icon: "disc", label: configuration.imageId });
  }

  if (configuration?.deleteSnapshots) {
    items.push({ icon: "trash", label: "delete snapshots" });
  }

  return items;
}

function deregisterImageEventSections(
  _nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id!,
    },
  ];
}
