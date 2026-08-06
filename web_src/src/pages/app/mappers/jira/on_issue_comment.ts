import { getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import type { JiraComment, JiraIssue, JiraProject } from "./types";

interface OnIssueCommentEventData {
  action?: string;
  comment?: JiraComment;
  issue?: JiraIssue;
}

interface OnIssueCommentConfiguration {
  project?: string;
  events?: string[];
  contentFilter?: string;
}

interface OnIssueCommentNodeMetadata {
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
  if (!issue) return "Comment event";
  const summary = issue.fields?.summary;
  return summary ? `${issue.key} - ${summary}` : issue.key || "Comment event";
}

const COMMENT_PREVIEW_LENGTH = 140;

function adfText(node: unknown): string {
  if (!node || typeof node !== "object") return "";
  const { text, content, attrs } = node as { text?: string; content?: unknown[]; attrs?: { text?: string } };
  if (typeof text === "string") return text;
  if (typeof attrs?.text === "string") return attrs.text;
  if (Array.isArray(content)) {
    const parts = content.map(adfText).filter((part) => part !== "");
    const separator = content.some(hasBlockContent) ? " " : "";
    return parts.join(separator);
  }
  return "";
}

function hasBlockContent(node: unknown): boolean {
  return !!node && typeof node === "object" && Array.isArray((node as { content?: unknown[] }).content);
}

function commentPreview(body: unknown): string {
  const text = typeof body === "string" ? body : adfText(body);
  return text.slice(0, COMMENT_PREVIEW_LENGTH);
}

/**
 * Renderer for the "jira.onIssueComment" trigger.
 */
export const onIssueCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const data = context.event?.data as OnIssueCommentEventData | undefined;
    const label = actionLabel(data?.action);
    const timeAgo = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return {
      title: issueTitle(data?.issue),
      subtitle: label && timeAgo ? `${label} - ${timeAgo}` : label || timeAgo,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = (context.event?.data ?? {}) as OnIssueCommentEventData;
    const { issue, comment } = data;
    const receivedAt = context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-";

    return {
      "Received At": receivedAt,
      Action: stringOrDash(actionLabel(data.action)),
      Key: stringOrDash(issue?.key),
      Comment: stringOrDash(commentPreview(comment?.body)),
      Author: stringOrDash(comment?.author?.displayName),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnIssueCommentNodeMetadata | undefined;
    const configuration = node.configuration as OnIssueCommentConfiguration | undefined;
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

    if (configuration?.contentFilter) {
      metadataItems.push({ icon: "funnel", label: `Filter: ${configuration.contentFilter}` });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jiraIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onIssueCommentTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
