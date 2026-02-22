import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import HarnessIcon from "@/assets/icons/integrations/harness.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnPipelineCompletedMetadata {
  pipelineIdentifier?: string;
}

interface OnPipelineCompletedEventData {
  executionId?: string;
  pipelineIdentifier?: string;
  status?: string;
  eventType?: string;
}

export const onPipelineCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPipelineCompletedEventData;
    const title = "Pipeline Completed · " + (eventData?.pipelineIdentifier || "unknown");
    const status = eventData?.status || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    const subtitle = status && timeAgo ? `${status} · ${timeAgo}` : status || timeAgo;

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPipelineCompletedEventData;

    return {
      Pipeline: eventData?.pipelineIdentifier || "",
      "Execution ID": eventData?.executionId || "",
      Status: eventData?.status || "",
      "Event Type": eventData?.eventType || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnPipelineCompletedMetadata;
    const configuration = node.configuration as { pipelineIdentifier?: string };
    const metadataItems: TriggerProps["metadata"] = [];

    const pipelineLabel = metadata?.pipelineIdentifier || configuration?.pipelineIdentifier;
    if (pipelineLabel) {
      metadataItems.push({ icon: "workflow", label: pipelineLabel });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: HarnessIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnPipelineCompletedEventData;
      const title = "Pipeline Completed · " + (eventData?.pipelineIdentifier || "unknown");
      const status = eventData?.status || "";
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = status && timeAgo ? `${status} · ${timeAgo}` : status || timeAgo;

      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
