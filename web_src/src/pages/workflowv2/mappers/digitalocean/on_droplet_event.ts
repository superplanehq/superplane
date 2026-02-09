import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";

interface OnDropletEventData {
  action?: {
    id: number;
    status: string;
    type: string;
    started_at: string;
    completed_at: string;
    resource_id: number;
    resource_type: string;
    region_slug: string;
  };
}

export const onDropletEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as OnDropletEventData;
    const action = eventData?.action;
    const title = `${action?.type || ""} - Droplet ${action?.resource_id || ""}`;
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as OnDropletEventData;
    const action = eventData?.action;

    return {
      "Action Type": action?.type || "-",
      "Resource ID": action?.resource_id?.toString() || "-",
      Status: action?.status || "-",
      Region: action?.region_slug || "-",
      "Started At": action?.started_at || "-",
      "Completed At": action?.completed_at || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (configuration?.events) {
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${configuration.events.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnDropletEventData;
      const action = eventData?.action;

      props.lastEventData = {
        title: `${action?.type || ""} - Droplet ${action?.resource_id || ""}`,
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
