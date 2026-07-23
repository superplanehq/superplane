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
    const webhookUrl = metadata?.webhookUrl || "URL will appear here after you save the canvas.";

    return (
      <div className="border-t-1 border-gray-200 dark:border-gray-600 pt-4">
        <div className="space-y-3">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Jira Webhook Setup</span>
          <div className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md space-y-3">
            <div>
              <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
              <pre className="mt-1 text-xs font-mono whitespace-pre-wrap break-all text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md">
                {webhookUrl}
              </pre>
            </div>
            <p className="text-gray-600 dark:text-gray-400">
              Jira Cloud's dynamic webhook API only works for Connect/OAuth apps, not API-token accounts, so this can't
              be provisioned automatically. Connect it on the Jira side with one of:
            </p>
            <ol className="list-decimal ml-4 space-y-1 text-gray-600 dark:text-gray-400">
              <li>
                <strong>Jira Administration → System → WebHooks</strong> (requires site admin): create a WebHook with
                the URL above, tick the Issue <strong>created</strong>/<strong>updated</strong>/<strong>deleted</strong>{" "}
                events you configured, and optionally scope it with a JQL filter such as{" "}
                <code>project = YOUR_PROJECT_KEY</code>.
              </li>
              <li>
                <strong>Project settings → Automation</strong> (no site admin needed): create a rule with an Issue
                created/updated/deleted trigger and a <strong>Send web request</strong> action pointing at the URL
                above, with a JSON body shaped like{" "}
                <code>{'{"webhookEvent": "jira:issue_created", "issue": {...}}'}</code>.
              </li>
            </ol>
            <p className="text-gray-600 dark:text-gray-400">
              Jira's native webhook delivery can't send custom headers, so requests aren't verified by default. If your
              setup can add one (for example an Automation rule), set the <strong>Webhook Shared Secret</strong> field
              on the Jira integration and send it as <code>Authorization: Bearer &lt;secret&gt;</code> to have
              SuperPlane verify each request.
            </p>
          </div>
        </div>
      </div>
    );
  },
};
