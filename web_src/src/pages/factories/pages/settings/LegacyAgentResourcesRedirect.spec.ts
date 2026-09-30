import { describe, expect, it } from "bun:test";

import { legacyAgentResourcesPath } from "./legacyAgentResourcesPath";

describe("legacyAgentResourcesPath", () => {
  it("sends the default route to MCP", () => {
    expect(legacyAgentResourcesPath(new URLSearchParams())).toBe("../workspace/agent");
  });

  it("keeps the add catalog on MCP", () => {
    expect(legacyAgentResourcesPath(new URLSearchParams("dialog=add"))).toBe("../workspace/agent?dialog=add");
  });

  it("sends the skills tab to Skills", () => {
    expect(legacyAgentResourcesPath(new URLSearchParams("tab=skills"))).toBe("../workspace/agent");
  });

  it("sends add on the skills tab to the skill editor", () => {
    expect(legacyAgentResourcesPath(new URLSearchParams("tab=skills&dialog=add"))).toBe("../workspace/skills/new");
  });
});
