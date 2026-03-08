import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";

export const onObjectFinalizedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const data = context.event?.data as { resourceName?: string } | undefined;
    const resourceName = data?.resourceName ?? "";
    const title = "Object finalized";
    const subtitle = extractObjectPath(resourceName);
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    return {
      title: node.name || definition.label || "On Object Finalized",
      iconSrc: gcpIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          title: "Object finalized",
          subtitle:
            extractObjectPath((lastEvent.data as { resourceName?: string })?.resourceName) ||
            formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};

function extractObjectPath(resourceName?: string): string {
  if (!resourceName) return "";
  const objectsIndex = resourceName.indexOf("/objects/");
  if (objectsIndex === -1) return resourceName;
  return resourceName.slice(objectsIndex + "/objects/".length);
}
