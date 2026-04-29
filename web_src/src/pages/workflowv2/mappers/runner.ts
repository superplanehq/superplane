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

const RUNNER_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
};

interface RunnerSource {
  repository?: string;
  ref?: string;
  commitSha?: string;
}

interface RunnerArtifact {
  name?: string;
  path?: string;
}

interface RunnerCommandPayload {
  command?: {
    exitCode?: number | string | null;
    status?: string;
    stdout?: string;
    stderr?: string;
    buildId?: string;
    buildArn?: string;
    logUrl?: string;
    source?: RunnerSource;
    artifacts?: RunnerArtifact[];
  };
}

interface RunnerMetadata {
  exitCode?: number | string | null;
  status?: string;
  buildId?: string;
  buildArn?: string;
  source?: RunnerSource;
  artifacts?: RunnerArtifact[];
  runtimeImage?: string;
  computeSize?: string;
  dockerEnabled?: boolean;
  logs?: {
    deepLink?: string;
  };
  output?: {
    stdout?: string;
    stderr?: string;
  };
}

interface RunnerConfiguration {
  source?: RunnerSource;
  runtimeImage?: string;
  computeSize?: string;
  timeout?: number;
  docker?: {
    enabled?: boolean;
  };
}

function getCommandResult(execution: ExecutionInfo): RunnerCommandPayload["command"] | undefined {
  const outputs = execution.outputs as { success?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
  const payload =
    (outputs?.failed?.[0]?.data as RunnerCommandPayload | undefined) ??
    (outputs?.success?.[0]?.data as RunnerCommandPayload | undefined);

  if (payload?.command) {
    return payload.command;
  }

  const metadata = execution.metadata as RunnerMetadata | undefined;
  if (!metadata) {
    return undefined;
  }

  return {
    exitCode: metadata.exitCode,
    status: metadata.status,
    stdout: metadata.output?.stdout,
    stderr: metadata.output?.stderr,
    buildId: metadata.buildId,
    buildArn: metadata.buildArn,
    logUrl: metadata.logs?.deepLink,
    source: metadata.source,
    artifacts: metadata.artifacts,
  };
}

function getExitCode(execution: ExecutionInfo): number | undefined {
  const code = getCommandResult(execution)?.exitCode;
  if (typeof code === "number") {
    return Number.isFinite(code) ? code : undefined;
  }
  if (typeof code === "string" && code.trim() !== "") {
    const parsed = Number(code);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

const runnerStateFunction = (execution: ExecutionInfo): EventState => {
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

    const exitCode = getExitCode(execution);
    if (exitCode === undefined || exitCode === 0) {
      return "success";
    }
    return "failed";
  }

  return "failed";
};

export const RUNNER_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUNNER_STATE_MAP,
  getState: runnerStateFunction,
};

export const runnerMapper: ComponentBaseMapper = {
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
        ? getRunnerEventSections(context.nodes, context.lastExecutions[0], runnerStateFunction)
        : undefined,
      includeEmptyState: !context.lastExecutions[0],
      metadata: getRunnerMetadataList(context.node),
      eventStateMap: RUNNER_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const result = getCommandResult(context.execution);

    if (context.execution.createdAt) {
      details["Started at"] = new Date(context.execution.createdAt).toLocaleString();
    }
    if (context.execution.updatedAt && context.execution.state === "STATE_FINISHED") {
      details["Finished at"] = new Date(context.execution.updatedAt).toLocaleString();
    }

    const source = result?.source;
    if (source?.repository) {
      details["Repository"] = source.repository;
    }
    if (source?.commitSha) {
      details["Commit"] = source.commitSha;
    }

    const exitCode = getExitCode(context.execution);
    if (exitCode !== undefined) {
      details["Exit code"] = String(exitCode);
    }
    if (result?.status) {
      details["Status"] = result.status;
    }
    if (result?.buildId) {
      details["Run ID"] = result.buildId;
    }
    if (result?.logUrl) {
      details["Logs"] = result.logUrl;
    }
    if (result?.artifacts?.length) {
      details["Artifacts"] = String(result.artifacts.length);
    }
    if (result?.stdout) {
      details["Stdout"] = result.stdout;
    }
    if (result?.stderr) {
      details["Stderr"] = result.stderr;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const state = runnerStateFunction(context.execution);

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
      const exitCode = getExitCode(context.execution);
      const exitStr = exitCode !== undefined ? `Exit ${exitCode}` : "";
      if (exitStr && context.execution.updatedAt) {
        return renderWithTimeAgo(exitStr, new Date(context.execution.updatedAt));
      }
    }

    if (context.execution.updatedAt) {
      return renderTimeAgo(new Date(context.execution.updatedAt));
    }
    return "";
  },
};

function getRunnerMetadataList(node: NodeInfo): Array<{ icon: string; label: string }> {
  const config = node.configuration as RunnerConfiguration | undefined;
  const metadata = node.metadata as RunnerMetadata | undefined;
  const items: Array<{ icon: string; label: string }> = [];

  const repository = metadata?.source?.repository || config?.source?.repository;
  if (repository) {
    items.push({ icon: "git-branch", label: repositoryLabel(repository) });
  }

  if (config?.docker?.enabled || metadata?.dockerEnabled) {
    items.push({ icon: "box", label: "Docker" });
  }

  const runtime = metadata?.runtimeImage || config?.runtimeImage;
  if (runtime && runtime !== "default") {
    items.push({ icon: "cpu", label: runtime });
  }

  return items.slice(0, 3);
}

function repositoryLabel(repository: string): string {
  return repository.replace(/^https?:\/\//, "").replace(/\.git$/, "");
}

function getRunnerEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  getState: (execution: ExecutionInfo) => EventState,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
