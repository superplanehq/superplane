import { describe, expect, it } from "bun:test";

import { enabledToolCount, mcpToolItems, nextDisabledTools, sortMCPTools, workspaceDisabledTools } from "./mcpTools";

const tools = mcpToolItems([
  { name: "write_issue", readOnly: false },
  { name: "search", readOnly: true },
  { name: "create_issue", readOnly: false },
]);

describe("mcpTools", () => {
  it("sorts by name, then read first, then write first", () => {
    expect(sortMCPTools(tools, "name").map((tool) => tool.name)).toEqual(["create_issue", "search", "write_issue"]);
    expect(sortMCPTools(tools, "read").map((tool) => tool.name)).toEqual(["search", "create_issue", "write_issue"]);
    expect(sortMCPTools(tools, "write").map((tool) => tool.name)).toEqual(["create_issue", "write_issue", "search"]);
  });

  it("counts enabled tools after the denylist", () => {
    expect(enabledToolCount(tools, ["create_issue"])).toBe(2);
    expect(workspaceDisabledTools({ disabledTools: [" create_issue ", ""] })).toEqual(["create_issue"]);
    expect(nextDisabledTools(["search"], "create_issue", false)).toEqual(["search", "create_issue"]);
    expect(nextDisabledTools(["search", "create_issue"], "search", true)).toEqual(["create_issue"]);
  });
});
