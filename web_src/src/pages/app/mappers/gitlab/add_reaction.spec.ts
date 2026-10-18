import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
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
  return buildGitLabDetailsContext(
    {
      name: "Add Reaction",
      componentName: "gitlab.addReaction",
    },
    execution,
  );
}
