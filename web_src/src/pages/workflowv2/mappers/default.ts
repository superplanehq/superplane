import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";

/**
 * Default renderer for trigger types that don't have a specific renderer.
 * Uses basic icon/color configuration from the trigger metadata.
 */
export const defaultTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    return { title: `Event received at ${new Date(event.createdAt!).toLocaleString()}`, subtitle: "" };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, string> => {
    return flattenObject(event.data || {});
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const props: TriggerProps = {
      title: node.name!,
      iconSlug: trigger.icon || "bolt",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: [],
    };

    if (lastEvent) {
      props.lastEventData = {
        title: "Event emitted by trigger",
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt!)),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "processed",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
