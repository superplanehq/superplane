import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import ociIcon from "@/assets/icons/integrations/oci.svg";

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
    const details: Record<string, string> = {};

    if (context.event?.createdAt) {
      details["Triggered At"] = new Date(context.event.createdAt).toLocaleString();
    } else if (envelope?.eventTime) {
      details["Event Time"] = new Date(envelope.eventTime).toLocaleString();
    }

    const data = envelope?.data;
    if (data?.resourceName) {
      details["Instance Name"] = data.resourceName;
    }
    if (data?.resourceId) {
      details["Instance ID"] = data.resourceId;
    }
    if (data?.additionalDetails?.shape) {
      details["Shape"] = data.additionalDetails.shape;
    }
    if (data?.availabilityDomain) {
      details["Availability Domain"] = data.availabilityDomain;
    }
    if (data?.compartmentId) {
      details["Compartment ID"] = data.compartmentId;
    }
    if (data?.compartmentName) {
      details["Compartment"] = data.compartmentName;
    }

    return details;
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
          title: "Compute instance created",
          subtitle: instanceName || renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
