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
