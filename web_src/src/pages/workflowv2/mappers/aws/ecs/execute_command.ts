import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsSubtitle } from "./common";

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

    if (!command) {
      return {};
    }

    return {
      "Executed At": stringOrDash(
        context.execution.updatedAt ? new Date(context.execution.updatedAt).toLocaleString() : "-",
      ),
      "Task ARN": stringOrDash(command.taskArn),
      "Cluster ARN": stringOrDash(command.clusterArn),
      "Container ARN": stringOrDash(command.containerArn),
      Container: stringOrDash(command.containerName),
      Interactive: stringOrDash(command.interactive),
      "Session ID": stringOrDash(command.session?.sessionId),
      "Stream URL": stringOrDash(command.session?.streamUrl),
      "Session Token": command.session?.tokenValue ? "[redacted]" : "-",
    };
  },

  subtitle(context: SubtitleContext): string {
    return ecsSubtitle(context);
  },
};

function executeCommandMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as ExecuteCommandConfiguration | undefined;
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
  if (config?.container) {
    items.push({ icon: "package", label: config.container });
  }
  if (config?.interactive) {
    items.push({ icon: "terminal", label: "interactive" });
  }

  return items;
}
