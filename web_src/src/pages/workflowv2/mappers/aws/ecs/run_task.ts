import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsConsoleUrl, ecsSubtitle, MAX_METADATA_ITEMS } from "./common";

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
  props(context) {
    return buildEcsComponentProps(context, runTaskMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as RunTaskOutput | undefined;
    const firstTask = data?.tasks?.[0];
    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      "Started At": timestamp,
    };
    if (data) {
      details["Tasks Started"] = String(data.tasks?.length ?? 0);
      if (data.failures?.length) {
        details["Failures"] = String(data.failures.length);
      }
      if (firstTask) {
        details["Task ARN"] = stringOrDash(firstTask.taskArn);
        details["Last Status"] = stringOrDash(firstTask.lastStatus);
        const region = firstTask.clusterArn?.split(":")[2] ?? "";
        const cluster = firstTask.clusterArn?.split("/").pop() ?? "";
        if (region && cluster && firstTask.taskArn) {
          details["ECS Console"] = ecsConsoleUrl(region, cluster, undefined, firstTask.taskArn);
        }
      }
    }
    return details;
  },

  subtitle(context: SubtitleContext): string {
    return ecsSubtitle(context);
  },
};

function runTaskMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as RunTaskConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.taskDefinition) {
    items.push({ icon: "package", label: config.taskDefinition });
  }
  if (config?.count && config.count > 1) {
    items.push({ icon: "hash", label: `count: ${config.count}` });
  }
  if (items.length < MAX_METADATA_ITEMS && config?.launchType && config.launchType !== "AUTO") {
    items.push({ icon: "rocket", label: config.launchType });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
