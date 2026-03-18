import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";
import { formatTimeAgo } from "@/utils/date";

interface GetCaseConfiguration {
  caseId?: string;
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
    if (!outputs?.default?.[0]?.data) {
      return {};
    }
    const doc = outputs.default[0].data as GetCaseOutputData;
    return getDetailsForGetCase(doc);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetCaseConfiguration | undefined;

  if (configuration?.caseId) {
    metadata.push({ icon: "hash", label: `Case: ${configuration.caseId}` });
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

  if (doc?.createdAt) {
    details["Created At"] = new Date(doc.createdAt).toLocaleString();
  }

  if (doc?.updatedAt) {
    details["Updated At"] = new Date(doc.updatedAt).toLocaleString();
  }

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
