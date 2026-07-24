import { getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import type { JiraIssue, JiraProject, JiraUser } from "./types";

interface OnIssueEventData {
  action?: string;
  issue?: JiraIssue;
  user?: JiraUser;
}

interface OnIssueConfiguration {
  project?: string;
  events?: string[];
}

interface OnIssueNodeMetadata {
  project?: JiraProject;
}

const ACTION_LABELS: Record<string, string> = {
  created: "Created",
  updated: "Updated",
  deleted: "Deleted",
};

function actionLabel(action?: string): string {
  if (!action) return "";
  return ACTION_LABELS[action] ?? action;
}

function issueTitle(issue?: JiraIssue): string {
  if (!issue) return "Issue event";
  const summary = issue.fields?.summary;
  return summary ? `${issue.key} - ${summary}` : issue.key || "Issue event";
}

/**
 * Renderer for the "jira.onIssue" trigger.
 */
export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const data = context.event?.data as OnIssueEventData | undefined;
    const label = actionLabel(data?.action);
    const timeAgo = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return {
      title: issueTitle(data?.issue),
      subtitle: label && timeAgo ? `${label} - ${timeAgo}` : label || timeAgo,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = (context.event?.data ?? {}) as OnIssueEventData;
    const issue = data.issue;
    const fields = issue?.fields;
    const receivedAt = context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-";

    return {
      "Received At": receivedAt,
      Action: stringOrDash(actionLabel(data.action)),
      Key: stringOrDash(issue?.key),
      Summary: stringOrDash(fields?.summary),
      Status: stringOrDash(fields?.status?.name),
      Priority: stringOrDash(fields?.priority?.name),
      "Issue Type": stringOrDash(fields?.issuetype?.name),
      Assignee: stringOrDash(fields?.assignee?.displayName),
      Reporter: stringOrDash(fields?.reporter?.displayName),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnIssueNodeMetadata | undefined;
    const configuration = node.configuration as OnIssueConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    const projectLabel = metadata?.project
      ? `${metadata.project.name} (${metadata.project.key})`
      : configuration?.project;
    if (projectLabel) {
      metadataItems.push({ icon: "folder", label: projectLabel });
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
      const { title, subtitle } = onIssueTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
