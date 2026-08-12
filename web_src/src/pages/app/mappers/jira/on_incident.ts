import { getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import type { JiraIssue, JiraServiceDesk, JiraUser } from "./types";
import { actionLabel, issueFieldValues } from "./on_issue";

interface OnIncidentEventData {
  action?: string;
  issue?: JiraIssue;
  user?: JiraUser;
}

interface OnIncidentConfiguration {
  serviceDesk?: string;
  requestTypes?: string[];
  events?: string[];
}

interface OnIncidentNodeMetadata {
  serviceDesk?: JiraServiceDesk;
}

function incidentTitle(issue?: JiraIssue): string {
  if (!issue) return "Incident event";
  const summary = issue.fields?.summary;
  return summary ? `${issue.key} - ${summary}` : issue.key || "Incident event";
}

/**
 * Renderer for the "jira.onIncident" trigger.
 */
export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const data = context.event?.data as OnIncidentEventData | undefined;
    const label = actionLabel(data?.action);
    const timeAgo = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return {
      title: incidentTitle(data?.issue),
      subtitle: label && timeAgo ? `${label} - ${timeAgo}` : label || timeAgo,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = (context.event?.data ?? {}) as OnIncidentEventData;
    const issue = data.issue;
    const receivedAt = context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-";

    return {
      "Received At": receivedAt,
      Action: stringOrDash(actionLabel(data.action)),
      Key: stringOrDash(issue?.key),
      ...issueFieldValues(issue?.fields),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnIncidentNodeMetadata | undefined;
    const configuration = node.configuration as OnIncidentConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    const serviceDeskLabel = metadata?.serviceDesk
      ? `${metadata.serviceDesk.projectName} (${metadata.serviceDesk.projectKey})`
      : configuration?.serviceDesk;
    if (serviceDeskLabel) {
      metadataItems.push({ icon: "layers", label: serviceDeskLabel });
    }

    if (configuration?.requestTypes?.length) {
      metadataItems.push({
        icon: "tag",
        label: `${configuration.requestTypes.length} request type${configuration.requestTypes.length === 1 ? "" : "s"}`,
      });
    }

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
      const { title, subtitle } = onIncidentTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
