import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";

interface AlertEventData {
  state?: string;
  summary?: string;
  conditionName?: string;
  policyName?: string;
}

function lastSegment(value: string | undefined): string {
  if (!value) return "";
  const idx = value.lastIndexOf("/");
  return idx >= 0 ? value.slice(idx + 1) : value;
}

function alertSummary(data: AlertEventData | undefined): string {
  if (!data) return "";
  if (data.summary) return data.summary;
  const condition = data.conditionName || lastSegment(data.policyName);
  const state = data.state ? data.state.toUpperCase() : "";
  return [state, condition].filter(Boolean).join(" — ");
}

export const onAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const data = context.event?.data as AlertEventData | undefined;
    return { title: "Alerting incident", subtitle: alertSummary(data) };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    return {
      title: node.name || definition.label || "On Alert",
      iconSrc: gcpIcon,
      iconSlug: definition.icon || "bell",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "blue"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          title: "Alerting incident",
          subtitle:
            alertSummary(lastEvent.data as AlertEventData) || renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
