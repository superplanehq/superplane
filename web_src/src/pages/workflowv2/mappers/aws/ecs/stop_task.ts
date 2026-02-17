import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsSubtitle } from "./common";

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
  props(context) {
    return buildEcsComponentProps(context, stopTaskMetadataList(context.node));
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
    return ecsSubtitle(context);
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
