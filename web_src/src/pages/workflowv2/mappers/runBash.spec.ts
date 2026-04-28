import { describe, expect, it } from "vitest";

import { runBashMapper, RUN_BASH_STATE_REGISTRY } from "./runBash";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "./types";

function buildNode(): NodeInfo {
  return {
    id: "node-1",
    name: "Run Bash",
    componentName: "run-bash",
    isCollapsed: false,
    configuration: {},
    metadata: {},
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "run-bash.result",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution({
  metadata = {},
  outputs,
  state = "STATE_FINISHED",
  result = "RESULT_PASSED",
  resultReason = "RESULT_REASON_OK",
  resultMessage = "",
  updatedAt = new Date().toISOString(),
}: {
  metadata?: Record<string, unknown>;
  outputs?: { success?: OutputPayload[]; failed?: OutputPayload[] };
  state?: ExecutionInfo["state"];
  result?: ExecutionInfo["result"];
  resultReason?: ExecutionInfo["resultReason"];
  resultMessage?: string;
  updatedAt?: string;
}): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt,
    state,
    result,
    resultReason,
    resultMessage,
    metadata,
    configuration: {},
    rootEvent: undefined,
    outputs,
  };
}

describe("RUN_BASH_STATE_REGISTRY", () => {
  it("maps failed output payloads to failed", () => {
    const execution = buildExecution({
      outputs: {
        failed: [buildOutput({ command: { exitCode: 1, status: "FAILED" } })],
      },
    });

    expect(RUN_BASH_STATE_REGISTRY.getState(execution)).toBe("failed");
  });

  it("maps running executions to running", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      outputs: {},
    });

    expect(RUN_BASH_STATE_REGISTRY.getState(execution)).toBe("running");
  });

  it("maps error results to error", () => {
    const execution = buildExecution({
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: "backend failed",
    });

    expect(RUN_BASH_STATE_REGISTRY.getState(execution)).toBe("error");
  });
});

describe("runBashMapper", () => {
  it("renders execution details from command payload", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [
            buildOutput({
              command: {
                exitCode: 0,
                status: "SUCCEEDED",
                buildId: "build-1",
                logUrl: "https://logs",
                source: { repository: "github.com/example/app", commitSha: "abc123" },
                stdout: "ok",
              },
            }),
          ],
        },
      }),
    };

    expect(runBashMapper.getExecutionDetails(ctx)).toMatchObject({
      Repository: "github.com/example/app",
      Commit: "abc123",
      "Exit code": "0",
      "CodeBuild status": "SUCCEEDED",
      "Build ID": "build-1",
      Logs: "https://logs",
      Stdout: "ok",
    });
  });

  it("renders exit code in subtitle", () => {
    const node = buildNode();
    const ctx: SubtitleContext = {
      node,
      execution: buildExecution({
        outputs: {
          failed: [buildOutput({ command: { exitCode: 2 } })],
        },
      }),
    };

    const subtitle = runBashMapper.subtitle(ctx);
    expect(typeof subtitle).not.toBe("string");
  });
});
