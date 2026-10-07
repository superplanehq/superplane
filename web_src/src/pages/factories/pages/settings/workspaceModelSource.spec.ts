import { describe, expect, it } from "bun:test";

import { workspaceModelSource } from "./workspaceModelSource";

describe("workspaceModelSource", () => {
  it("reads the source from the saved agent harness", () => {
    expect(workspaceModelSource("AGENT_HARNESS_SUPERPLANE", true)).toBe("hosted");
    expect(workspaceModelSource("AGENT_HARNESS_CLAUDE_CODE", false)).toBe("own-key");
    expect(workspaceModelSource("AGENT_HARNESS_CODEX", false)).toBe("own-key");
  });

  it("uses a connected key when no harness is saved", () => {
    expect(workspaceModelSource(undefined, true)).toBe("own-key");
    expect(workspaceModelSource("AGENT_HARNESS_UNSPECIFIED", false)).toBe("hosted");
  });
});
