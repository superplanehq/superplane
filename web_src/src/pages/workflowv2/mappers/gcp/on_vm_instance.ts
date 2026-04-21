import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/pages/workflowv2/mappers/types";
import { flattenObject } from "@/lib/utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";

export const onVMInstanceTriggerRenderer: TriggerRenderer = {
  subtitle: (context: TriggerEventContext): string | React.ReactNode => {
    const data = context.event?.data as { resourceName?: string } | undefined;
    const resourceName = data?.resourceName ?? "";
    const subtitle = resourceName || "";
    return subtitle;
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    return {
      title: node.name || definition.label || "On VM Instance",
      iconSrc: gcpIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          subtitle:
            (lastEvent.data as { resourceName?: string })?.resourceName ?? renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
