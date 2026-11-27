import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { updateReleaseMapper } from "./update_release";

const requiredFallbackEntries = [
  ["Published At", "-"],
  ["Updated By", "-"],
  ["Release URL", ""],
  ["Release ID", ""],
  ["Tag Name", ""],
];

describe("updateReleaseMapper.getExecutionDetails", () => {
  it.each([undefined, {}, { default: [] }])("returns no details without default outputs: %j", (outputs) => {
    expect(updateReleaseMapper.getExecutionDetails(buildDetailsContext(outputs))).toEqual({});
  });

  it.each([{}, undefined, null, { author: {} }, { name: "", published_at: "", author: { login: "" } }])(
    "preserves ordered required fallbacks for sparse release data: %j",
    (data) => {
      const details = updateReleaseMapper.getExecutionDetails(buildDetailsContext({ default: [{ data }] }));

      expect(Object.entries(details)).toEqual(requiredFallbackEntries);
    },
  );

  it("shows complete release details in order with a local publication timestamp", () => {
    const publishedAt = new Date().toISOString();
    const details = updateReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        default: [
          {
            data: {
              id: 42,
              tag_name: "v1.0.0",
              name: "First release",
              html_url: "https://github.com/testhq/hello/releases/tag/v1.0.0",
              draft: true,
              prerelease: false,
              published_at: publishedAt,
              author: { login: "octocat" },
            },
          },
        ],
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Published At", new Date(publishedAt).toLocaleString()],
      ["Updated By", "octocat"],
      ["Release URL", "https://github.com/testhq/hello/releases/tag/v1.0.0"],
      ["Release ID", "42"],
      ["Tag Name", "v1.0.0"],
      ["Name", "First release"],
      ["Draft", "Yes"],
      ["Prerelease", "No"],
    ]);
  });

  it.each([
    {
      release: { draft: true, prerelease: true },
      optionalEntries: [
        ["Draft", "Yes"],
        ["Prerelease", "Yes"],
      ],
    },
    {
      release: { draft: false, prerelease: false },
      optionalEntries: [
        ["Draft", "No"],
        ["Prerelease", "No"],
      ],
    },
    { release: { draft: false }, optionalEntries: [["Draft", "No"]] },
    { release: { prerelease: true }, optionalEntries: [["Prerelease", "Yes"]] },
    { release: { draft: undefined, prerelease: undefined }, optionalEntries: [] },
  ])("formats flags independently and omits absent flags: $release", ({ release, optionalEntries }) => {
    const details = updateReleaseMapper.getExecutionDetails(buildDetailsContext({ default: [{ data: release }] }));

    expect(Object.entries(details)).toEqual([...requiredFallbackEntries, ...optionalEntries]);
  });

  it("uses only the first default output and preserves numeric ID zero", () => {
    const details = updateReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        default: [{ data: { id: 0 } }, { data: { id: 42, name: "Later release", draft: true } }],
        other: [{ data: { tag_name: "Other release", prerelease: true } }],
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Published At", "-"],
      ["Updated By", "-"],
      ["Release URL", ""],
      ["Release ID", "0"],
      ["Tag Name", ""],
    ]);
  });
});

function buildDetailsContext(outputs: ExecutionInfo["outputs"]): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Update release",
    componentName: "github.updateRelease",
    isCollapsed: false,
  };
  const timestamp = new Date().toISOString();

  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: timestamp,
      updatedAt: timestamp,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      outputs,
    },
  };
}
