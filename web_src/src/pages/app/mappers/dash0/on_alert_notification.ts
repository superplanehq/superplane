import { getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { stringOrDash } from "../eventDisplay";

interface AlertNotificationIssue {
  id?: string;
  issueIdentifier?: string;
  status?: string;
  summary?: string;
  start?: string;
  end?: string;
  url?: string;
  dataset?: string;
  description?: string;
  labels?: AlertIssueLabel[];
}

interface AlertIssueLabel {
  key?: string;
  value?: AlertIssueLabelValue;
}

interface AlertIssueLabelValue {
  stringValue?: string;
}

interface AlertNotificationEventData {
  issue?: AlertNotificationIssue;
}

interface OnAlertNotificationConfiguration {
  statuses?: string[];
}

function alertTitle(issue?: AlertNotificationIssue): string {
  return issue?.summary || issue?.issueIdentifier || issue?.id || "Dash0 alert notification";
}

function alertSubtitle(issue?: AlertNotificationIssue, createdAt?: string): string | React.ReactNode {
  const subtitleParts = [issue?.status].filter(Boolean).join(" · ");
  if (subtitleParts && createdAt) {
    return renderWithTimeAgo(subtitleParts, new Date(createdAt));
  }
  if (subtitleParts) {
    return subtitleParts;
  }
  return createdAt ? renderTimeAgo(new Date(createdAt)) : "";
}

function formatAlertLabels(labels?: AlertIssueLabel[]): string {
  return stringOrDash(labels?.map((label) => `${label.key}: ${label.value?.stringValue}`).join(", "));
}

function alertRootEventValues(issue?: AlertNotificationIssue): Record<string, string> {
  return {
    "Issue ID": stringOrDash(issue?.id),
    "Issue Identifier": stringOrDash(issue?.issueIdentifier),
    URL: stringOrDash(issue?.url),
    Status: stringOrDash(issue?.status),
    Summary: stringOrDash(issue?.summary),
    Dataset: stringOrDash(issue?.dataset),
    Start: stringOrDash(issue?.start),
    Labels: formatAlertLabels(issue?.labels),
  };
}

export const onAlertNotificationTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as AlertNotificationEventData | undefined;
    const issue = eventData?.issue;
    return {
      title: alertTitle(issue),
      subtitle: alertSubtitle(issue, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as AlertNotificationEventData | undefined;
    return alertRootEventValues(eventData?.issue);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnAlertNotificationConfiguration | undefined;
    const metadataItems = [];

    if (configuration?.statuses?.length) {
      metadataItems.push({
        icon: "funnel",
        label: `Statuses: ${configuration.statuses.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: dash0Icon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onAlertNotificationTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
