import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";
import type { NewRelicNRQLResultPayload, RunNRQLQueryConfiguration } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./utils";

export const runNRQLQueryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: newrelicIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (!outputs?.default?.[0]?.data) {
      details["Response"] = "No data returned";
      return details;
    }

    const result = outputs.default[0].data as NewRelicNRQLResultPayload;
    return { ...details, ...getDetailsForNRQLResult(result) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as RunNRQLQueryConfiguration | undefined;

  if (configuration?.query) {
    const truncated =
      configuration.query.length > 50 ? configuration.query.substring(0, 50) + "..." : configuration.query;
    metadata.push({ icon: "search", label: `Query: ${truncated}` });
  }

  return metadata;
}

function getDetailsForNRQLResult(result: NewRelicNRQLResultPayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (result?.query) {
    details["Query"] = result.query;
  }

  if (result?.results === undefined || result.results.length === 0) {
    details["Results"] = "No results found";
    return details;
  }

  details["Result Count"] = String(result.results.length);

  const firstResult = result.results[0];

  if (!firstResult || typeof firstResult !== "object") {
    details["Result"] = String(firstResult);
    return details;
  }

  const keys = Object.keys(firstResult);
  let allValuesEmpty = keys.length > 0;
  for (const key of keys.slice(0, 5)) {
    details[key] = String(firstResult[key]);
    if (firstResult[key] !== null && firstResult[key] !== undefined) {
      allValuesEmpty = false;
    }
  }

  if (allValuesEmpty) {
    details["Tip"] =
      "New Relic ingestion can take up to 60 seconds. If you expected data, try running the workflow again in a minute.";
  }

  return details;
}
