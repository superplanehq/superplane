import { describe, expect, it } from "bun:test";

import { toolCallSummary } from "./phaseLogStream";

describe("toolCallSummary", () => {
  it("distinguishes files, commands, and other tools", () => {
    expect(toolCallSummary([{ type: "read" }, { type: "read" }, { type: "bash" }])).toBe("Read 2 files, ran 1 command");
    expect(toolCallSummary([{ componentType: "read" }, { componentType: "bash" }])).toBe("Read 1 file, ran 1 command");
    expect(toolCallSummary([{ type: "read" }])).toBe("Read 1 file");
    expect(toolCallSummary([{ type: "bash" }, { type: "bash" }])).toBe("Ran 2 commands");
    expect(
      toolCallSummary([
        { type: "read" },
        { type: "task" },
        { type: "bash" },
        { type: "superplane_survey" },
        { type: "superplane_propose_clarity" },
        { type: "superplane_propose_confidence" },
      ]),
    ).toBe("Read 1 file, ran 1 command, used 4 tools");
    expect(toolCallSummary([{ type: "task" }])).toBe("Used 1 tool");
  });
});
