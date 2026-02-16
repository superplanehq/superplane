import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsEcsIcon from "@/assets/icons/integrations/aws.ecs.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface StopTaskConfiguration {
  region?: string;
  cluster?: string;
  task?: string;
}

interface EcsTask {
  taskArn?: string;
  clusterArn?: string;
  taskDefinitionArn?: string;
  lastStatus?: string;
  desiredStatus?: string;
  stoppedReason?: string;
  launchType?: string;
  platformVersion?: string;
  group?: string;
  startedBy?: string;
}

interface StopTaskOutput {
  task?: EcsTask;
}

export const stopTaskMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: awsEcsIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? stopTaskEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: stopTaskMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as StopTaskOutput | undefined;
    const task = data?.task;

    if (!task) {
      return {};
    }

    return {
      "Stopped At": stringOrDash(
        context.execution.updatedAt ? new Date(context.execution.updatedAt).toLocaleString() : "-",
      ),
      "Task ARN": stringOrDash(task.taskArn),
      "Task Definition": stringOrDash(task.taskDefinitionArn),
      "Cluster ARN": stringOrDash(task.clusterArn),
      "Last Status": stringOrDash(task.lastStatus),
      "Desired Status": stringOrDash(task.desiredStatus),
      "Stopped Reason": stringOrDash(task.stoppedReason),
      "Launch Type": stringOrDash(task.launchType),
      "Platform Version": stringOrDash(task.platformVersion),
      Group: stringOrDash(task.group),
      "Started By": stringOrDash(task.startedBy),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function stopTaskMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as StopTaskConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }
  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.task) {
    items.push({ icon: "square", label: config.task });
  }

  return items;
}

function stopTaskEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
