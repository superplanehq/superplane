import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import HarnessIcon from "@/assets/icons/integrations/harness.svg";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";

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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as OnPipelineCompletedEventData;
    const title = "Pipeline Completed · " + (eventData?.pipelineIdentifier || "unknown");
    const status = eventData?.status || "";
    const subtitle =
      status && context.event?.createdAt
        ? renderWithTimeAgo(status, new Date(context.event.createdAt))
        : status || (context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "");

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
      const subtitle =
        status && lastEvent.createdAt
          ? renderWithTimeAgo(status, new Date(lastEvent.createdAt))
          : status || (lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "");

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
