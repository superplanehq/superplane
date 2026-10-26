import notionIcon from "@/assets/icons/integrations/notion.svg";
import { getBackgroundColorClass } from "@/lib/colors";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { formatTimestamp } from "../eventDisplay";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionInfo, OutputPayload } from "../types";

export function notionComponentProps(
  context: ComponentBaseContext,
  metadata: MetadataItem[],
  subtitle: ComponentBaseMapper["subtitle"],
): ComponentBaseProps {
  const execution = context.lastExecutions[0];
  const componentName = context.componentDefinition.name || context.node.componentName;
  return {
    iconSrc: notionIcon,
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition.label || "Unnamed component",
    eventSections: execution ? eventSections(context, execution, componentName, subtitle) : undefined,
    metadata,
    includeEmptyState: !execution,
    eventStateMap: getStateMap(componentName),
  };
}

function eventSections(
  context: ComponentBaseContext,
  execution: ExecutionInfo,
  componentName: string,
  subtitle: ComponentBaseMapper["subtitle"],
): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];
  const trigger = context.nodes.find((node) => node.id === rootEvent.nodeId);
  if (!trigger?.componentName) return [];

  const { title } = getTriggerRenderer(trigger.componentName).getTitleAndSubtitle({ event: rootEvent });
  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: subtitle({ node: context.node, execution }),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}

export function notionOutput<T>(execution: ExecutionInfo): T | undefined {
  const outputs = execution.outputs?.default;
  if (!Array.isArray(outputs)) return undefined;
  return (outputs[0] as OutputPayload | undefined)?.data as T | undefined;
}

export function notionExecutionDetails(execution: ExecutionInfo): Record<string, string> {
  return { "Started At": formatTimestamp(execution.createdAt) };
}
