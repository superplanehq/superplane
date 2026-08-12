import { getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import type { JiraOpsTeam } from "./types";

interface JiraAlert {
  alertId?: string;
  tinyId?: string;
  message?: string;
  priority?: string;
  tags?: string[];
  source?: string;
}

interface OnAlertEventData {
  action?: string;
  alert?: JiraAlert;
}

interface OnAlertConfiguration {
  team?: string;
  events?: string[];
}

interface OnAlertNodeMetadata {
  team?: JiraOpsTeam;
}

const ACTION_LABELS: Record<string, string> = {
  created: "Created",
  acknowledged: "Acknowledged",
  closed: "Closed",
  deleted: "Deleted",
};

function actionLabel(action?: string): string {
  if (!action) return "";
  return ACTION_LABELS[action] ?? action;
}

function alertTitle(alert?: JiraAlert): string {
  if (!alert) return "Alert event";
  const id = alert.tinyId ? `#${alert.tinyId}` : alert.alertId;
  return alert.message && id ? `${id} - ${alert.message}` : alert.message || id || "Alert event";
}

/**
 * Renderer for the "jira.onAlert" trigger.
 */
export const onAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const data = context.event?.data as OnAlertEventData | undefined;
    const label = actionLabel(data?.action);
    const timeAgo = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return {
      title: alertTitle(data?.alert),
      subtitle: label && timeAgo ? `${label} - ${timeAgo}` : label || timeAgo,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = (context.event?.data ?? {}) as OnAlertEventData;
    const alert = data.alert;
    const receivedAt = context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-";

    return {
      "Received At": receivedAt,
      Action: stringOrDash(actionLabel(data.action)),
      Message: stringOrDash(alert?.message),
      Priority: stringOrDash(alert?.priority),
      Source: stringOrDash(alert?.source),
      Tags: stringOrDash(alert?.tags?.join(", ")),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnAlertNodeMetadata | undefined;
    const configuration = node.configuration as OnAlertConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    const teamLabel = metadata?.team?.teamName || configuration?.team;
    metadataItems.push({ icon: "users", label: teamLabel || "All teams" });

    if (configuration?.events?.length) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.events.map((event) => actionLabel(event)).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jiraIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onAlertTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
