import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
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

interface OnWorkflowCompletedConfiguration {
  projectSlug?: string;
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

    return {
      title: workflowName(eventData),
      subtitle: workflowSubtitle(workflowStatus(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return workflowCompletedValues(context.event?.data as OnWorkflowCompletedEventData);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnWorkflowCompletedMetadata;
    const configuration = node.configuration as OnWorkflowCompletedConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: CircleCILogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: workflowCompletedMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onWorkflowCompletedTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

function workflowCompletedValues(eventData?: OnWorkflowCompletedEventData): Record<string, string> {
  return {
    Workflow: emptyIfMissing(eventData?.workflow?.name),
    Status: emptyIfMissing(eventData?.workflow?.status),
    "Workflow URL": emptyIfMissing(eventData?.workflow?.url),
    "Pipeline Number": emptyIfMissing(eventData?.pipeline?.number),
    Project: emptyIfMissing(eventData?.project?.name),
    Organization: emptyIfMissing(eventData?.organization?.name),
  };
}

function emptyIfMissing(value: string | number | undefined): string {
  if (value == null) {
    return "";
  }

  return String(value);
}

function workflowName(eventData?: OnWorkflowCompletedEventData): string {
  return eventData?.workflow?.name || "Workflow";
}

function workflowStatus(eventData?: OnWorkflowCompletedEventData): string {
  return eventData?.workflow?.status || "";
}

function workflowSubtitle(status: string, createdAt?: string): string | React.ReactNode {
  if (status && createdAt) {
    return renderWithTimeAgo(status, new Date(createdAt));
  }

  return status || (createdAt ? renderTimeAgo(new Date(createdAt)) : "");
}

function workflowCompletedMetadataItems(
  metadata?: OnWorkflowCompletedMetadata,
  configuration?: OnWorkflowCompletedConfiguration,
) {
  const projectLabel = metadata?.project?.name || metadata?.project?.slug || configuration?.projectSlug;
  if (!projectLabel) {
    return [];
  }

  return [
    {
      icon: "folder",
      label: projectLabel,
    },
  ];
}
