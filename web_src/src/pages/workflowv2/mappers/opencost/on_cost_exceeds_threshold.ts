import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import opencostIcon from "@/assets/icons/integrations/opencost.svg";
import { CostExceedsThresholdPayload } from "./types";

export const onCostExceedsThresholdTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as CostExceedsThresholdPayload;
    const title = buildEventTitle(eventData);
    const subtitle = buildEventSubtitle(eventData, context.event?.createdAt);

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as CostExceedsThresholdPayload;
    return getDetailsForCostEvent(eventData);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as Record<string, unknown> | undefined;
    const metadataItems = [];

    if (configuration?.window) {
      metadataItems.push({
        icon: "clock",
        label: `Window: ${configuration.window}`,
      });
    }

    if (configuration?.aggregate) {
      metadataItems.push({
        icon: "layers",
        label: `By: ${configuration.aggregate}`,
      });
    }

    if (configuration?.threshold) {
      metadataItems.push({
        icon: "alert-triangle",
        label: `Threshold: $${configuration.threshold}`,
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
      const eventData = lastEvent.data as CostExceedsThresholdPayload;
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

function buildEventTitle(eventData: CostExceedsThresholdPayload): string {
  if (!eventData) return "Cost threshold exceeded";

  const cost = eventData.totalCost !== undefined ? `$${eventData.totalCost.toFixed(2)}` : "";
  const threshold = eventData.threshold !== undefined ? `$${eventData.threshold.toFixed(2)}` : "";

  if (cost && threshold) {
    return `Cost ${cost} exceeds ${threshold}`;
  }

  return "Cost threshold exceeded";
}

function buildEventSubtitle(eventData: CostExceedsThresholdPayload, createdAt?: string): string {
  const parts: string[] = [];

  if (eventData?.aggregate) {
    parts.push(`by ${eventData.aggregate}`);
  }

  if (eventData?.window) {
    parts.push(eventData.window);
  }

  if (createdAt) {
    parts.push(formatTimeAgo(new Date(createdAt)));
  }

  return parts.join(" · ");
}

function getDetailsForCostEvent(eventData: CostExceedsThresholdPayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (eventData?.totalCost !== undefined) {
    details["Total Cost"] = `$${eventData.totalCost.toFixed(2)}`;
  }

  if (eventData?.threshold !== undefined) {
    details["Threshold"] = `$${eventData.threshold.toFixed(2)}`;
  }

  if (eventData?.window) {
    details["Window"] = eventData.window;
  }

  if (eventData?.aggregate) {
    details["Aggregate By"] = eventData.aggregate;
  }

  if (eventData?.exceedingItems && eventData.exceedingItems.length > 0) {
    details["Exceeding Items"] = String(eventData.exceedingItems.length);
  }

  return details;
}
