import { getBackgroundColorClass } from "@/utils/colors";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

interface OnCaseStatusChangeConfiguration {
  cases?: string[];
  statuses?: string[];
  severities?: string[];
  tags?: { value?: string }[];
}

interface OnCaseStatusChangeNodeMetadata {
  caseNames?: Record<string, string>;
}

interface OnCaseStatusChangeEventData {
  id?: string;
  title?: string;
  status?: string;
  severity?: string;
}

export const onCaseStatusChangeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const payload = context.event?.data as OnCaseStatusChangeEventData | undefined;
    const status = payload?.status ? ` to ${payload.status}` : "";
    const title = payload?.title ? `Case "${payload.title}" changed${status}` : "Case status changed";
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const payload = context.event?.data as OnCaseStatusChangeEventData | undefined;
    const details: Record<string, string> = {};
    if (context.event?.createdAt) details["Triggered At"] = new Date(context.event.createdAt).toLocaleString();
    if (payload?.id) details["Case ID"] = String(payload.id);
    if (payload?.title) details["Title"] = String(payload.title);
    if (payload?.status) details["Status"] = String(payload.status);
    if (payload?.severity) details["Severity"] = String(payload.severity);
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const config = node.configuration as OnCaseStatusChangeConfiguration | undefined;
    const nodeMetadata = node.metadata as OnCaseStatusChangeNodeMetadata | undefined;
    const metadata: MetadataItem[] = [];
    if (config?.cases && config.cases.length > 0) {
      const caseNames = config.cases.map((id) => nodeMetadata?.caseNames?.[id] ?? id);
      metadata.push({ icon: "folder", label: caseNames.join(", ") });
    }
    if (config?.statuses && config.statuses.length > 0) {
      metadata.push({ icon: "activity", label: config.statuses.join(", ") });
    }
    if (config?.severities && config.severities.length > 0) {
      metadata.push({ icon: "alert-triangle", label: config.severities.join(", ") });
    }
    if (config?.tags && config.tags.length > 0) {
      metadata.push({
        icon: "tag",
        label: config.tags
          .map((t) => t.value)
          .filter(Boolean)
          .join(", "),
      });
    }

    if (lastEvent) {
      const payload = lastEvent.data as OnCaseStatusChangeEventData | undefined;
      const status = payload?.status ? ` to ${payload.status}` : "";
      const title = payload?.title ? `Case "${payload.title}" changed${status}` : "Case status changed";
      return {
        title: node.name || definition.label || "Unnamed trigger",
        iconSrc: elasticIcon,
        collapsedBackground: getBackgroundColorClass(definition.color),
        metadata,
        lastEventData: {
          title,
          subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
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
      metadata,
    };
  },
};
