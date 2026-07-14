import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { addReactionMapper } from "./add_reaction";

describe("addReactionMapper", () => {
  it("shows reaction details for a merge request target", () => {
    const details = addReactionMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          mergeRequestIid: "1",
          target: "mergeRequest",
          content: "eyes",
        },
        outputs: {
          default: [
            {
              data: {
                id: 25,
                name: "eyes",
                user: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      Reaction: "👀",
      "Created By": "root",
      "Created At": expect.any(String),
    });
  });

  it("shows reaction details for a note target", () => {
    const details = addReactionMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          mergeRequestIid: "1",
          target: "note",
          noteId: "302",
          content: "rocket",
        },
        outputs: {
          default: [
            {
              data: {
                id: 26,
                name: "rocket",
                user: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(details["Reaction"]).toEqual("🚀");
  });

  it("handles missing outputs", () => {
    const details = addReactionMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", mergeRequestIid: "1", target: "mergeRequest", content: "eyes" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Add Reaction",
    componentName: "gitlab.addReaction",
    isCollapsed: false,
    metadata: {
      project: {
        id: 123,
        name: "felixgateru/hello-world",
        url: "https://gitlab.com/felixgateru/hello-world",
      },
    },
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
