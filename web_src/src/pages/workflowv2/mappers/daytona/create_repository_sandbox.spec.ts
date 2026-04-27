import { describe, expect, it } from "vitest";

import { createRepositorySandboxMapper } from "./create_repository_sandbox";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(): NodeInfo {
  return {
    id: "node-1",
    name: "Daytona",
    componentName: "daytona.createRepositorySandbox",
    isCollapsed: false,
    configuration: {},
    metadata: {},
  };
}

function buildExecution(metadata: Record<string, unknown>): ExecutionInfo {
  const now = new Date().toISOString();
  return {
    id: "exec-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata,
    configuration: {},
    rootEvent: undefined,
    outputs: undefined,
  };
}

describe("createRepositorySandboxMapper.getExecutionDetails", () => {
  it("exposes core sandbox metadata and elapsed/timeout once bootstrap has started", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        stage: "bootstrapping",
        sandboxId: "sandbox-123",
        repository: "https://github.com/superplanehq/superplane.git",
        directory: "/home/daytona/superplane",
        sandboxStartedAt: new Date().toISOString(),
        timeout: 300,
        bootstrap: { startedAt: new Date().toISOString() },
      }),
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    expect(details).toMatchObject({
      Step: "bootstrapping",
      "Sandbox ID": "sandbox-123",
      Repository: "https://github.com/superplanehq/superplane.git",
      Directory: "/home/daytona/superplane",
    });
    expect(details.Elapsed).toMatch(/\//); // "<elapsed> / <timeout>"
  });

  it("shows plain elapsed (no timeout) before bootstrap has started", () => {
    // Pre-bootstrap stages must not display the bootstrap timeout against
    // sandbox creation time — that comparison is misleading because the
    // bootstrap deadline is anchored at bootstrap.startedAt on the backend.
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        stage: "preparingSandbox",
        sandboxStartedAt: new Date(Date.now() - 5000).toISOString(),
        timeout: 300,
      }),
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    expect(details.Elapsed).toBeDefined();
    expect(details.Elapsed).not.toMatch(/\//);
  });

  it("surfaces bootstrap log when the backend has captured output", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        stage: "bootstrapping",
        sandboxStartedAt: new Date().toISOString(),
        timeout: 300,
        bootstrap: {
          log: "installing deps...\nrunning tests...",
        },
      }),
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    expect(details["Bootstrap log"]).toBe("installing deps...\nrunning tests...");
  });

  it("freezes bootstrap elapsed between bootstrap.startedAt and bootstrap.finishedAt", () => {
    const node = buildNode();
    const sandboxStarted = new Date("2026-04-23T09:55:00Z").toISOString();
    const bootstrapStarted = new Date("2026-04-23T10:00:00Z").toISOString();
    const bootstrapFinished = new Date("2026-04-23T10:04:30Z").toISOString();

    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        stage: "bootstrapping",
        sandboxStartedAt: sandboxStarted,
        timeout: 300,
        bootstrap: { startedAt: bootstrapStarted, finishedAt: bootstrapFinished },
      }),
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    // Elapsed must measure from bootstrap.startedAt, NOT sandboxStartedAt;
    // otherwise the 5-minute pre-bootstrap gap would inflate the figure to
    // ~9m30s and exceed the 5m timeout prematurely.
    expect(details.Elapsed).toContain("4m");
    expect(details.Elapsed).toContain("30s");
    expect(details.Elapsed).not.toContain("9m");
  });

  it("freezes elapsed at execution.updatedAt for non-bootstrap finishes", () => {
    // For a preparingSandbox-stage timeout there is no bootstrap.finishedAt,
    // but the execution still reaches STATE_FINISHED. The elapsed counter
    // must freeze at execution.updatedAt instead of ticking forward forever.
    const node = buildNode();
    const started = new Date("2026-04-23T10:00:00Z").toISOString();
    const finished = new Date("2026-04-23T10:00:45Z").toISOString();

    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: {
        ...buildExecution({
          stage: "preparingSandbox",
          sandboxStartedAt: started,
          timeout: 60,
        }),
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        updatedAt: finished,
      },
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    expect(details.Elapsed).toContain("45s");
  });

  it("omits elapsed when sandboxStartedAt is missing", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({ stage: "preparingSandbox" }),
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    expect(details.Elapsed).toBeUndefined();
  });
});
