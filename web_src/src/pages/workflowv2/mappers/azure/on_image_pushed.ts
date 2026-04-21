import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type React from "react";
import type { TriggerProps } from "@/pages/workflowv2/mappers/types";
import azureIcon from "@/assets/icons/integrations/azure.svg";
import type { ACREventData } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../utils";
import { getBackgroundColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";

export interface OnImagePushedConfiguration {
  resourceGroup?: string;
  registry?: string;
  repositoryFilter?: string;
  tagFilter?: string;
}

export const onImagePushedTriggerRenderer: TriggerRenderer = {
  subtitle: (context: TriggerEventContext): string | React.ReactNode => {
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return subtitle;
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as ACREventData;
    const target = eventData?.target;

    return {
      Repository: stringOrDash(target?.repository),
      Tag: stringOrDash(target?.tag),
      Digest: stringOrDash(target?.digest),
      Actor: stringOrDash(eventData?.actor?.name),
      Timestamp: stringOrDash(eventData?.timestamp),
      Registry: stringOrDash(eventData?.request?.host),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnImagePushedConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    if (configuration?.registry) {
      metadataItems.push({ icon: "package", label: configuration.registry });
    }

    if (configuration?.repositoryFilter) {
      metadataItems.push({ icon: "funnel", label: configuration.repositoryFilter });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: azureIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const subtitle = onImagePushedTriggerRenderer.subtitle({ event: lastEvent });
      props.lastEventData = {
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
