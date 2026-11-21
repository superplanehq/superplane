import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import { runPipelineMapper } from "./run_pipeline";

describe("runPipelineMapper.getExecutionDetails", () => {
  it("maps all six details from a flat payload", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          success: [
            output({
              executionId: "execution-1",
              pipelineIdentifier: "deploy",
              status: "Success",
              planExecutionUrl: "https://example.test/execution-1",
              startedAt: "1700000000000",
              endedAt: "1700000001000",
            }),
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Execution ID": "execution-1",
      Pipeline: "deploy",
      Status: "Success",
      "Plan URL": "https://example.test/execution-1",
      "Started At": "1700000000000",
      "Ended At": "1700000001000",
    });
  });

  it.each([
    {
      name: "success before failed and default",
      success: { executionId: "success" },
      failed: { executionId: "failed" },
      expected: "success",
    },
    { name: "failed before default", success: undefined, failed: { executionId: "failed" }, expected: "failed" },
    { name: "default after empty channels", success: undefined, failed: undefined, expected: "default" },
    {
      name: "failed after a falsy success payload",
      success: false,
      failed: { executionId: "failed" },
      expected: "failed",
    },
    { name: "default after falsy success and failed payloads", success: "", failed: 0, expected: "default" },
    {
      name: "metadata after a truthy malformed success payload",
      success: "malformed",
      failed: { executionId: "failed" },
      expected: "metadata",
    },
  ])("selects $name", ({ success, failed, expected }) => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          success: [output(success)],
          failed: [output(failed)],
          default: [output({ executionId: "default" })],
        },
        metadata: { executionId: "metadata" },
      }),
    );

    expect(details).toEqual({ "Execution ID": expected });
  });

  it.each(["success", "failed", "default"])("inspects only the first item in the %s channel", (channel) => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: { [channel]: [output(null), output({ executionId: "second" })] },
        metadata: { executionId: "metadata" },
      }),
    );

    expect(details).toEqual({ "Execution ID": "metadata" });
  });

  it("uses nested object data instead of conflicting outer fields", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          success: [output({ executionId: "outer", status: "outer-status", data: { executionId: "nested" } })],
        },
        metadata: { status: "metadata-status" },
      }),
    );

    expect(details).toEqual({ "Execution ID": "nested" });
  });

  it.each([undefined, null, "malformed", 42, false])("retains outer fields for non-object nested data %j", (data) => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({ outputs: { success: [output({ executionId: "outer", data })] } }),
    );

    expect(details).toEqual({ "Execution ID": "outer" });
  });

  it("unwraps only one nested data level", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: { success: [output({ data: { executionId: "nested", data: { executionId: "deeper" } } })] },
      }),
    );

    expect(details).toEqual({ "Execution ID": "nested" });
  });

  it.each([undefined, null, "malformed", 42, false])("uses object metadata for non-object payload %j", (payload) => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: { success: [output(payload)] },
        metadata: { executionId: "metadata" },
      }),
    );

    expect(details).toEqual({ "Execution ID": "metadata" });
  });

  it("uses metadata when outputs are missing", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({ metadata: { executionId: "metadata" } }),
    );

    expect(details).toEqual({ "Execution ID": "metadata" });
  });

  it.each([
    { name: "empty object", payload: {} },
    { name: "array", payload: [] },
    { name: "nested empty object", payload: { data: {} } },
    { name: "nested array", payload: { data: [] } },
  ])("blocks metadata fallback for $name payload", ({ payload }) => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: { success: [output(payload)] },
        metadata: { executionId: "metadata" },
      }),
    );

    expect(details).toEqual({});
  });

  it.each([undefined, null, "malformed", 42, false])("returns no details for malformed metadata %j", (metadata) => {
    const details = runPipelineMapper.getExecutionDetails(buildDetailsContext({ metadata }));

    expect(details).toEqual({});
  });

  it("omits empty, null, missing, and falsy values without metadata backfill", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          success: [
            output({ executionId: "", pipelineIdentifier: null, status: "Success", startedAt: 0, endedAt: false }),
          ],
        },
        metadata: {
          executionId: "metadata",
          pipelineIdentifier: "metadata-pipeline",
          planExecutionUrl: "metadata-url",
        },
      }),
    );

    expect(details).toEqual({ Status: "Success" });
  });

  it("preserves truthy detail values without coercion or field validation", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({ outputs: { success: [output({ executionId: 42, status: true })] } }),
    );

    expect(details).toEqual({ "Execution ID": 42, Status: true });
  });
});

function output(data: unknown): OutputPayload {
  return { type: "harness.pipeline.completed", timestamp: "", data };
}

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Run Pipeline",
    componentName: "harness.runPipeline",
    isCollapsed: false,
    configuration: {},
  };

  return {
    nodes: [node],
    node,
    execution: {
      id: "execution-1",
      createdAt: "",
      updatedAt: "",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      ...execution,
    },
  };
}
