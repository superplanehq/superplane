import { isValidElement, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../types";

function makeNode(componentName: string): NodeInfo {
  return {
    id: `${componentName}-node`,
    name: componentName,
    componentName: `daytona.${componentName}`,
    isCollapsed: false,
    configuration: {},
    metadata: {},
  };
}

function makeExecution(overrides: Partial<ExecutionInfo> = {}): ExecutionInfo {
  const now = new Date().toISOString();

  return {
    id: "exec-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_FINISHED" as ExecutionInfo["state"],
    result: "RESULT_PASSED" as ExecutionInfo["result"],
    resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs: undefined,
    ...overrides,
  };
}

function makeExecutionContext(node: NodeInfo, execution: ExecutionInfo = makeExecution()): ExecutionDetailsContext {
  return {
    nodes: [node],
    node,
    execution,
  };
}

function makeSubtitleContext(node: NodeInfo, execution: ExecutionInfo): SubtitleContext {
  return { node, execution };
}

function payloadOutputs(payload: Partial<OutputPayload>): { default: OutputPayload[] } {
  return {
    default: [
      {
        type: payload.type ?? "json",
        timestamp: payload.timestamp ?? new Date().toISOString(),
        data: payload.data,
      },
    ],
  };
}

describe("daytona baseMapper.getExecutionDetails", () => {
  it("returns 'Pending…' when the execution has no outputs", () => {
    const node = makeNode("executeCommand");
    const details = baseMapper.getExecutionDetails(makeExecutionContext(node));

    expect(details.Response).toBe("Pending…");
  });

  it("returns 'Pending…' when the payload exists but has no data", () => {
    const node = makeNode("executeCommand");
    const execution = makeExecution({ outputs: payloadOutputs({ data: undefined }) });
    const details = baseMapper.getExecutionDetails(makeExecutionContext(node, execution));

    expect(details.Response).toBe("Pending…");
  });

  it("returns 'Pending…' while the execution is still running (regression for issue #3261)", () => {
    const node = makeNode("executeCommand");
    const execution = makeExecution({ state: "STATE_STARTED" as ExecutionInfo["state"], outputs: undefined });
    const details = baseMapper.getExecutionDetails(makeExecutionContext(node, execution));

    expect(details.Response).toBe("Pending…");
  });

  it("projects getPreviewUrl payload fields into structured details", () => {
    const node = makeNode("getPreviewUrl");
    const execution = makeExecution({
      outputs: payloadOutputs({
        data: {
          sandbox: "sbx-123",
          port: 8080,
          signed: true,
          expiresInSeconds: 3600,
          token: "tkn-abc",
          url: "https://preview.example/sbx-123",
        },
      }),
    });

    const details = baseMapper.getExecutionDetails(makeExecutionContext(node, execution));

    expect(details["Sandbox"]).toBe("sbx-123");
    expect(details["Port"]).toBe("8080");
    expect(details["Signed URL"]).toBe("true");
    expect(details["Expires In Seconds"]).toBe("3600");
    expect(details["Token"]).toBe("tkn-abc");
    expect(details["Preview URL"]).toBe("https://preview.example/sbx-123");
    expect(details["Executed At"]).toBeDefined();
    expect(details["Response"]).toContain("sbx-123");
  });

  it("includes 'Executed At' and a serialized JSON Response for arbitrary actions", () => {
    const node = makeNode("executeCommand");
    const execution = makeExecution({
      outputs: payloadOutputs({ data: { exitCode: 0, stdout: "ok" } }),
    });

    const details = baseMapper.getExecutionDetails(makeExecutionContext(node, execution));

    expect(details["Executed At"]).toBeDefined();
    const parsed = JSON.parse(details["Response"]);
    expect(parsed).toEqual({ exitCode: 0, stdout: "ok" });
  });
});

function fragmentChildren(node: unknown): unknown[] {
  if (!isValidElement(node)) return [];
  const children = (node as ReactElement<{ children?: unknown }>).props.children;
  return Array.isArray(children) ? children : [children];
}

describe("daytona baseMapper.subtitle", () => {
  it("renders 'exit code N' for executeCommand when the payload reports an exitCode", () => {
    const node = makeNode("executeCommand");
    const execution = makeExecution({ outputs: payloadOutputs({ data: { exitCode: 0 } }) });

    const result = baseMapper.subtitle(makeSubtitleContext(node, execution));

    expect(fragmentChildren(result)).toContain("exit code 0");
  });

  it("renders 'timed out' for executeCommand when the payload reports a timeout", () => {
    const node = makeNode("executeCommand");
    const execution = makeExecution({ outputs: payloadOutputs({ data: { timeout: true } }) });

    const result = baseMapper.subtitle(makeSubtitleContext(node, execution));

    expect(fragmentChildren(result)).toContain("timed out");
  });
});
