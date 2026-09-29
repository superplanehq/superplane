import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
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

    return {
      title: pipelineCompletedTitle(eventData),
      subtitle: pipelineCompletedSubtitle(pipelineStatus(eventData), context.event?.createdAt),
    };
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

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: HarnessIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: pipelineCompletedMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onPipelineCompletedTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

function pipelineCompletedTitle(eventData?: OnPipelineCompletedEventData): string {
  return "Pipeline Completed · " + (eventData?.pipelineIdentifier || "unknown");
}

function pipelineStatus(eventData?: OnPipelineCompletedEventData): string {
  return eventData?.status || "";
}

function pipelineCompletedSubtitle(status: string, createdAt?: string): string | React.ReactNode {
  if (status && createdAt) {
    return renderWithTimeAgo(status, new Date(createdAt));
  }

  return status || (createdAt ? renderTimeAgo(new Date(createdAt)) : "");
}

function pipelineCompletedMetadataItems(
  metadata?: OnPipelineCompletedMetadata,
  configuration?: { pipelineIdentifier?: string },
) {
  const pipelineLabel = metadata?.pipelineIdentifier || configuration?.pipelineIdentifier;
  if (!pipelineLabel) {
    return [];
  }

  return [{ icon: "workflow", label: pipelineLabel }];
}
