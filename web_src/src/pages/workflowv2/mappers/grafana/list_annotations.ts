import type { ComponentBaseProps } from "@/pages/workflowv2/mappers/types";
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
import type { AnnotationNodeMetadata, ListAnnotationsConfiguration, ListAnnotationsOutput } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";
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
      eventSections: lastExecution ? baseEventSections(lastExecution, componentName) : undefined,
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

    addConfiguredFilters(details, configuration);

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      addConfiguredRange(details, configuration);
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
    const outputFrom = formatTimestamp(output?.from);
    const outputTo = formatTimestamp(output?.to);

    addResolvedRange(details, configuration, outputFrom, outputTo);
    details["Count"] = String(annotations.length);
    addLatestAnnotation(details, annotations);

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
  const nodeMetadata = context.node.metadata as AnnotationNodeMetadata | undefined;

  if (configuration?.tags && configuration.tags.length > 0) {
    metadata.push({ icon: "tag", label: `Tags: ${configuration.tags.join(", ")}` });
  }

  if (configuration?.text) {
    const preview = configuration.text.length > 50 ? configuration.text.substring(0, 50) + "..." : configuration.text;
    metadata.push({ icon: "search", label: `Text: ${preview}` });
  }

  const dashboardTitle = nodeMetadata?.dashboardTitle || configuration?.dashboard;
  if (dashboardTitle) {
    metadata.push({ icon: "layout-dashboard", label: `Dashboard: ${dashboardTitle}` });
  }

  return metadata;
}

function addConfiguredFilters(
  details: Record<string, string>,
  configuration: ListAnnotationsConfiguration | undefined,
) {
  if (configuration?.tags && configuration.tags.length > 0) {
    details["Tags Filter"] = configuration.tags.join(", ");
  }

  if (configuration?.text) {
    details["Text Filter"] = truncateText(configuration.text, 80);
  }
}

function addConfiguredRange(details: Record<string, string>, configuration: ListAnnotationsConfiguration | undefined) {
  if (configuration?.from) {
    details["From"] = configuration.from;
  }

  if (configuration?.to) {
    details["To"] = configuration.to;
  }
}

function addResolvedRange(
  details: Record<string, string>,
  configuration: ListAnnotationsConfiguration | undefined,
  outputFrom: string,
  outputTo: string,
) {
  details["From"] = outputFrom !== "-" ? outputFrom : configuration?.from || "";
  details["To"] = outputTo !== "-" ? outputTo : configuration?.to || "";

  if (details["From"] === "") {
    delete details["From"];
  }

  if (details["To"] === "") {
    delete details["To"];
  }
}

function addLatestAnnotation(
  details: Record<string, string>,
  annotations: ListAnnotationsOutput["annotations"] | undefined,
) {
  const latestText = annotations?.[0]?.text;
  if (!latestText) {
    return;
  }

  details["Latest"] = truncateText(latestText, 60);
}

function truncateText(value: string, maxLength: number) {
  return value.length > maxLength ? `${value.substring(0, maxLength)}...` : value;
}
