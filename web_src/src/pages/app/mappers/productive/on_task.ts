import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import productiveIcon from "@/assets/icons/integrations/productive.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { OnTaskConfiguration, ProductiveNodeMetadata, ProductiveWebhookEvent } from "./types";

/** Webhook event names Productive.io sends for task events. */
const ACTION_LABELS: Record<string, string> = {
  created: "Created",
  updated: "Updated",
};

const EVENT_ACTIONS: Record<string, string> = {
  "task.created": "created",
  "task.updated": "updated",
};

function actionLabel(event: string | undefined): string | undefined {
  if (!event) return undefined;
  const action = EVENT_ACTIONS[event];
  return action ? ACTION_LABELS[action] : event;
}

function taskTitle(envelope: ProductiveWebhookEvent | undefined): string {
  return envelope?.data?.attributes?.title?.trim() || "";
}

export const onTaskTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const envelope = context.event?.data as ProductiveWebhookEvent | undefined;

    return {
      title: taskTitle(envelope) || "Task",
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = (context.event?.data ?? {}) as ProductiveWebhookEvent;
    const task = envelope.data;

    return {
      "Received At": context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-",
      Task: stringOrDash(task?.id),
      Title: stringOrDash(task?.attributes?.title),
      Action: stringOrDash(actionLabel(envelope.meta?.event)),
      Description: stringOrDash(task?.attributes?.description),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as ProductiveNodeMetadata | undefined;
    const configuration = node.configuration as OnTaskConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    const projectLabel = metadata?.project?.name || configuration?.project;
    if (projectLabel) {
      metadataItems.push({ icon: "folder", label: projectLabel });
    }

    if (configuration?.actions && configuration.actions.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.map((action) => ACTION_LABELS[action] ?? action).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: productiveIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onTaskTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
