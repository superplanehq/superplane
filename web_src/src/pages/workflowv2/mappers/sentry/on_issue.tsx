import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { addDetail, addFormattedTimestamp, getProjectLabel, splitSentryIssueTitle } from "./utils";

interface OnIssueConfiguration {
  project?: string;
  actions?: string[];
}

interface OnIssueNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
}

interface SentryIssueEventData {
  action?: string;
  data?: {
    issue?: {
      id?: string;
      shortId?: string;
      title?: string;
      status?: string;
      permalink?: string;
      web_url?: string;
      url?: string;
      project?: {
        slug?: string;
        name?: string;
      };
      assignedTo?: {
        name?: string;
      };
    };
  };
}

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as SentryIssueEventData;
    const issue = eventData?.data?.issue;
    const parsedTitle = splitSentryIssueTitle(issue?.title);
    const title = parsedTitle.title || "Issue";
    const action = eventData?.action?.trim();
    const status = issue?.status?.trim();

    const subtitleParts = [
      action,
      status && action?.toLowerCase() === status.toLowerCase() ? undefined : status,
      context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : undefined,
    ]
      .filter(Boolean)
      .map((value) => String(value));

    return {
      title,
      subtitle: subtitleParts.join(" · "),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as SentryIssueEventData;
    const issue = eventData?.data?.issue;
    const details: Record<string, string> = {};
    const issueUrl = issue?.web_url || issue?.permalink || issue?.url;

    addFormattedTimestamp(details, "Triggered At", context.event?.createdAt);
    addDetail(details, "Title", issue?.title);
    addDetail(details, "Issue URL", issueUrl);
    addDetail(details, "Action", eventData?.action);
    addDetail(details, "Status", issue?.status);
    addDetail(details, "Project", getProjectLabel(issue));
    addDetail(details, "Assigned To", issue?.assignedTo?.name);

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnIssueConfiguration | undefined;
    const nodeMetadata = node.metadata as OnIssueNodeMetadata | undefined;
    const metadata = [];

    const projectLabel = nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
    if (projectLabel) {
      metadata.push({ icon: "folder", label: projectLabel });
    }

    if (configuration?.actions?.length) {
      metadata.push({ icon: "funnel", label: configuration.actions.join(", ") });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: sentryIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const { title, subtitle } = onIssueTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
