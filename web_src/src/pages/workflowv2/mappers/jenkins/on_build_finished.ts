import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import jenkinsIcon from "@/assets/icons/integrations/jenkins.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnBuildFinishedEventData {
  job?: {
    name: string;
    url: string;
  };
  build?: {
    number: number;
    url: string;
    result: string;
  };
}

interface OnBuildFinishedMetadata {
  job?: {
    name: string;
    url: string;
  };
}

export const onBuildFinishedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnBuildFinishedEventData;
    const jobName = eventData?.job?.name || "Build";
    const result = eventData?.build?.result || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

    return {
      title: jobName,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnBuildFinishedEventData;

    return {
      Job: eventData?.job?.name || "",
      "Build Number": eventData?.build?.number?.toString() || "",
      Result: eventData?.build?.result || "",
      "Build URL": eventData?.build?.url || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnBuildFinishedMetadata;
    const configuration = node.configuration as any;
    const metadataItems = [];

    const jobLabel = metadata?.job?.name || configuration?.job;
    if (jobLabel) {
      metadataItems.push({
        icon: "folder",
        label: jobLabel,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jenkinsIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnBuildFinishedEventData;
      const jobName = eventData?.job?.name || "Build";
      const result = eventData?.build?.result || "";
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

      props.lastEventData = {
        title: jobName,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
