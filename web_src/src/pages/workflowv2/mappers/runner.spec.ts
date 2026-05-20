import { describe, expect, it } from "vitest";

import { runnerConfigurationDetails, runnerMapper } from "./runner";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "./types";

function buildRunnerNode(configuration: Record<string, unknown>): NodeInfo {
  return {
    id: "node-runner-1",
    name: "Runner",
    componentName: "runner",
    isCollapsed: false,
    configuration,
    metadata: {},
  };
}

function buildExecution(overrides: {
  outputs?: { passed?: OutputPayload[]; failed?: OutputPayload[] };
  state?: ExecutionInfo["state"];
  result?: ExecutionInfo["result"];
  createdAt?: string;
}): ExecutionInfo {
  const now = overrides.createdAt ?? new Date().toISOString();
  return {
    id: "exec-1",
    createdAt: now,
    state: overrides.state ?? "STATE_FINISHED",
    result: overrides.result ?? "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs: overrides.outputs,
  } as ExecutionInfo;
}

describe("runnerConfigurationDetails", () => {
  it.each([
    [
      "host — minimal",
      { execution_mode: "host", commands: "echo hi", execution_timeout_seconds: 0 },
      {
        "Execution mode": "Host",
        "Timeout (seconds)": "Broker default (0)",
      },
    ],
    [
      "host — ignores stray docker_image (no Container image row)",
      {
        execution_mode: "host",
        commands: "echo hi",
        docker_image: "should-not-show:tag",
        docker_image_preset: "debian:bookworm-slim",
        execution_timeout_seconds: 0,
      },
      {
        "Execution mode": "Host",
        "Timeout (seconds)": "Broker default (0)",
      },
    ],
    [
      "docker — quick pick",
      {
        execution_mode: "docker",
        commands: "uname -a",
        docker_image_preset: "debian:bookworm-slim",
        execution_timeout_seconds: 0,
      },
      {
        "Execution mode": "Docker",
        "Container image": "debian:bookworm-slim",
        "Timeout (seconds)": "Broker default (0)",
      },
    ],
    [
      "docker — custom image",
      {
        execution_mode: "docker",
        commands: "date",
        docker_image_preset: "custom",
        docker_image: "registry.example.com/app:1.2.3",
        execution_timeout_seconds: 90,
      },
      {
        "Execution mode": "Docker",
        "Container image": "registry.example.com/app:1.2.3",
        "Timeout (seconds)": "90",
      },
    ],
    [
      "docker — legacy (no preset, only docker_image)",
      {
        execution_mode: "docker",
        commands: "true",
        docker_image: "alpine:3.20",
        execution_timeout_seconds: 0,
      },
      {
        "Execution mode": "Docker",
        "Container image": "alpine:3.20",
        "Timeout (seconds)": "Broker default (0)",
      },
    ],
    [
      "timeout as string zero",
      { execution_mode: "host", commands: "x", execution_timeout_seconds: "0" },
      {
        "Execution mode": "Host",
        "Timeout (seconds)": "Broker default (0)",
      },
    ],
    [
      "timeout as string non-zero",
      { execution_mode: "host", commands: "x", execution_timeout_seconds: "  45  " },
      {
        "Execution mode": "Host",
        "Timeout (seconds)": "45",
      },
    ],
    ["non-object configuration", null, {}],
  ] as const)("case: %s", (_label, configuration, expected) => {
    expect(runnerConfigurationDetails(configuration)).toEqual(expected);
  });

  it("omits Timeout when execution_timeout_seconds is absent", () => {
    expect(
      runnerConfigurationDetails({
        execution_mode: "host",
        commands: "echo",
      }),
    ).toEqual({
      "Execution mode": "Host",
    });
  });
});

describe("runnerMapper.getExecutionDetails", () => {
  it("merges configuration details with finished payload fields", () => {
    const node = buildRunnerNode({
      execution_mode: "docker",
      docker_image_preset: "python:3.12-slim",
      commands: "python -V",
      execution_timeout_seconds: 0,
    });
    const execution = buildExecution({
      outputs: {
        passed: [
          {
            type: "runner.finished",
            timestamp: new Date().toISOString(),
            data: { status: "succeeded", exit_code: 0 },
          },
        ],
      },
    });
    const ctx: ExecutionDetailsContext = { nodes: [node], node, execution };

    expect(runnerMapper.getExecutionDetails(ctx)).toEqual({
      "Execution mode": "Docker",
      "Container image": "python:3.12-slim",
      "Timeout (seconds)": "Broker default (0)",
      Status: "succeeded",
      "Exit code": "0",
    });
  });

  it("returns configuration-only details when there is no output payload", () => {
    const node = buildRunnerNode({ execution_mode: "host", commands: "id", execution_timeout_seconds: 0 });
    const execution = buildExecution({ outputs: undefined });
    const ctx: ExecutionDetailsContext = { nodes: [node], node, execution };

    expect(runnerMapper.getExecutionDetails(ctx)).toEqual({
      "Execution mode": "Host",
      "Timeout (seconds)": "Broker default (0)",
    });
  });
});
