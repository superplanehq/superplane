import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { createReleaseMapper } from "./create_release";

const requiredFallbackEntries = [
  ["Created At", "-"],
  ["Created By", "-"],
  ["Release URL", ""],
  ["Release ID", ""],
  ["Tag Name", ""],
];

describe("createReleaseMapper.getExecutionDetails", () => {
  it.each([undefined, {}, { default: [] }])("returns no details without default outputs: %j", (outputs) => {
    expect(createReleaseMapper.getExecutionDetails(buildDetailsContext(outputs))).toEqual({});
  });

  it.each([{}, undefined, null, { author: {} }, { name: "", published_at: "" }])(
    "preserves ordered required fallbacks for sparse release data: %j",
    (data) => {
      const details = createReleaseMapper.getExecutionDetails(buildDetailsContext({ default: [{ data }] }));

      expect(Object.entries(details)).toEqual(requiredFallbackEntries);
    },
  );

  it("shows populated release details in order with local timestamps", () => {
    const createdAt = new Date().toISOString();
    const publishedAt = new Date(Date.now() + 60_000).toISOString();
    const details = createReleaseMapper.getExecutionDetails(
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
              created_at: createdAt,
              published_at: publishedAt,
              author: { login: "octocat" },
            },
          },
        ],
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Created At", new Date(createdAt).toLocaleString()],
      ["Created By", "octocat"],
      ["Release URL", "https://github.com/testhq/hello/releases/tag/v1.0.0"],
      ["Release ID", "42"],
      ["Tag Name", "v1.0.0"],
      ["Release Name", "First release"],
      ["Draft", "Yes"],
      ["Prerelease", "No"],
      ["Published At", new Date(publishedAt).toLocaleString()],
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
    {
      release: { draft: false, prerelease: true },
      optionalEntries: [
        ["Draft", "No"],
        ["Prerelease", "Yes"],
      ],
    },
    { release: { draft: true }, optionalEntries: [["Draft", "Yes"]] },
    { release: { prerelease: false }, optionalEntries: [["Prerelease", "No"]] },
    { release: {}, optionalEntries: [] },
  ])("formats flags independently and omits absent flags: $release", ({ release, optionalEntries }) => {
    const details = createReleaseMapper.getExecutionDetails(buildDetailsContext({ default: [{ data: release }] }));

    expect(Object.entries(details)).toEqual([...requiredFallbackEntries, ...optionalEntries]);
  });

  it("uses only the first default output and preserves numeric ID zero", () => {
    const details = createReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        default: [{ data: { id: 0 } }, { data: { id: 42, name: "Later release", draft: true } }],
        other: [{ data: { tag_name: "Other release", prerelease: true } }],
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Created At", "-"],
      ["Created By", "-"],
      ["Release URL", ""],
      ["Release ID", "0"],
      ["Tag Name", ""],
    ]);
  });
});

function buildDetailsContext(outputs: ExecutionInfo["outputs"]): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create release",
    componentName: "github.createRelease",
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
