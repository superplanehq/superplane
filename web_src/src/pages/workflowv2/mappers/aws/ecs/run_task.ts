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

interface RunTaskConfiguration {
  region?: string;
  cluster?: string;
  taskDefinition?: string;
  count?: number;
  launchType?: string;
}

interface EcsFailure {
  arn?: string;
  reason?: string;
  detail?: string;
}

interface EcsTask {
  taskArn?: string;
  clusterArn?: string;
  taskDefinitionArn?: string;
  lastStatus?: string;
  desiredStatus?: string;
  launchType?: string;
  platformVersion?: string;
  group?: string;
  startedBy?: string;
}

interface RunTaskOutput {
  tasks?: EcsTask[];
  failures?: EcsFailure[];
}

export const runTaskMapper: ComponentBaseMapper = {
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
      eventSections: lastExecution ? runTaskEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: runTaskMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as RunTaskOutput | undefined;

    if (!data) {
      return {};
    }

    const firstTask = data.tasks?.[0];
    return {
      "Started At": stringOrDash(
        context.execution.updatedAt ? new Date(context.execution.updatedAt).toLocaleString() : "-",
      ),
      "Tasks Started": String(data.tasks?.length || 0),
      Failures: String(data.failures?.length || 0),
      "Task ARN": stringOrDash(firstTask?.taskArn),
      "Task Definition": stringOrDash(firstTask?.taskDefinitionArn),
      "Cluster ARN": stringOrDash(firstTask?.clusterArn),
      "Last Status": stringOrDash(firstTask?.lastStatus),
      "Desired Status": stringOrDash(firstTask?.desiredStatus),
      "Launch Type": stringOrDash(firstTask?.launchType),
      "Platform Version": stringOrDash(firstTask?.platformVersion),
      Group: stringOrDash(firstTask?.group),
      "Started By": stringOrDash(firstTask?.startedBy),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function runTaskMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as RunTaskConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }
  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.taskDefinition) {
    items.push({ icon: "package", label: config.taskDefinition });
  }
  if (config?.count && config.count > 1) {
    items.push({ icon: "hash", label: `count: ${config.count}` });
  }
  if (config?.launchType) {
    if (config.launchType !== "AUTO") {
      items.push({ icon: "rocket", label: config.launchType });
    }
  }

  return items;
}

function runTaskEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
