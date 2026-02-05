import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";

interface CreateOrUpdateContactConfiguration {
  email?: string;
  listIds?: string[];
}

interface CreateOrUpdateContactMetadata {
  email?: string;
}

export const createOrUpdateContactMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || context.node.componentName || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: sendgridIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? eventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs: contactSpecs(context.node),
      metadata: metadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    const failed = outputs?.failed?.[0]?.data as Record<string, unknown> | undefined;

    const updatedAt = formatTimestamp(context.execution.updatedAt, context.execution.createdAt);

    if (failed) {
      return {
        "Updated At": updatedAt,
        Status: stringOrDash(failed.statusCode),
        Error: stringOrDash(failed.error),
      };
    }

    return {
      "Updated At": updatedAt,
      Status: stringOrDash(result?.status),
      "Job ID": stringOrDash(result?.jobId),
      Email: stringOrDash(result?.email),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as CreateOrUpdateContactMetadata | undefined;
  const configuration = node.configuration as CreateOrUpdateContactConfiguration | undefined;

  const email = nodeMetadata?.email || configuration?.email;
  if (email) {
    metadata.push({ icon: "mail", label: `Email: ${email}` });
  }

  return metadata;
}

function contactSpecs(node: NodeInfo): ComponentBaseSpec[] | undefined {
  const configuration = node.configuration as CreateOrUpdateContactConfiguration | undefined;
  if (!configuration?.listIds?.length) {
    return undefined;
  }

  return [
    {
      title: "List",
      tooltipTitle: "Lists",
      values: configuration.listIds.map((listId) => ({
        badges: [
          {
            label: listId,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    },
  ];
}

function eventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  if (Array.isArray(value)) {
    return value.join(", ");
  }

  return String(value);
}

function formatTimestamp(updatedAt?: string, createdAt?: string): string {
  const timestamp = updatedAt || createdAt;
  if (!timestamp) {
    return "-";
  }

  return new Date(timestamp).toLocaleString();
}
