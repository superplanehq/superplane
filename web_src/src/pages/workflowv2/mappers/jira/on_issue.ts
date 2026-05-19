import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { buildSubtitle, stringOrDash } from "../utils";

interface OnIssueConfiguration {
  project?: string;
  actions?: string[];
}

interface JiraProject {
  id?: string;
  key?: string;
  name?: string;
}

interface JiraIssue {
  id?: string;
  key?: string;
  self?: string;
  fields?: {
    summary?: string;
    project?: JiraProject;
    issuetype?: {
      name?: string;
    };
    status?: {
      name?: string;
    };
  };
}

interface OnIssueEventData {
  action?: string;
  issue?: JiraIssue;
}

interface JiraNodeMetadata {
  project?: JiraProject;
}

const actionLabels: Record<string, string> = {
  created: "Created",
  updated: "Updated",
  deleted: "Deleted",
};

function formatAction(action?: string): string {
  if (!action) {
    return "";
  }

  return actionLabels[action] || action;
}

function issueTitle(issue?: JiraIssue): string {
  const key = issue?.key;
  const summary = issue?.fields?.summary;
  if (key && summary) {
    return `${key} - ${summary}`;
  }

  return key || summary || "Jira issue";
}

function getDetailsForIssue(issue?: JiraIssue, action?: string): Record<string, string> {
  return {
    Action: stringOrDash(formatAction(action)),
    Key: stringOrDash(issue?.key),
    Summary: stringOrDash(issue?.fields?.summary),
    Project: stringOrDash(issue?.fields?.project?.name || issue?.fields?.project?.key),
    Type: stringOrDash(issue?.fields?.issuetype?.name),
    Status: stringOrDash(issue?.fields?.status?.name),
    URL: stringOrDash(issue?.self),
  };
}

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnIssueEventData;
    return {
      title: issueTitle(eventData?.issue),
      subtitle: buildSubtitle(formatAction(eventData?.action), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueEventData;
    return getDetailsForIssue(eventData?.issue, eventData?.action);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnIssueConfiguration;
    const metadata = node.metadata as JiraNodeMetadata;
    const metadataItems: { icon: string; label: string }[] = [];

    if (metadata?.project?.name || configuration?.project) {
      metadataItems.push({
        icon: "folder",
        label: metadata?.project?.name || configuration?.project || "",
      });
    }

    if (configuration?.actions?.length) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.map(formatAction).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jiraIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;
      props.lastEventData = {
        title: issueTitle(eventData?.issue),
        subtitle: buildSubtitle(formatAction(eventData?.action), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
