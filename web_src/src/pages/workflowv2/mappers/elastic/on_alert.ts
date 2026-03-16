import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

export const onAlertFiresTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as Record<string, any> | undefined;
    const payload = eventData?.payload as Record<string, any> | undefined;

    const title = alertTitle(payload);
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as Record<string, any> | undefined;
    const payload = eventData?.payload as Record<string, any> | undefined;

    const details: Record<string, string> = {};

    if (payload?.ruleId) details["Rule ID"] = String(payload.ruleId);
    if (payload?.ruleName) details["Rule Name"] = String(payload.ruleName);
    if (payload?.alertName) details["Alert Name"] = String(payload.alertName);
    if (payload?.spaceId) details["Space"] = String(payload.spaceId);
    if (payload?.status) details["Status"] = String(payload.status);
    if (payload?.severity) details["Severity"] = String(payload.severity);

    const tags = payload?.tags;
    if (Array.isArray(tags) && tags.length > 0) {
      details["Tags"] = tags.join(", ");
    }

    if (eventData?.receivedAt) details["Received At"] = String(eventData.receivedAt);

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const config = node.configuration as Record<string, any> | undefined;
    const metadataItems: MetadataItem[] = buildMetadataItems(config);

    if (lastEvent) {
      const eventData = lastEvent.data as Record<string, any> | undefined;
      const payload = eventData?.payload as Record<string, any> | undefined;
      const title = alertTitle(payload);
      const subtitle = formatTimeAgo(new Date(lastEvent.createdAt));

      return {
        title: node.name || definition.label || "Unnamed trigger",
        iconSrc: elasticIcon,
        collapsedBackground: getBackgroundColorClass(definition.color),
        metadata: metadataItems,
        lastEventData: {
          title,
          subtitle,
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      };
    }

    return {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: elasticIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };
  },
};

function buildMetadataItems(config: Record<string, any> | undefined): MetadataItem[] {
  const items: MetadataItem[] = [];
  if (!config) return items;

  const ruleIds: string[] = Array.isArray(config.ruleIds) ? config.ruleIds : [];
  const spaceIds: string[] = Array.isArray(config.spaceIds) ? config.spaceIds : [];
  const tags: string[] = Array.isArray(config.tags) ? config.tags : [];
  const severities: string[] = Array.isArray(config.severities) ? config.severities : [];
  const statuses: string[] = Array.isArray(config.statuses) ? config.statuses : [];

  if (ruleIds.length > 0) items.push({ icon: "hash", label: ruleIds.join(", ") });
  if (spaceIds.length > 0) items.push({ icon: "layers", label: spaceIds.join(", ") });
  if (tags.length > 0) items.push({ icon: "tag", label: tags.join(", ") });
  if (severities.length > 0) items.push({ icon: "alert-triangle", label: severities.join(", ") });
  if (statuses.length > 0) items.push({ icon: "activity", label: statuses.join(", ") });

  return items;
}

function alertTitle(payload: Record<string, any> | undefined): string {
  if (!payload) return "Elastic alert received";
  return (
    payload.ruleName ||
    payload.alertName ||
    payload.name ||
    payload.title ||
    "Elastic alert received"
  );
}
