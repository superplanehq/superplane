import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

type UnknownRecord = Record<string, unknown>;

interface ElasticAlertPayload {
  ruleName?: string;
  alertName?: string;
  ruleId?: string;
  spaceId?: string;
  status?: string;
  severity?: string;
  tags?: string[];
  name?: string;
  title?: string;
}

interface OnAlertConfiguration {
  rules: string[];
  spaces: string[];
  tags: Predicate[];
  severities: Predicate[];
  statuses: Predicate[];
}

export const onAlertFiresTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const payload = toAlertPayload(context.event?.data);

    const title = alertTitle(payload);
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const payload = toAlertPayload(context.event?.data);

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
    const config = toOnAlertConfiguration(node.configuration);
    const metadataItems: MetadataItem[] = buildMetadataItems(config);

    if (lastEvent) {
      const payload = toAlertPayload(lastEvent.data);
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

function buildMetadataItems(config: OnAlertConfiguration): MetadataItem[] {
  const items: MetadataItem[] = [];
  const { rules, spaces, tags, severities, statuses } = config;

  if (rules.length > 0) items.push({ icon: "bell", label: rules.join(", ") });
  if (spaces.length > 0) items.push({ icon: "layers", label: spaces.join(", ") });
  if (tags.length > 0) items.push({ icon: "tag", label: predicateLabel(tags) });
  if (severities.length > 0) items.push({ icon: "alert-triangle", label: predicateLabel(severities) });
  if (statuses.length > 0) items.push({ icon: "activity", label: predicateLabel(statuses) });

  return items;
}

function alertTitle(payload: ElasticAlertPayload | undefined): string {
  if (!payload) return "Elastic alert received";
  const baseTitle = payload.ruleName || payload.alertName || payload.name || payload.title || "Elastic alert received";
  return payload.spaceId ? `${baseTitle} · ${payload.spaceId}` : baseTitle;
}

function toOnAlertConfiguration(value: unknown): OnAlertConfiguration {
  const config = toUnknownRecord(value);
  return {
    rules: toStringList(config?.rules),
    spaces: toStringList(config?.spaces),
    tags: toPredicates(config?.tags),
    severities: toPredicates(config?.severities),
    statuses: toPredicates(config?.statuses),
  };
}

function toAlertPayload(value: unknown): ElasticAlertPayload | undefined {
  const payload = toUnknownRecord(value);
  if (!payload) return undefined;

  return {
    ruleName: toOptionalString(payload.ruleName),
    alertName: toOptionalString(payload.alertName),
    ruleId: toOptionalString(payload.ruleId),
    spaceId: toOptionalString(payload.spaceId),
    status: toOptionalString(payload.status),
    severity: toOptionalString(payload.severity),
    tags: toStringList(payload.tags),
    name: toOptionalString(payload.name),
    title: toOptionalString(payload.title),
  };
}

function toUnknownRecord(value: unknown): UnknownRecord | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  return value as UnknownRecord;
}

function toOptionalString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function toStringList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === "string");
}

function toPredicates(value: unknown): Predicate[] {
  if (!Array.isArray(value)) return [];

  return value
    .map((item) => {
      const predicate = toUnknownRecord(item);
      const type = toOptionalString(predicate?.type);
      const predicateValue = toOptionalString(predicate?.value);
      if (!type || !predicateValue) return undefined;
      return { type, value: predicateValue };
    })
    .filter((predicate): predicate is Predicate => predicate !== undefined);
}
