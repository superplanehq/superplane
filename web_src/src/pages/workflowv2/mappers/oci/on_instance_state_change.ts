import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import ociIcon from "@/assets/icons/integrations/oci.svg";
import { compactDetails } from "./base";

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
      instanceActionType?: string;
    };
  };
}

const instanceActionTypeLabels: Record<string, string> = {
  start: "Instance started",
  stop: "Instance stopped",
  reset: "Instance reset",
  softstop: "Instance soft-stopped",
  softreset: "Instance soft-reset",
};

const nonActionEventTypeLabels: Record<string, string> = {
  "com.oraclecloud.computeapi.terminateinstance.end": "Instance terminated",
};

function getEventEnvelope(event: TriggerEventContext["event"]): OciInstanceStateChangeEvent | undefined {
  return event?.data as OciInstanceStateChangeEvent | undefined;
}

function getInstanceName(event: TriggerEventContext["event"]): string {
  const envelope = getEventEnvelope(event);
  return envelope?.data?.resourceName ?? "";
}

function getEventSubtitle(event: TriggerEventContext["event"]): string | React.ReactNode {
  const title = getEventTitle(event);
  return title && event?.createdAt
    ? renderWithTimeAgo(title, new Date(event.createdAt))
    : title || (event?.createdAt ? renderTimeAgo(new Date(event.createdAt)) : "");
}

function getEventTitle(event: TriggerEventContext["event"]): string {
  const envelope = getEventEnvelope(event);
  const actionType = envelope?.data?.additionalDetails?.instanceActionType;
  if (actionType) {
    return instanceActionTypeLabels[actionType] ?? "Instance state changed";
  }

  return nonActionEventTypeLabels[envelope?.eventType ?? ""] ?? "Instance state changed";
}

export const onInstanceStateChangeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const name = getInstanceName(context.event);
    return {
      title: name || getEventTitle(context.event),
      subtitle: getEventSubtitle(context.event),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = getEventEnvelope(context.event);
    const data = envelope?.data;
    const compartment = data?.compartmentName ?? data?.compartmentId;
    return compactDetails([
      getTimeDetail(context.event, envelope),
      ["Instance Name", data?.resourceName],
      ["Action", data?.additionalDetails?.instanceActionType],
      ["Shape", data?.additionalDetails?.shape],
      ["Availability Domain", data?.availabilityDomain],
      ["Compartment", compartment],
    ]);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;

    return {
      title: node.name || definition.label || "On Instance State Change",
      iconSrc: ociIcon,
      iconSlug: definition.icon || "oci",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          ...onInstanceStateChangeTriggerRenderer.getTitleAndSubtitle({ event: lastEvent }),
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
