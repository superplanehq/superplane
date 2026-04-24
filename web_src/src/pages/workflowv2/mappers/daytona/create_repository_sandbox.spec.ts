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
  it("exposes core sandbox metadata for in-flight runs", () => {
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

  it("freezes elapsed at bootstrap.finishedAt when the phase ended", () => {
    const node = buildNode();
    const started = new Date("2026-04-23T10:00:00Z").toISOString();
    const finished = new Date("2026-04-23T10:04:30Z").toISOString();

    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        stage: "bootstrapping",
        sandboxStartedAt: started,
        timeout: 300,
        bootstrap: { finishedAt: finished },
      }),
    };

    const details = createRepositorySandboxMapper.getExecutionDetails(ctx);
    // Elapsed is deterministic when bootstrap.finishedAt is present,
    // so the label should not depend on "now".
    expect(details.Elapsed).toContain("4m");
    expect(details.Elapsed).toContain("30s");
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
