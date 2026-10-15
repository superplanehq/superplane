import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
import { getIssueMapper } from "./get_issue";

describe("getIssueMapper", () => {
  it("shows fetched issue details", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          issueIid: "1",
        },
        outputs: {
          default: [
            {
              data: {
                id: 101,
                iid: 1,
                title: "Bug report",
                state: "opened",
                web_url: "https://gitlab.com/felixgateru/hello-world/-/issues/1",
                author: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
                labels: ["bug"],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Executed At": expect.any(String),
      Title: "Bug report",
      State: "opened",
      "Created By": "root",
      Labels: "bug",
      URL: "https://gitlab.com/felixgateru/hello-world/-/issues/1",
    });
  });

  it("handles missing outputs", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "1" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  return buildGitLabDetailsContext(
    {
      name: "Get Issue",
      componentName: "gitlab.getIssue",
    },
    execution,
  );
}
