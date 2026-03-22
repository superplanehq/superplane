import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import React from "react";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";

export const onVMInstanceTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const data = context.event?.data as { resourceName?: string } | undefined;
    const resourceName = data?.resourceName ?? "";
    const title = "VM instance event";
    const subtitle = resourceName || "";
    return { title, subtitle };
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
          title: "VM instance event",
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
