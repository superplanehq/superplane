import { getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { stringOrDash } from "../eventDisplay";

interface SyntheticCheckNotificationIssue {
  id?: string;
  issueIdentifier?: string;
  status?: string;
  summary?: string;
  start?: string;
  end?: string;
  url?: string;
  dataset?: string;
  description?: string;
  labels?: SyntheticCheckLabelTuple[];
}

type SyntheticCheckLabelTuple = [string, SyntheticCheckLabelEntry];

interface SyntheticCheckLabelEntry {
  key?: string;
  value?: SyntheticCheckLabelValue;
}

interface SyntheticCheckLabelValue {
  stringValue?: string;
}

interface SyntheticCheckNotificationEventData {
  issue?: SyntheticCheckNotificationIssue;
}

interface OnSyntheticCheckNotificationConfiguration {
  statuses?: string[];
}

function formatSyntheticCheckLabels(labels?: SyntheticCheckLabelTuple[]): string | undefined {
  if (!labels?.length) {
    return undefined;
  }

  return labels
    .map((tuple) => {
      if (!Array.isArray(tuple) || tuple.length < 2) {
        return undefined;
      }

      const entry = tuple[1];
      const key = entry?.key;
      const value = entry?.value?.stringValue;
      return key ? `${key}: ${value ?? ""}` : undefined;
    })
    .filter(Boolean)
    .join(", ");
}

function issueRootEventValues(issue: SyntheticCheckNotificationIssue | undefined): Record<string, string> {
  return {
    "Issue ID": stringOrDash(issue?.id),
    "Issue Identifier": stringOrDash(issue?.issueIdentifier),
    URL: stringOrDash(issue?.url),
    Status: stringOrDash(issue?.status),
    Summary: stringOrDash(issue?.summary),
    Dataset: stringOrDash(issue?.dataset),
    Start: stringOrDash(issue?.start),
    Labels: stringOrDash(formatSyntheticCheckLabels(issue?.labels)),
  };
}

function syntheticCheckSubtitle(subtitleParts: string, createdAt?: string): string | React.ReactNode {
  return subtitleParts && createdAt
    ? renderWithTimeAgo(subtitleParts, new Date(createdAt))
    : subtitleParts || (createdAt ? renderTimeAgo(new Date(createdAt)) : "");
}

export const onSyntheticCheckNotificationTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as SyntheticCheckNotificationEventData | undefined;
    const issue = eventData?.issue;
    const title = issue?.summary || issue?.issueIdentifier || issue?.id || "Dash0 synthetic check notification";
    const subtitleParts = [issue?.status].filter(Boolean).join(" · ");
    const subtitle = syntheticCheckSubtitle(subtitleParts, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as SyntheticCheckNotificationEventData | undefined;
    return issueRootEventValues(eventData?.issue);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnSyntheticCheckNotificationConfiguration | undefined;
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
      const { title, subtitle } = onSyntheticCheckNotificationTriggerRenderer.getTitleAndSubtitle({
        event: lastEvent,
      });
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
