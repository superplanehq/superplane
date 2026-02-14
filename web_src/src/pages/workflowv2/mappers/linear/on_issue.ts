import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import { Issue } from "./types";

interface OnIssueEventData {
  action?: string;
  type?: string;
  data?: Issue & { teamId?: string; stateId?: string; assigneeId?: string };
  actor?: { name?: string; email?: string };
  url?: string;
}

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.data;
    const subtitle = buildSubtitle(issue?.identifier, context.event?.createdAt);

    return {
      title: buildTitle(eventData),
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.data;
    if (!issue) return {};

    const details: Record<string, string> = {};

    Object.assign(details, {
      "Created At": issue.createdAt ? new Date(issue.createdAt).toLocaleString() : "-",
    });

    details.Identifier = issue.identifier || "-";
    details.Title = issue.title || "-";

    if (eventData?.actor?.name) {
      details.Actor = eventData.actor.name;
    }

    if (eventData?.url) {
      details.URL = eventData.url;
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadataItems = [];

    const nodeMetadata = node.metadata as { team?: { name?: string; key?: string } };
    if (nodeMetadata?.team?.name) {
      const label = nodeMetadata.team.key
        ? `Team: ${nodeMetadata.team.name} (${nodeMetadata.team.key})`
        : `Team: ${nodeMetadata.team.name}`;
      metadataItems.push({ icon: "funnel", label });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: linearIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;
      const issue = eventData?.data;
      const subtitle = buildSubtitle(issue?.identifier, lastEvent.createdAt);

      props.lastEventData = {
        title: buildTitle(eventData),
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

const actionLabels: Record<string, string> = {
  create: "Issue created",
  update: "Issue updated",
  remove: "Issue removed",
};

function buildTitle(eventData?: OnIssueEventData): string {
  const action = eventData?.action;
  const label = (action && actionLabels[action]) || "Issue";
  const title = eventData?.data?.title;
  return title ? `${label}: ${title}` : label;
}

function buildSubtitle(identifier?: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (identifier && timeAgo) {
    return `${identifier} · ${timeAgo}`;
  }

  return identifier || timeAgo;
}
