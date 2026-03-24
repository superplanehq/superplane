import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "./types";
import type { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { renderTimeAgo } from "@/components/TimeAgo";

/**
 * Default renderer for trigger types that don't have a specific renderer.
 * Uses basic icon/color configuration from the trigger metadata.
 */
export const defaultTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    return { title: `Event received at ${new Date(context.event?.createdAt || "").toLocaleString()}`, subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
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
        subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
