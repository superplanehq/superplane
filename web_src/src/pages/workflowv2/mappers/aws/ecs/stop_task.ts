import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsConsoleUrl, ecsSubtitle, MAX_METADATA_ITEMS, truncateForDisplay } from "./common";

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
    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      "Stopped At": timestamp,
    };
    if (task) {
      details["Task ARN"] = stringOrDash(task.taskArn);
      details["Last Status"] = stringOrDash(task.lastStatus);
      details["Stopped Reason"] = stringOrDash(task.stoppedReason);
      const region = task.clusterArn?.split(":")[2] ?? "";
      const cluster = task.clusterArn?.split("/").pop() ?? "";
      if (region && cluster && task.taskArn) {
        details["ECS Console"] = ecsConsoleUrl(region, cluster, undefined, task.taskArn);
      }
    }
    return details;
  },

  subtitle(context: SubtitleContext): string {
    return ecsSubtitle(context);
  },
};

function stopTaskMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as StopTaskConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.task) {
    items.push({ icon: "square", label: truncateForDisplay(config.task) });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
