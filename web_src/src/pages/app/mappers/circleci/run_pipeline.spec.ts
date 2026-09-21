import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { runPipelineMapper } from "./run_pipeline";

const CREATED_AT = "2026-06-08T09:01:00Z";
const UPDATED_AT = "2026-06-08T09:05:00Z";

describe("runPipelineMapper.getExecutionDetails", () => {
  it("maps pipeline fields from the success output", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          success: [
            {
              type: "circleci.workflow.completed",
              timestamp: CREATED_AT,
              data: {
                pipeline: {
                  id: "pipe-1",
                  number: 42,
                  pipeline_url: "https://app.circleci.com/pipelines/github/org/repo/42",
                },
              },
            },
          ],
        },
      }),
    );

    expect(details["Pipeline Number"]).toBe("42");
    expect(details["Pipeline ID"]).toBe("pipe-1");
    expect(details["Pipeline URL"]).toBe("https://app.circleci.com/pipelines/github/org/repo/42");
    expect(details["Started At"]).toBeDefined();
    expect(details["Finished At"]).toBeDefined();
  });

  it("reads nested output data when present", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          failed: [
            {
              type: "circleci.workflow.completed",
              timestamp: CREATED_AT,
              data: {
                data: {
                  pipeline: { id: "nested-pipe", number: 7, pipeline_url: "https://example.test/7" },
                },
              },
            },
          ],
        },
      }),
    );

    expect(details["Pipeline ID"]).toBe("nested-pipe");
    expect(details["Pipeline Number"]).toBe("7");
    expect(details["Pipeline URL"]).toBe("https://example.test/7");
  });

  it("falls back to execution metadata when output data is missing", () => {
    const details = runPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {},
        metadata: {
          pipeline: { id: "meta-pipe", number: 9, pipeline_url: "https://example.test/9" },
        },
      }),
    );

    expect(details["Pipeline ID"]).toBe("meta-pipe");
    expect(details["Pipeline Number"]).toBe("9");
    expect(details["Pipeline URL"]).toBe("https://example.test/9");
  });

  it("returns timestamps only when pipeline data is absent", () => {
    const details = runPipelineMapper.getExecutionDetails(buildDetailsContext({ outputs: {} }));

    expect(details["Pipeline ID"]).toBeUndefined();
    expect(details["Started At"]).toBeDefined();
    expect(details["Finished At"]).toBeDefined();
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Run Pipeline",
    componentName: "circleci.runPipeline",
    isCollapsed: false,
    configuration: {},
  };

  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: CREATED_AT,
      updatedAt: UPDATED_AT,
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
