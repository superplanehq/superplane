import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getPipelineMapper } from "./get_pipeline";

describe("getPipelineMapper", () => {
  it("maps pipeline payload fields into execution details", () => {
    const details = getPipelineMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              data: {
                ppl_id: "ppl-1",
                name: "Initial Pipeline",
                wf_id: "wf-1",
                state: "done",
                result: "passed",
                result_reason: "test",
                branch_name: "main",
                commit_sha: "abc123",
                commit_message: "feat: add new feature",
                yaml_file_name: "semaphore.yml",
                working_directory: ".semaphore",
                project_id: "proj-1",
                created_at: "2026-01-22T15:32:47.000000Z",
                done_at: "2026-01-22T15:32:56.000000Z",
                running_at: "2026-01-22T15:32:48.000000Z",
                error_description: "boom",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Pipeline ID": "ppl-1",
      "Pipeline Name": "Initial Pipeline",
      "Workflow ID": "wf-1",
      State: "done",
      Result: "passed",
      "Result Reason": "test",
      Branch: "main",
      "Commit SHA": "abc123",
      "Commit Message": "feat: add new feature",
      "YAML File": "semaphore.yml",
      "Working Directory": ".semaphore",
      "Project ID": "proj-1",
      "Created At": "2026-01-22T15:32:47.000000Z",
      "Done At": "2026-01-22T15:32:56.000000Z",
      "Running At": "2026-01-22T15:32:48.000000Z",
      Error: "boom",
    });
  });

  it("returns no details when outputs are missing", () => {
    const details = getPipelineMapper.getExecutionDetails(buildDetailsContext({ outputs: {} }));

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get Pipeline",
    componentName: "semaphore.getPipeline",
    isCollapsed: false,
    configuration: {},
  };

  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
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
