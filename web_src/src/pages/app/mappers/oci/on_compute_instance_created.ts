import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import ociIcon from "@/assets/icons/integrations/oci.svg";
import { compactDetails } from "./base";

interface OciComputeLaunchEvent {
  eventType?: string;
  eventTime?: string;
  data?: {
    resourceName?: string;
    resourceId?: string;
    compartmentId?: string;
    compartmentName?: string;
    availabilityDomain?: string;
    additionalDetails?: {
      shape?: string;
      imageId?: string;
    };
  };
}

function getEventEnvelope(event: TriggerEventContext["event"]): OciComputeLaunchEvent | undefined {
  return event?.data as OciComputeLaunchEvent | undefined;
}

function getInstanceName(event: TriggerEventContext["event"]): string {
  const envelope = getEventEnvelope(event);
  return envelope?.data?.resourceName ?? "";
}

export const onComputeInstanceCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const name = getInstanceName(context.event);
    return {
      title: "Compute instance created",
      subtitle: name || "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = getEventEnvelope(context.event);
    const data = envelope?.data;
    return compactDetails([
      getTimeDetail(context.event, envelope),
      ["Instance Name", data?.resourceName],
      ["Instance ID", data?.resourceId],
      ["Shape", data?.additionalDetails?.shape],
      ["Availability Domain", data?.availabilityDomain],
      ["Compartment ID", data?.compartmentId],
      ["Compartment", data?.compartmentName],
    ]);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const instanceName = lastEvent ? getInstanceName(lastEvent) : "";

    return {
      title: node.name || definition.label || "On Compute Instance Created",
      iconSrc: ociIcon,
      iconSlug: definition.icon || "oci",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          title: instanceName || "Compute instance created",
          subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};

function getTimeDetail(
  event: TriggerEventContext["event"],
  envelope: OciComputeLaunchEvent | undefined,
): [string, string | undefined] {
  if (event?.createdAt) {
    return ["Triggered At", new Date(event.createdAt).toLocaleString()];
  }

  if (envelope?.eventTime) {
    return ["Triggered At", new Date(envelope.eventTime).toLocaleString()];
  }

  return ["Triggered At", undefined];
}
