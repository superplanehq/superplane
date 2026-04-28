import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import ociIcon from "@/assets/icons/integrations/oci.svg";

interface OciFunctionInvokeEvent {
  eventType?: string;
  eventTime?: string;
  data?: {
    resourceName?: string;
    resourceId?: string;
    compartmentId?: string;
    compartmentName?: string;
    availabilityDomain?: string;
  };
}

function getEventEnvelope(event: TriggerEventContext["event"]): OciFunctionInvokeEvent | undefined {
  return event?.data as OciFunctionInvokeEvent | undefined;
}

function getFunctionName(event: TriggerEventContext["event"]): string {
  const envelope = getEventEnvelope(event);
  return envelope?.data?.resourceName ?? "";
}

export const onFunctionInvokeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const name = getFunctionName(context.event);
    return {
      title: "Function invoked",
      subtitle: name || "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = getEventEnvelope(context.event);
    const data = envelope?.data;

    const triggeredAtRaw = context.event?.createdAt ?? envelope?.eventTime;
    const triggeredAt = triggeredAtRaw ? new Date(triggeredAtRaw).toLocaleString() : undefined;

    const rawEntries: [string, string | undefined][] = [
      ["Triggered At", triggeredAt],
      ["Function Name", data?.resourceName],
      ["Compartment", data?.compartmentName],
    ];

    return Object.fromEntries(rawEntries.filter((e): e is [string, string] => e[1] != null));
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const functionName = lastEvent ? getFunctionName(lastEvent) : "";

    return {
      title: node.name || definition.label || "On Function Invoked",
      iconSrc: ociIcon,
      iconSlug: definition.icon || "oci",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          title: functionName || "Function invoked",
          subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
