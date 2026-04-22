import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import ociIcon from "@/assets/icons/integrations/oci.svg";

interface OciInstanceStateChangeEvent {
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

const eventTypeLabels: Record<string, string> = {
  "com.oraclecloud.computeapi.startinstance.end": "Instance started",
  "com.oraclecloud.computeapi.stopinstance.end": "Instance stopped",
  "com.oraclecloud.computeapi.terminateinstance.end": "Instance terminated",
  "com.oraclecloud.computeapi.resetinstance.end": "Instance reset",
  "com.oraclecloud.computeapi.softstopinstance.end": "Instance soft-stopped",
  "com.oraclecloud.computeapi.softresetinstance.end": "Instance soft-reset",
};

function getEventEnvelope(event: TriggerEventContext["event"]): OciInstanceStateChangeEvent | undefined {
  return event?.data as OciInstanceStateChangeEvent | undefined;
}

function getInstanceName(event: TriggerEventContext["event"]): string {
  const envelope = getEventEnvelope(event);
  return envelope?.data?.resourceName ?? "";
}

function getEventTitle(event: TriggerEventContext["event"]): string {
  const envelope = getEventEnvelope(event);
  return eventTypeLabels[envelope?.eventType ?? ""] ?? "Instance state changed";
}

export const onInstanceStateChangeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const name = getInstanceName(context.event);
    return {
      title: getEventTitle(context.event),
      subtitle: name || "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = getEventEnvelope(context.event);
    const data = envelope?.data;
    const compartment = data?.compartmentName ?? data?.compartmentId;
    return compactDetails([
      getTimeDetail(context.event, envelope),
      ["Instance Name", data?.resourceName],
      ["Instance ID", data?.resourceId],
      ["Shape", data?.additionalDetails?.shape],
      ["Availability Domain", data?.availabilityDomain],
      ["Compartment", compartment],
    ]);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const instanceName = lastEvent ? getInstanceName(lastEvent) : "";

    return {
      title: node.name || definition.label || "On Instance State Change",
      iconSrc: ociIcon,
      iconSlug: definition.icon || "oci",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          title: getEventTitle(lastEvent),
          subtitle: instanceName || renderTimeAgo(new Date(lastEvent.createdAt)),
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
  envelope: OciInstanceStateChangeEvent | undefined,
): [string, string | undefined] {
  if (event?.createdAt) {
    return ["Triggered At", new Date(event.createdAt).toLocaleString()];
  }

  if (envelope?.eventTime) {
    return ["Event Time", new Date(envelope.eventTime).toLocaleString()];
  }

  return ["Triggered At", undefined];
}

function compactDetails(entries: Array<[string, string | undefined]>): Record<string, string> {
  const details: Record<string, string> = {};

  for (const [key, value] of entries) {
    if (value) {
      details[key] = value;
    }
  }

  return details;
}
