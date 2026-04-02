import type { ComponentBaseProps } from "@/ui/componentBase";
import { createElement } from "react";
import type React from "react";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import type { ListAnnotationsConfiguration, ListAnnotationsOutput } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";
import { GrafanaDashboardMetadataLabel } from "./DashboardMetadataLabel";
import { baseEventSections } from "./base";

export const listAnnotationsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as ListAnnotationsConfiguration | undefined;
    const details: Record<string, string> = {
      "Listed At": formatTimestamp(context.execution.createdAt),
    };

    if (configuration?.tags && configuration.tags.length > 0) {
      details["Tags Filter"] = configuration.tags.join(", ");
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details["Count"] = "0";
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Listed At"] = payloadTimestamp;
    }

    const output = payload?.data as ListAnnotationsOutput | undefined;
    const annotations = output?.annotations ?? [];

    details["Count"] = String(annotations.length);

    if (annotations.length > 0 && annotations[0].text) {
      details["Latest"] =
        annotations[0].text.length > 60 ? annotations[0].text.substring(0, 60) + "..." : annotations[0].text;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(context: ComponentBaseContext): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = context.node.configuration as ListAnnotationsConfiguration | undefined;

  if (configuration?.tags && configuration.tags.length > 0) {
    metadata.push({ icon: "tag", label: `Tags: ${configuration.tags.join(", ")}` });
  }

  if (configuration?.dashboardUID) {
    metadata.push({
      icon: "layout-dashboard",
      label: createElement(GrafanaDashboardMetadataLabel, {
        organizationId: context.organizationId,
        integrationId: context.integrationId,
        dashboardUid: configuration.dashboardUID,
      }),
    });
  }

  return metadata;
}
