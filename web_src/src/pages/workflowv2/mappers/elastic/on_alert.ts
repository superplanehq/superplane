import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

export const onAlertFiresTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const payload = context.event?.data as Record<string, any> | undefined;

    const title = alertTitle(payload);
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const payload = context.event?.data as Record<string, any> | undefined;

    const details: Record<string, string> = {};

    const rule = payload?.ruleName || payload?.alertName || payload?.ruleId;
    if (rule) details["Rule"] = String(rule);
    if (payload?.alertName && payload.alertName !== rule) details["Alert Name"] = String(payload.alertName);
    if (payload?.spaceId) details["Space"] = String(payload.spaceId);
    if (payload?.status) details["Status"] = String(payload.status);
    if (payload?.severity) details["Severity"] = String(payload.severity);

    const tags = payload?.tags;
    if (Array.isArray(tags) && tags.length > 0) {
      details["Tags"] = tags.join(", ");
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const config = node.configuration as Record<string, any> | undefined;
    const metadataItems: MetadataItem[] = buildMetadataItems(config);

    if (lastEvent) {
      const payload = lastEvent.data as Record<string, any> | undefined;
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

interface Predicate {
  type: string;
  value: string;
}

function predicateLabel(predicates: Predicate[]): string {
  return predicates.map((p) => (p.type === "equals" ? p.value : `${p.type}: ${p.value}`)).join(", ");
}

function buildMetadataItems(config: Record<string, any> | undefined): MetadataItem[] {
  const items: MetadataItem[] = [];
  if (!config) return items;

  const rules: string[] = Array.isArray(config.rules) ? config.rules : [];
  const spaces: string[] = Array.isArray(config.spaces) ? config.spaces : [];
  const tags: Predicate[] = Array.isArray(config.tags) ? config.tags : [];
  const severities: Predicate[] = Array.isArray(config.severities) ? config.severities : [];
  const statuses: Predicate[] = Array.isArray(config.statuses) ? config.statuses : [];

  if (rules.length > 0) items.push({ icon: "bell", label: rules.join(", ") });
  if (spaces.length > 0) items.push({ icon: "layers", label: spaces.join(", ") });
  if (tags.length > 0) items.push({ icon: "tag", label: predicateLabel(tags) });
  if (severities.length > 0) items.push({ icon: "alert-triangle", label: predicateLabel(severities) });
  if (statuses.length > 0) items.push({ icon: "activity", label: predicateLabel(statuses) });

  return items;
}

function alertTitle(payload: Record<string, any> | undefined): string {
  if (!payload) return "Elastic alert received";
  const baseTitle = payload.ruleName || payload.alertName || payload.name || payload.title || "Elastic alert received";
  return payload.spaceId ? `${baseTitle} · ${payload.spaceId}` : baseTitle;
}
