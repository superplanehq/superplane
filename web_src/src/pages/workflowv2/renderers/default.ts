import { ComponentsNode, TriggersTrigger } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "./types";

/**
 * Default renderer for trigger types that don't have a specific renderer.
 * Uses basic icon/color configuration from the trigger metadata.
 */
export const defaultTriggerRenderer: TriggerRenderer = {
  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger) => ({
    title: node.name!,
    iconSlug: trigger.icon || "bolt",
    iconColor: getColorClass(trigger.color),
    headerColor: getBackgroundColorClass(trigger.color),
    collapsedBackground: getBackgroundColorClass(trigger.color),
    metadata: [],
    zeroStateText: "No events yet",
  }),
};
