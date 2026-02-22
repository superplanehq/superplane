import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { AmiStateChangeEvent } from "./types";

interface Configuration {
  region?: string;
  states?: string[];
}

function buildMetadata(configuration?: Configuration): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.states) {
    items.push({ icon: "tag", label: configuration.states.join(", ") });
  }

  return items;
}

export const onImageTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as AmiStateChangeEvent;
    const imageId = eventData?.detail?.ImageId;
    const state = eventData?.detail?.State || "";
    const title = imageId || "EC2 AMI state change";
    const subtitle = `${state} · ${formatTimeAgo(new Date(context.event?.createdAt || ""))}`;
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as AmiStateChangeEvent;

    let details: Record<string, string> = {
      "Image ID": stringOrDash(eventData?.detail?.ImageId),
      State: stringOrDash(eventData?.detail?.State),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };

    if (eventData?.detail?.ErrorMessage) {
      details["Error Message"] = stringOrDash(eventData?.detail?.ErrorMessage);
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as Configuration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsEc2Icon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadata(configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onImageTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
