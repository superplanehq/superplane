import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

interface GetCaseConfiguration {
  caseId?: string;
}

interface GetCaseNodeMetadata {
  caseName?: string;
}

interface GetCaseOutputData {
  id?: string;
  title?: string;
  description?: string;
  status?: string;
  severity?: string;
  tags?: string[];
  version?: string;
  createdAt?: string;
  updatedAt?: string;
}

export const getCaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: elasticIcon,
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
    const outputs = context.execution.outputs as { default: OutputPayload[] };
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    if (!outputs?.default?.[0]?.data) {
      return details;
    }
    const doc = outputs.default[0].data as GetCaseOutputData;
    return { ...details, ...getDetailsForGetCase(doc) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetCaseConfiguration | undefined;
  const nodeMetadata = node.metadata as GetCaseNodeMetadata | undefined;

  const caseName = nodeMetadata?.caseName || configuration?.caseId;
  if (caseName) {
    metadata.push({ icon: "folder", label: caseName });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const triggerComponent = rootTriggerNode?.componentName ?? componentName;
  const rootTriggerRenderer = getTriggerRenderer(triggerComponent);
  const titleAndSubtitle = rootTriggerRenderer?.getTitleAndSubtitle({ event: execution.rootEvent });
  const title = titleAndSubtitle?.title ?? "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getDetailsForGetCase(doc: GetCaseOutputData): Record<string, string> {
  const details: Record<string, string> = {};

  if (doc?.id) {
    details["Case ID"] = String(doc.id);
  }

  if (doc?.title) {
    details["Title"] = doc.title;
  }

  if (doc?.status) {
    details["Status"] = doc.status;
  }

  if (doc?.severity) {
    details["Severity"] = doc.severity;
  }

  if (doc?.description) {
    const desc = doc.description;
    details["Description"] = desc.length > 100 ? desc.slice(0, 100) + "…" : desc;
  }

  return details;
}
