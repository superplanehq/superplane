import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import azureIcon from "@/assets/icons/integrations/azure.svg";
import { ACREventData } from "./types";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../utils";
import { getBackgroundColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";

export interface OnImagePushedConfiguration {
  resourceGroup?: string;
  registry?: string;
  repositoryFilter?: string;
  tagFilter?: string;
}

export const onImagePushedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as ACREventData;
    const repository = eventData?.target?.repository;
    const tag = eventData?.target?.tag;

    const title = repository ? `${repository}${tag ? `:${tag}` : ""}` : "Image pushed";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
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
      const { title, subtitle } = onImagePushedTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
