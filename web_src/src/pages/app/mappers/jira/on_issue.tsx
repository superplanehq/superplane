import { getBackgroundColorClass } from "@/lib/colors";
import type {
  CustomFieldRenderer,
  NodeInfo,
  TriggerEventContext,
  TriggerRenderer,
  TriggerRendererContext,
} from "../types";
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
  webhookUrl?: string;
  webhookId?: number;
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

export const onIssueCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnIssueNodeMetadata | undefined;
    const registered = metadata?.webhookId != null;

    return (
      <div className="border-t-1 border-gray-200 dark:border-gray-600 pt-4">
        <div className="space-y-3">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Jira Webhook</span>
          <div className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md space-y-2">
            {registered ? (
              <p className="text-gray-600 dark:text-gray-400">
                Registered automatically on Jira (webhook id <code>{metadata!.webhookId}</code>) via OAuth. It's removed
                automatically if you delete this trigger.
              </p>
            ) : (
              <p className="text-gray-600 dark:text-gray-400">
                Will be registered automatically on Jira once you save the canvas - no manual setup needed.
              </p>
            )}
          </div>
        </div>
      </div>
    );
  },
};
