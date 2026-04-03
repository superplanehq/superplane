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
import type { DeleteAnnotationConfiguration, DeleteAnnotationOutput } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";
import { AnnotationMetadataLabel } from "./AnnotationMetadataLabel";
import { baseEventSections } from "./base";

export const deleteAnnotationMapper: ComponentBaseMapper = {
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
    const details: Record<string, string> = {
      "Deleted At": formatTimestamp(context.execution.createdAt),
    };

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Deleted At"] = payloadTimestamp;
    }

    const output = payload?.data as DeleteAnnotationOutput | undefined;
    if (output != null && typeof output.id === "number") {
      details["Annotation ID"] = String(output.id);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(context: ComponentBaseContext): MetadataItem[] {
  const configuration = context.node.configuration as DeleteAnnotationConfiguration | undefined;
  if (!configuration?.annotationId?.trim()) {
    return [];
  }
  return [
    {
      icon: "bookmark",
      label: createElement(AnnotationMetadataLabel, {
        organizationId: context.organizationId,
        integrationId: context.integrationId,
        annotationId: configuration.annotationId.trim(),
      }),
    },
  ];
}
