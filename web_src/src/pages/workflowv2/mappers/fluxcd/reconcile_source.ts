import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
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
import fluxcdIcon from "@/assets/icons/integrations/fluxcd.svg";
import { ReconcileSourceOutput } from "./types";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../utils";

export const reconcileSourceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: fluxcdIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Triggered At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    if (!outputs?.default?.[0]?.data) {
      return details;
    }

    const result = outputs.default[0].data as ReconcileSourceOutput;

    if (result.kind) {
      details["Kind"] = result.kind;
    }

    if (result.namespace) {
      details["Namespace"] = result.namespace;
    }

    if (result.name) {
      details["Name"] = result.name;
    }

    if (result.lastAppliedRevision) {
      details["Last Applied Revision"] = result.lastAppliedRevision;
    }

    if (result.resourceVersion) {
      details["Resource Version"] = result.resourceVersion;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "-";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { kind?: string; namespace?: string; name?: string } | undefined;

  if (configuration?.kind && configuration?.name) {
    metadata.push({ icon: "box", label: `${configuration.kind}/${configuration.name}` });
  }

  if (configuration?.namespace) {
    metadata.push({ icon: "folder", label: stringOrDash(configuration.namespace) });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const eventTitle = title || "Trigger event";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventTitle: eventTitle,
      eventSubtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : "-",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
