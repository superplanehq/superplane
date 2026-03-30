import React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import type { OnAlertFiringConfiguration, OnAlertFiringEventData } from "./types";
import { stringOrDash } from "../utils";
import { formatOptionalIsoTimestamp } from "@/lib/timezone";

/**
 * Renderer for the "grafana.onAlertFiring" trigger
 */
export const onAlertFiringTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as OnAlertFiringEventData | undefined;
    const alertName = getAlertName(eventData);
    const status = eventData?.status || "firing";
    const subtitle = buildSubtitle(status, context.event?.createdAt);

    return {
      title: alertName || "Grafana alert firing",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnAlertFiringEventData | undefined;
    const createdAt = formatOptionalIsoTimestamp(context.event?.createdAt);

    return {
      "Triggered At": createdAt,
      Status: stringOrDash(eventData?.status || "firing"),
      "Alert Name": stringOrDash(getAlertName(eventData)),
      "Rule UID": stringOrDash(eventData?.ruleUid),
      "Rule ID": stringOrDash(eventData?.ruleId),
      "Org ID": stringOrDash(eventData?.orgId),
      "External URL": stringOrDash(eventData?.externalURL),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnAlertFiringConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: grafanaIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildTriggerMetadata(configuration, lastEvent?.data as OnAlertFiringEventData | undefined).slice(0, 3),
    };

    const lastEventData = buildLastEventData(lastEvent);
    if (lastEventData) {
      props.lastEventData = lastEventData;
    }

    return props;
  },
};

function getAlertName(eventData?: OnAlertFiringEventData): string | undefined {
  if (!eventData) return undefined;

  if (eventData.title && eventData.title.trim() !== "") {
    return eventData.title;
  }

  const commonLabel = eventData.commonLabels?.alertname;
  if (commonLabel && commonLabel.trim() !== "") {
    return commonLabel;
  }

  const firstAlert = eventData.alerts?.[0];
  const labelName = firstAlert?.labels?.alertname;
  if (labelName && labelName.trim() !== "") {
    return labelName;
  }

  return undefined;
}

function buildSubtitle(status: string, createdAt?: string): string | React.ReactNode {
  if (status && createdAt) {
    return renderWithTimeAgo(status, new Date(createdAt), " - ");
  }
  if (status) {
    return status;
  }
  return createdAt ? renderTimeAgo(new Date(createdAt)) : "-";
}

function buildTriggerMetadata(
  configuration: OnAlertFiringConfiguration | undefined,
  eventData: OnAlertFiringEventData | undefined,
): MetadataItem[] {
  const configuredAlertNameMetadata = buildConfiguredAlertNameMetadata(configuration);
  if (configuredAlertNameMetadata.length > 0) {
    return configuredAlertNameMetadata;
  }

  const alertName = getAlertName(eventData);
  if (!alertName) {
    return [];
  }

  return [{ icon: "bell", label: alertName }];
}

function buildConfiguredAlertNameMetadata(configuration: OnAlertFiringConfiguration | undefined): MetadataItem[] {
  const alertNames = configuration?.alertNames?.filter((predicate) => predicate.value.trim().length > 0) ?? [];
  if (alertNames.length === 0) {
    return [];
  }

  const label =
    alertNames.length > 3
      ? `Alert Names: ${alertNames.length} selected`
      : `Alert Names: ${alertNames.map((predicate) => predicate.value).join(", ")}`;

  return [{ icon: "bell", label }];
}

function buildLastEventData(lastEvent: TriggerRendererContext["lastEvent"]): TriggerProps["lastEventData"] | undefined {
  if (!lastEvent) {
    return undefined;
  }

  const eventData = lastEvent.data as OnAlertFiringEventData | undefined;
  const status = eventData?.status || "firing";
  const alertName = getAlertName(eventData);
  const subtitle = buildSubtitle(status, lastEvent.createdAt);

  return {
    title: alertName || "Grafana alert firing",
    subtitle,
    receivedAt: new Date(lastEvent.createdAt),
    state: "triggered",
    eventId: lastEvent.id,
  };
}
