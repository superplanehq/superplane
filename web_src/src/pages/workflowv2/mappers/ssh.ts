import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "./types";
import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { getTriggerRenderer } from ".";
import { formatTimeAgo } from "@/utils/date";

const SSH_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
};

const sshStateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const metadata = execution.metadata as Record<string, unknown> | undefined;
    const result = metadata?.result as { exitCode?: number; ExitCode?: number } | undefined;
    const code = result?.exitCode ?? result?.ExitCode;
    if (code === 0) {
      return "success";
    }
    return "failed";
  }

  return "failed";
};

export const SSH_STATE_REGISTRY: EventStateRegistry = {
  stateMap: SSH_STATE_MAP,
  getState: sshStateFunction,
};

type SSHConfiguration = {
  host: string;
  port?: number;
  username: string;
  command: string;
  authMethod?: string;
};

export const sshMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      iconSlug: context.componentDefinition.icon || "terminal",
      iconColor: getColorClass("black"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: context.lastExecutions[0]
        ? getSSHEventSections(context.nodes, context.lastExecutions[0], sshStateFunction)
        : undefined,
      includeEmptyState: !context.lastExecutions[0],
      metadata: getSSHMetadataList(context.node),
      eventStateMap: SSH_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = context.execution.metadata as Record<string, unknown> | undefined;
    const result = metadata?.result as { stdout?: string; stderr?: string; exitCode?: number } | undefined;
    const host = metadata?.host as string | undefined;
    const port = metadata?.port as number | undefined;
    // Backend sends username as "user" in execution metadata (ExecutionMetadata.User json:"user")
    const username = (metadata?.user ?? metadata?.username) as string | undefined;
    if (host) {
      const portSuffix = port && port !== 22 ? `:${port}` : "";
      details["Host"] = `${username || "user"}@${host}${portSuffix}`;
    }

    if (context.execution.createdAt) {
      details["Started at"] = new Date(context.execution.createdAt).toLocaleString();
    }
    if (context.execution.updatedAt && context.execution.state === "STATE_FINISHED") {
      details["Finished at"] = new Date(context.execution.updatedAt).toLocaleString();
    }

    // Show connection retry progress
    const retryAttempt = typeof metadata?.attempt === "number" ? metadata.attempt : 0;
    const retryConfig = (
      context.node.configuration as SSHConfiguration & { connectionRetry?: { enabled?: boolean; retries?: number } }
    )?.connectionRetry;
    if (retryConfig?.enabled && retryAttempt > 0) {
      details["Connection retry"] = `${retryAttempt} / ${retryConfig.retries ?? "?"}`;
    }

    if (result?.exitCode !== undefined) {
      details["Exit code"] = String(result.exitCode);
    }
    if (result?.stdout !== undefined && result.stdout !== "") {
      details["Stdout"] = result.stdout;
    }
    if (result?.stderr !== undefined && result.stderr !== "") {
      details["Stderr"] = result.stderr;
    }
    if (context.execution.resultMessage) {
      details["Error"] = context.execution.resultMessage;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const state = sshStateFunction(context.execution);

    if (state === "running" && context.execution.createdAt) {
      const startTime = new Date(context.execution.createdAt);
      const now = new Date();
      const durationMs = now.getTime() - startTime.getTime();
      if (durationMs < 60000) {
        return `Running for ${Math.floor(durationMs / 1000)}s`;
      }
      return `Running for ${Math.floor(durationMs / 60000)}m`;
    }

    if (state === "success" || state === "failed") {
      const metadata = context.execution.metadata as Record<string, unknown> | undefined;
      const result = metadata?.result as { exitCode?: number } | undefined;
      const exitStr = result?.exitCode !== undefined ? `Exit ${result.exitCode}` : "";
      const timeAgo = context.execution.updatedAt ? formatTimeAgo(new Date(context.execution.updatedAt)) : "";
      if (exitStr && timeAgo) {
        return `${exitStr} · ${timeAgo}`;
      }
      if (timeAgo) return timeAgo;
    }

    if (context.execution.updatedAt) {
      return formatTimeAgo(new Date(context.execution.updatedAt));
    }
    return "";
  },
};

function getSSHMetadataList(node: NodeInfo): Array<{ icon: string; label: string }> {
  const config = node.configuration as SSHConfiguration;
  const metadata: Array<{ icon: string; label: string }> = [];

  if (config?.host) {
    const port = config.port && config.port !== 22 ? `:${config.port}` : "";
    metadata.push({
      icon: "server",
      label: `${config.username || "user"}@${config.host}${port}`,
    });
  }
  if (config?.command) {
    const cmd = config.command.length > 40 ? config.command.slice(0, 40) + "…" : config.command;
    metadata.push({
      icon: "terminal",
      label: cmd,
    });
  }

  return metadata;
}

function getSSHEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  stateFunction: (e: ExecutionInfo) => EventState,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const generateEventSubtitle = (): string => {
    const state = stateFunction(execution);
    if (state === "running" && execution.createdAt) {
      const startTime = new Date(execution.createdAt);
      const now = new Date();
      const durationMs = now.getTime() - startTime.getTime();
      return durationMs < 60000
        ? `Running for ${Math.floor(durationMs / 1000)}s`
        : `Running for ${Math.floor(durationMs / 60000)}m`;
    }
    if (state === "success" || state === "failed") {
      const metadata = execution.metadata as Record<string, unknown> | undefined;
      const result = metadata?.result as { exitCode?: number } | undefined;
      const exitStr = result?.exitCode !== undefined ? `Exit ${result.exitCode}` : "";
      const timeAgo = execution.updatedAt ? formatTimeAgo(new Date(execution.updatedAt)) : "";
      if (exitStr && timeAgo) return `${exitStr} · ${timeAgo}`;
      if (timeAgo) return timeAgo;
    }
    if (execution.updatedAt) {
      return formatTimeAgo(new Date(execution.updatedAt));
    }
    return "";
  };

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: generateEventSubtitle(),
      eventState: stateFunction(execution),
      eventId: execution.rootEvent!.id!,
      showAutomaticTime: stateFunction(execution) === "running",
    },
  ];
}
