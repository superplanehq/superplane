import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";

interface OnBroadcastConfiguration {
  app?: string;
}

interface OnBroadcastMetadata {
  app?: {
    id?: string;
    name?: string;
  };
}

interface OnBroadcastEventData {
  app?: {
    id?: string;
    name?: string;
  };
  node?: {
    id?: string;
    name?: string;
  };
  payload?: unknown;
}

export const onBroadcastTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnBroadcastEventData | undefined;

    return {
      title: broadcastTitle(eventData),
      subtitle: "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnBroadcastEventData | undefined;
    const values: Record<string, string> = {};
    const appLabel = namedOrId(eventData?.app);
    const sourceNodeLabel = namedOrId(eventData?.node);

    if (appLabel) {
      values.App = appLabel;
    }

    if (sourceNodeLabel) {
      values["Source node"] = sourceNodeLabel;
    }

    if (context.event?.createdAt) {
      values["Received at"] = new Date(context.event.createdAt).toLocaleString();
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnBroadcastMetadata | undefined;
    const configuration = node.configuration as OnBroadcastConfiguration | undefined;
    const appLabel = metadata?.app?.name || configuration?.app;

    const props: TriggerProps = {
      title: node.name || definition.label || "On Broadcast",
      iconSlug: definition.icon || "rss",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: appLabel ? [{ icon: "layout-grid", label: appLabel }] : [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnBroadcastEventData | undefined;

      props.lastEventData = {
        title: broadcastTitle(eventData),
        subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function namedOrId(entity?: { name?: string; id?: string }): string | undefined {
  if (entity?.name) {
    return entity.name;
  }
  if (entity?.id) {
    return entity.id;
  }
  return undefined;
}

function broadcastTitle(eventData: OnBroadcastEventData | undefined): string {
  const appName = eventData?.app?.name?.trim();
  if (appName) {
    return `Message from ${appName}`;
  }

  return "Message received";
}
