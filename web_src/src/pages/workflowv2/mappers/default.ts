import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";

/**
 * Default renderer for trigger types that don't have a specific renderer.
 * Uses basic icon/color configuration from the trigger metadata.
 */
export const defaultTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    return { title: `Event received at ${new Date(context.event.createdAt!).toLocaleString()}`, subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSlug: definition.icon || "bolt",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
    };

    if (lastEvent) {
      props.lastEventData = {
        title: "Event emitted by trigger",
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
