import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

interface OnIssueConfiguration {
  project?: string;
  actions?: string[];
}

interface SentryIssueEventData {
  action?: string;
  data?: {
    issue?: {
      id?: string;
      shortId?: string;
      title?: string;
      status?: string;
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

type SentryIssue = NonNullable<NonNullable<SentryIssueEventData["data"]>["issue"]>;

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as SentryIssueEventData;
    const issue = eventData?.data?.issue;
    const title = issue?.shortId ? `${issue.shortId} · ${issue.title || "Issue"}` : issue?.title || "Issue";

    const subtitleParts = [
      eventData?.action,
      issue?.status,
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

    addDetail(details, "Issue ID", issue?.id);
    addDetail(details, "Short ID", issue?.shortId);
    addDetail(details, "Title", issue?.title);
    addDetail(details, "Action", eventData?.action);
    addDetail(details, "Status", issue?.status);
    addDetail(details, "Project", getProjectLabel(issue));
    addDetail(details, "Assigned To", issue?.assignedTo?.name);

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnIssueConfiguration | undefined;
    const metadata = [];

    if (configuration?.project) {
      metadata.push({ icon: "folder", label: configuration.project });
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

function addDetail(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = value;
}

function getProjectLabel(issue?: SentryIssue) {
  return issue?.project?.name || issue?.project?.slug;
}
