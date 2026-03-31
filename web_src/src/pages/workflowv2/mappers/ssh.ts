import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getColorClass } from "@/lib/colors";
import type React from "react";
import { getTriggerRenderer } from ".";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";

const SSH_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
};

type SSHResultPayload = {
  stdout?: string;
  stderr?: string;
  Stdout?: string;
  Stderr?: string;
  exitCode?: number | string;
  ExitCode?: number | string;
};

function extractSSHResultFromPayload(payload: unknown): SSHResultPayload | undefined {
  if (!payload || typeof payload !== "object") {
    return undefined;
  }

  return payload as SSHResultPayload;
}

function getSSHResult(execution: ExecutionInfo): SSHResultPayload | undefined {
  const outputs = execution.outputs as
    | { success?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
    | undefined;

  const payloadResult =
    extractSSHResultFromPayload(outputs?.failed?.[0]?.data) ??
    extractSSHResultFromPayload(outputs?.success?.[0]?.data) ??
    extractSSHResultFromPayload(outputs?.default?.[0]?.data);
  if (payloadResult) {
    return payloadResult;
  }

  const metadata = execution.metadata as Record<string, unknown> | undefined;
  return extractSSHResultFromPayload(metadata?.result);
}

function getSSHExitCode(execution: ExecutionInfo): number | undefined {
  const result = getSSHResult(execution);
  const code = result?.exitCode ?? result?.ExitCode;
  if (typeof code === "number") {
    return Number.isFinite(code) ? code : undefined;
  }
  if (typeof code === "string" && code.trim() !== "") {
    const parsed = Number(code);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

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
    const outputs = execution.outputs as { failed?: OutputPayload[] } | undefined;
    if (outputs?.failed?.length) {
      return "failed";
    }

    const code = getSSHExitCode(execution);
    if (code === 0) {
      return "success";
    }
    if (typeof code === "number") {
      return "failed";
    }
    return "success";
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
  commands?: string;
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
    const result = getSSHResult(context.execution);
    const host = metadata?.host as string | undefined;
    const port = metadata?.port as number | undefined;
    const username = metadata?.user as string | undefined;
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

    const exitCode = getSSHExitCode(context.execution);
    if (exitCode !== undefined) {
      details["Exit code"] = String(exitCode);
    }
    const stdout = result?.stdout ?? result?.Stdout;
    if (stdout !== undefined && stdout !== "") {
      details["Stdout"] = stdout;
    }
    const stderr = result?.stderr ?? result?.Stderr;
    if (stderr !== undefined && stderr !== "") {
      details["Stderr"] = stderr;
    }
    if (context.execution.resultMessage) {
      details["Error"] = context.execution.resultMessage;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
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
      const exitCode = getSSHExitCode(context.execution);
      const exitStr = exitCode !== undefined ? `Exit ${exitCode}` : "";
      if (exitStr && context.execution.updatedAt) {
        return renderWithTimeAgo(exitStr, new Date(context.execution.updatedAt));
      }
      if (context.execution.updatedAt) return renderTimeAgo(new Date(context.execution.updatedAt));
    }

    if (context.execution.updatedAt) {
      return renderTimeAgo(new Date(context.execution.updatedAt));
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
  if (config?.commands) {
    const oneline = config.commands
      .split("\n")
      .filter((l) => l.trim() !== "")
      .join(" && ");
    metadata.push({
      icon: "terminal",
      label: oneline,
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

  const generateEventSubtitle = (): string | React.ReactNode => {
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
      const exitCode = getSSHExitCode(execution);
      const exitStr = exitCode !== undefined ? `Exit ${exitCode}` : "";
      if (exitStr && execution.updatedAt) return renderWithTimeAgo(exitStr, new Date(execution.updatedAt));
      if (execution.updatedAt) return renderTimeAgo(new Date(execution.updatedAt));
    }
    if (execution.updatedAt) {
      return renderTimeAgo(new Date(execution.updatedAt));
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
