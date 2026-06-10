import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";

interface AlertEventData {
  state?: string;
  summary?: string;
  conditionName?: string;
  policyName?: string;
  resourceName?: string;
  resourceDisplayName?: string;
  observedValue?: string;
}

function lastSegment(value: string | undefined): string {
  if (!value) return "";
  const idx = value.lastIndexOf("/");
  return idx >= 0 ? value.slice(idx + 1) : value;
}

// Short, human label for the incident: the condition name, falling back to the
// policy's last path segment.
function incidentLabel(data: AlertEventData | undefined): string {
  if (!data) return "";
  return data.conditionName || lastSegment(data.policyName);
}

function buildTitle(data: AlertEventData | undefined): string {
  const label = incidentLabel(data);
  return label ? `Alerting incident · ${label}` : "Alerting incident";
}

export const onAlertTriggerRenderer: TriggerRenderer = {
  getEventState: (_context: TriggerEventContext) => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const data = context.event?.data as AlertEventData | undefined;
    return {
      title: buildTitle(data),
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  // The Details tab. Keep it to a handful of the most useful fields, with
  // "Emitted At" first, matching the other triggers across the repo.
  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = context.event?.data as AlertEventData | undefined;
    const details: Record<string, string> = {};
    if (context.event?.createdAt) details["Emitted At"] = new Date(context.event.createdAt).toLocaleString();
    if (data?.state) details["State"] = data.state;
    const condition = incidentLabel(data);
    if (condition) details["Condition"] = condition;
    if (data?.summary) details["Summary"] = data.summary;
    const resource = data?.resourceDisplayName || data?.resourceName;
    if (resource) details["Resource"] = resource;
    if (data?.observedValue) details["Observed Value"] = data.observedValue;
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const eventTitleAndSubtitle = lastEvent
      ? onAlertTriggerRenderer.getTitleAndSubtitle({ event: lastEvent })
      : undefined;
    // Surface the auto-created Cloud Monitoring notification channel on the node.
    const nodeMetadata = node.metadata as { notificationChannel?: string } | undefined;
    const metadata: { icon: string; label: string }[] = [];
    if (nodeMetadata?.notificationChannel) {
      metadata.push({ icon: "bell", label: lastSegment(nodeMetadata.notificationChannel) });
    }
    return {
      title: node.name || definition.label || "On Alert",
      iconSrc: gcpIcon,
      iconSlug: definition.icon || "bell",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "blue"),
      metadata,
      ...(lastEvent && {
        lastEventData: {
          title: eventTitleAndSubtitle?.title ?? "Alerting incident",
          subtitle: eventTitleAndSubtitle?.subtitle ?? renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
