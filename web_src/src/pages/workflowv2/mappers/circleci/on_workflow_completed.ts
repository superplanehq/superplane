import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import CircleCILogo from "@/assets/icons/integrations/circleci.svg";
import { formatTimeAgo } from "@/utils/date";

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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnWorkflowCompletedEventData;
    const workflowName = eventData?.workflow?.name || "Workflow";
    const status = eventData?.workflow?.status || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = status && timeAgo ? `${status} · ${timeAgo}` : status || timeAgo;

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
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = status && timeAgo ? `${status} · ${timeAgo}` : status || timeAgo;

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
