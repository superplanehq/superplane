import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsConsoleUrl, ecsSubtitle, MAX_METADATA_ITEMS, truncateForDisplay } from "./common";

interface ExecuteCommandConfiguration {
  region?: string;
  cluster?: string;
  task?: string;
  container?: string;
  command?: string;
  interactive?: boolean;
}

interface ExecuteCommandSession {
  sessionId?: string;
  streamUrl?: string;
  tokenValue?: string;
}

interface ExecuteCommandResult {
  clusterArn?: string;
  containerArn?: string;
  containerName?: string;
  interactive?: boolean;
  session?: ExecuteCommandSession;
  taskArn?: string;
}

interface ExecuteCommandOutput {
  command?: ExecuteCommandResult;
}

export const executeCommandMapper: ComponentBaseMapper = {
  props(context) {
    return buildEcsComponentProps(context, executeCommandMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as ExecuteCommandOutput | undefined;
    const command = data?.command;
    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      "Executed At": timestamp,
    };
    if (command) {
      details["Task ARN"] = stringOrDash(command.taskArn);
      details["Container"] = stringOrDash(command.containerName);
      details["Interactive"] = stringOrDash(command.interactive);
      if (command.session?.streamUrl) {
        details["Stream URL"] = command.session.streamUrl;
      }
      const region = command.clusterArn?.split(":")[2] ?? "";
      const cluster = command.clusterArn?.split("/").pop() ?? "";
      if (region && cluster && command.taskArn) {
        details["ECS Console"] = ecsConsoleUrl(region, cluster, undefined, command.taskArn);
      }
    }
    return details;
  },

  subtitle(context: SubtitleContext): string {
    return ecsSubtitle(context);
  },
};

function executeCommandMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as ExecuteCommandConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.task) {
    items.push({ icon: "square", label: truncateForDisplay(config.task) });
  }
  if (config?.container) {
    items.push({ icon: "package", label: config.container });
  }
  if (items.length < MAX_METADATA_ITEMS && config?.interactive) {
    items.push({ icon: "terminal", label: "interactive" });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
