import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import SentryLogo from "@/assets/icons/integrations/sentry.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";

interface OnIssueEventMetadata {
  project?: {
    id: string;
    slug: string;
    name: string;
  };
}

interface OnIssueEventConfiguration {
  actions?: string[];
  project?: string;
}

interface OnIssueEventData {
  action?: string;
  data?: {
    issue?: {
      id?: string;
      shortId?: string;
      title?: string;
      level?: string;
      status?: string;
      permalink?: string;
      project?: {
        slug?: string;
        name?: string;
      };
      assignedTo?: {
        name?: string;
        email?: string;
      } | null;
      firstSeen?: string;
      lastSeen?: string;
    };
  };
}

/**
 * Renderer for the "sentry.onIssueEvent" trigger type
 */
export const onIssueEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.data?.issue;
    const action = eventData?.action || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";

    const shortId = issue?.shortId || "";
    const issueTitle = issue?.title || "Unknown issue";
    const title = shortId ? `[${shortId}] ${issueTitle}` : issueTitle;
    const subtitle = action && timeAgo ? `${action} · ${timeAgo}` : action || timeAgo;

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.data?.issue;

    return {
      Action: eventData?.action || "",
      "Issue URL": issue?.permalink || "",
      Title: issue?.title || "",
      Level: issue?.level || "",
      Status: issue?.status || "",
      Project: issue?.project?.name || "",
      "Assigned To": issue?.assignedTo?.name || issue?.assignedTo?.email || "",
      "First Seen": issue?.firstSeen ? new Date(issue.firstSeen).toLocaleString() : "",
      "Last Seen": issue?.lastSeen ? new Date(issue.lastSeen).toLocaleString() : "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnIssueEventMetadata;
    const configuration = node.configuration as unknown as OnIssueEventConfiguration;
    const metadataItems: MetadataItem[] = [];

    if (metadata?.project?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.project.name,
      });
    }

    if (configuration?.actions?.length) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: SentryLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;
      const issue = eventData?.data?.issue;
      const action = eventData?.action || "";
      const shortId = issue?.shortId || "";
      const issueTitle = issue?.title || "Unknown issue";
      const title = shortId ? `[${shortId}] ${issueTitle}` : issueTitle;
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = action && timeAgo ? `${action} · ${timeAgo}` : action || timeAgo;

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
