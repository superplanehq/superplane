import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { TriggerProps } from "@/ui/trigger";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { stringOrDash } from "../utils";

const eventLabels: Record<string, string> = {
  created: "Created",
  resolved: "Resolved",
  assigned: "Assigned",
  archived: "Archived",
  unresolved: "Unresolved",
};

interface OnIssueEventData {
  action?: string;
  installation?: {
    uuid?: string;
  };
  issue?: {
    id?: string;
    title?: string;
    shortId?: string;
    status?: string;
    permalink?: string;
    project?: {
      id?: string;
      slug?: string;
      name?: string;
    };
  };
  actor?: {
    id?: string | number;
    name?: string;
    type?: string;
  };
}

function normalizeEventData(raw: unknown): OnIssueEventData | undefined {
  if (!raw) return undefined;

  let value: unknown = raw;
  if (typeof value === "string") {
    try {
      value = JSON.parse(value) as unknown;
    } catch {
      return undefined;
    }
  }

  if (!value || typeof value !== "object") return undefined;

  const maybeWrapped = value as { data?: unknown };
  if (maybeWrapped.data && typeof maybeWrapped.data === "object") {
    return maybeWrapped.data as OnIssueEventData;
  }

  return value as OnIssueEventData;
}

function formatAction(action: string): string {
  return eventLabels[action] || action;
}

function buildSubtitle(label: string, createdAt?: string): string {
  if (createdAt) {
    return `${label} · ${formatTimeAgo(new Date(createdAt))}`;
  }
  return label;
}

/**
 * Renderer for the "sentry.onIssueEvent" trigger
 */
export const onIssueEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = normalizeEventData(context.event?.data);
    const action = eventData?.action ? formatAction(eventData.action) : "Issue event";
    const title = eventData?.issue?.title?.trim() || eventData?.issue?.shortId || action;
    const subtitle = buildSubtitle(action, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = normalizeEventData(context.event?.data);
    const issue = eventData?.issue;
    const project = issue?.project;
    const actor = eventData?.actor;

    return {
      Action: stringOrDash(eventData?.action && formatAction(eventData.action)),
      "Installation UUID": stringOrDash(eventData?.installation?.uuid),
      "Issue ID": stringOrDash(issue?.id),
      "Short ID": stringOrDash(issue?.shortId),
      Title: stringOrDash(issue?.title),
      Status: stringOrDash(issue?.status),
      Project: stringOrDash(project?.slug || project?.name),
      "Project ID": stringOrDash(project?.id),
      Actor: stringOrDash(actor?.name),
      "Actor ID": stringOrDash(actor?.id),
      Permalink: stringOrDash(issue?.permalink),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { events?: string[] } | undefined;
    const metadataItems: { icon: string; label: string }[] = [];

    if (configuration?.events?.length) {
      const label =
        configuration.events.length > 3
          ? `Events: ${configuration.events.length} selected`
          : `Events: ${configuration.events.map(formatAction).join(", ")}`;
      metadataItems.push({
        icon: "bell",
        label,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "On Issue Event",
      iconSrc: sentryIcon,
      iconSlug: "alert-triangle",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = normalizeEventData(lastEvent.data);
      const action = eventData?.action ? formatAction(eventData.action) : "Issue event";
      const title = eventData?.issue?.title?.trim() || eventData?.issue?.shortId || action;
      const subtitle = buildSubtitle(action, lastEvent.createdAt);

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
