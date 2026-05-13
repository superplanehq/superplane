import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { JiraIssue, JiraNodeMetadata, JiraWebhookEvent } from "./types";
import { addDetail, buildJiraSubtitle, getDetailsForIssue, getProjectLabel } from "./utils";

interface OnIssueConfiguration {
  project?: string;
  events?: string[];
  jql?: string;
}

function eventLabel(event: string | undefined): string {
  switch (event) {
    case "jira:issue_created":
      return "Created";
    case "jira:issue_updated":
      return "Updated";
    case "jira:issue_deleted":
      return "Deleted";
    default:
      return event || "";
  }
}

function titleFromIssue(issue: JiraIssue | undefined): string {
  const summary = issue?.fields?.summary;
  if (issue?.key && summary) {
    return `${issue.key} · ${summary}`;
  }
  return issue?.key || summary || "Issue";
}

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as JiraWebhookEvent | undefined;
    return {
      title: titleFromIssue(eventData?.issue),
      subtitle: buildJiraSubtitle(eventLabel(eventData?.webhookEvent), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as JiraWebhookEvent | undefined;
    const values = getDetailsForIssue(eventData?.issue);
    addDetail(values, "Event", eventLabel(eventData?.webhookEvent));
    addDetail(values, "User", eventData?.user?.displayName);
    addDetail(values, "Project", getProjectLabel(eventData?.issue));
    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as JiraNodeMetadata | undefined;
    const configuration = node.configuration as unknown as OnIssueConfiguration | undefined;
    const metadataItems = [] as { icon: string; label: string }[];

    const projectLabel = metadata?.project?.name || metadata?.project?.key || configuration?.project;
    if (projectLabel) {
      metadataItems.push({ icon: "folder", label: projectLabel });
    }

    if (configuration?.events && configuration.events.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.events.map(eventLabel).join(", "),
      });
    }

    if (configuration?.jql) {
      metadataItems.push({ icon: "filter", label: configuration.jql });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jiraIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as JiraWebhookEvent | undefined;
      props.lastEventData = {
        title: titleFromIssue(eventData?.issue),
        subtitle: buildJiraSubtitle(eventLabel(eventData?.webhookEvent), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
