import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import CircleCILogo from "@/assets/icons/integrations/circleci.svg";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";

interface OnWorkflowCompletedMetadata {
  project?: {
    name: string;
    slug: string;
  };
}

interface OnWorkflowCompletedEventData {
  workflow?: {
    id: string;
    name: string;
    status: string;
    url: string;
  };
  pipeline?: {
    id: string;
    number: number;
  };
  project?: {
    name: string;
    slug: string;
  };
  organization?: {
    name: string;
  };
}

export const onWorkflowCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as OnWorkflowCompletedEventData;
    const workflowName = eventData?.workflow?.name || "Workflow";
    const status = eventData?.workflow?.status || "";
    const subtitle =
      status && context.event?.createdAt
        ? renderWithTimeAgo(status, new Date(context.event.createdAt))
        : status || (context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "");

    return {
      title: workflowName,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnWorkflowCompletedEventData;
    const workflowUrl = eventData?.workflow?.url || "";

    return {
      Workflow: eventData?.workflow?.name || "",
      Status: eventData?.workflow?.status || "",
      "Workflow URL": workflowUrl,
      "Pipeline Number": eventData?.pipeline?.number?.toString() || "",
      Project: eventData?.project?.name || "",
      Organization: eventData?.organization?.name || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnWorkflowCompletedMetadata;
    const configuration = node.configuration as any;
    const metadataItems = [];

    const projectLabel = metadata?.project?.name || metadata?.project?.slug || configuration?.projectSlug;
    if (projectLabel) {
      metadataItems.push({
        icon: "folder",
        label: projectLabel,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: CircleCILogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnWorkflowCompletedEventData;
      const workflowName = eventData?.workflow?.name || "Workflow";
      const status = eventData?.workflow?.status || "";
      const subtitle =
        status && lastEvent.createdAt
          ? renderWithTimeAgo(status, new Date(lastEvent.createdAt))
          : status || (lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "");

      props.lastEventData = {
        title: workflowName,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
