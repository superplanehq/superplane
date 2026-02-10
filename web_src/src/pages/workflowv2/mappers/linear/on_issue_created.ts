import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import { Issue } from "./types";
import { getDetailsForIssue } from "./base";

interface OnIssueCreatedEventData {
  action?: string;
  type?: string;
  data?: Issue & { teamId?: string; stateId?: string; assigneeId?: string };
  actor?: { name?: string; email?: string };
  url?: string;
}

export const onIssueCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIssueCreatedEventData;
    const issue = eventData?.data;
    const subtitle = buildSubtitle(issue?.identifier, context.event?.createdAt);

    return {
      title: issue?.title || "Issue Created",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueCreatedEventData;
    const issue = eventData?.data;
    if (!issue) return {};

    const details: Record<string, string> = {};
    details.Identifier = issue.identifier || "-";
    details.Title = issue.title || "-";

    if (issue.teamId) {
      details["Team ID"] = issue.teamId;
    }

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
      const eventData = lastEvent.data as OnIssueCreatedEventData;
      const issue = eventData?.data;
      const subtitle = buildSubtitle(issue?.identifier, lastEvent.createdAt);

      props.lastEventData = {
        title: issue?.title || "Issue Created",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildSubtitle(identifier?: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (identifier && timeAgo) {
    return `${identifier} · ${timeAgo}`;
  }

  return identifier || timeAgo;
}
