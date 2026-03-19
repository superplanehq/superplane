import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import azureIcon from "@/assets/icons/integrations/azure.svg";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../utils";
import { getBackgroundColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";

// ── Shared configuration interfaces ─────────────────────────────────────────

interface MessageAvailableConfiguration {
  resourceGroup?: string;
  namespaceName?: string;
  queueName?: string;
}

interface MessageReceivedConfiguration {
  namespaceName?: string;
  entityType?: string;
  queueName?: string;
  topicName?: string;
  subscriptionName?: string;
}

// ── Event data shapes ────────────────────────────────────────────────────────

interface ServiceBusEventGridData {
  namespaceName?: string;
  queueName?: string;
  entityType?: string;
  deadLetterQueue?: boolean;
}

interface ServiceBusEventGridEvent {
  data?: ServiceBusEventGridData;
  eventType?: string;
  eventTime?: string;
}

interface ServiceBusMessageReceivedEvent {
  body?: string;
  messageId?: string;
  contentType?: string;
  namespaceName?: string;
  entityPath?: string;
}

// ── On Service Bus Message Available ────────────────────────────────────────

export const onServiceBusMessageAvailableTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle(context: TriggerEventContext): { title: string; subtitle: string } {
    const event = context.event?.data as ServiceBusEventGridEvent | undefined;
    const queue = event?.data?.queueName;
    const ns = event?.data?.namespaceName?.replace(".servicebus.windows.net", "");
    const title = queue ? `${queue}${ns ? ` (${ns})` : ""}` : "Messages available";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues(context: TriggerEventContext): Record<string, any> {
    const event = context.event?.data as ServiceBusEventGridEvent | undefined;
    return {
      Namespace: stringOrDash(event?.data?.namespaceName),
      Queue: stringOrDash(event?.data?.queueName),
      "Event Type": stringOrDash(event?.eventType),
      Time: stringOrDash(event?.eventTime),
    };
  },

  getTriggerProps(context: TriggerRendererContext): TriggerProps {
    const { node, definition, lastEvent } = context;
    const cfg = node.configuration as MessageAvailableConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (cfg?.queueName) metadata.push({ icon: "inbox", label: cfg.queueName });

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: azureIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const { title, subtitle } = onServiceBusMessageAvailableTriggerRenderer.getTitleAndSubtitle({
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

// ── On Service Bus Dead-Letter Available ─────────────────────────────────────

export const onServiceBusDeadLetterAvailableTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle(context: TriggerEventContext): { title: string; subtitle: string } {
    const event = context.event?.data as ServiceBusEventGridEvent | undefined;
    const queue = event?.data?.queueName;
    const ns = event?.data?.namespaceName?.replace(".servicebus.windows.net", "");
    const title = queue ? `${queue}/$deadLetterQueue${ns ? ` (${ns})` : ""}` : "Dead-letter messages available";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues(context: TriggerEventContext): Record<string, any> {
    const event = context.event?.data as ServiceBusEventGridEvent | undefined;
    return {
      Namespace: stringOrDash(event?.data?.namespaceName),
      Queue: stringOrDash(event?.data?.queueName),
      "Event Type": stringOrDash(event?.eventType),
      Time: stringOrDash(event?.eventTime),
    };
  },

  getTriggerProps(context: TriggerRendererContext): TriggerProps {
    const { node, definition, lastEvent } = context;
    const cfg = node.configuration as MessageAvailableConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (cfg?.queueName) metadata.push({ icon: "inbox", label: cfg.queueName });

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: azureIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const { title, subtitle } = onServiceBusDeadLetterAvailableTriggerRenderer.getTitleAndSubtitle({
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

// ── On Service Bus Message Received ─────────────────────────────────────────

export const onServiceBusMessageReceivedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle(context: TriggerEventContext): { title: string; subtitle: string } {
    const event = context.event?.data as ServiceBusMessageReceivedEvent | undefined;
    const entity = event?.entityPath ?? event?.namespaceName;
    const msgId = event?.messageId;
    const title = msgId ? `msg:${msgId.slice(0, 8)}${entity ? ` from ${entity}` : ""}` : "Message received";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues(context: TriggerEventContext): Record<string, any> {
    const event = context.event?.data as ServiceBusMessageReceivedEvent | undefined;
    return {
      "Message ID": stringOrDash(event?.messageId),
      "Content Type": stringOrDash(event?.contentType),
      Namespace: stringOrDash(event?.namespaceName),
      Entity: stringOrDash(event?.entityPath),
      Body: stringOrDash(event?.body),
    };
  },

  getTriggerProps(context: TriggerRendererContext): TriggerProps {
    const { node, definition, lastEvent } = context;
    const cfg = node.configuration as MessageReceivedConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (cfg?.entityType === "topic" && cfg.topicName) {
      metadata.push({ icon: "radio", label: cfg.topicName });
      if (cfg.subscriptionName) metadata.push({ icon: "git-branch", label: cfg.subscriptionName });
    } else if (cfg?.queueName) {
      metadata.push({ icon: "inbox", label: cfg.queueName });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: azureIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const { title, subtitle } = onServiceBusMessageReceivedTriggerRenderer.getTitleAndSubtitle({
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
