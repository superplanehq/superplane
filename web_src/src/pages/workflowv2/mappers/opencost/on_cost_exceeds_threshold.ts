import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import opencostIcon from "@/assets/icons/integrations/opencost.svg";
import { getDetailsForAllocation } from "./base";
import { CostAllocationPayload, OnCostExceedsThresholdConfiguration } from "./types";

const windowLabels: Record<string, string> = {
  "1h": "1 Hour",
  "1d": "1 Day",
  "2d": "2 Days",
  "7d": "7 Days",
};

const aggregateLabels: Record<string, string> = {
  namespace: "Namespace",
  cluster: "Cluster",
  controller: "Controller",
  service: "Service",
  deployment: "Deployment",
};

export const onCostExceedsThresholdTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as CostAllocationPayload;
    const title = buildEventTitle(eventData);
    const subtitle = buildEventSubtitle(eventData, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as CostAllocationPayload;
    return getDetailsForAllocation(eventData);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnCostExceedsThresholdConfiguration | undefined;
    const metadataItems = [];

    if (configuration?.threshold) {
      metadataItems.push({
        icon: "alert-triangle",
        label: `Threshold: $${configuration.threshold}`,
      });
    }

    if (configuration?.window) {
      metadataItems.push({
        icon: "clock",
        label: windowLabels[configuration.window] || configuration.window,
      });
    }

    if (configuration?.aggregate) {
      metadataItems.push({
        icon: "layers",
        label: aggregateLabels[configuration.aggregate] || configuration.aggregate,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: opencostIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems.slice(0, 3),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as CostAllocationPayload;
      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: buildEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildEventTitle(eventData: CostAllocationPayload): string {
  const name = eventData?.name || "Unknown";
  const totalCost = eventData?.totalCost !== undefined ? `$${eventData.totalCost.toFixed(2)}` : "";

  if (totalCost) {
    return `${name} · ${totalCost}`;
  }

  return name;
}

function buildEventSubtitle(eventData: CostAllocationPayload, createdAt?: string): string {
  const parts: string[] = [];

  if (eventData?.threshold !== undefined) {
    parts.push(`Threshold: $${eventData.threshold}`);
  }

  if (createdAt) {
    parts.push(formatTimeAgo(new Date(createdAt)));
  }

  return parts.join(" · ");
}
