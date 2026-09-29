import { describe, expect, it } from "bun:test";

import type { AgentToolItem } from "@/lib/agentActivity";
import { agentToolDisplayText, agentToolOutputPreview } from "@/lib/agentToolLabels";

function tool(overrides: Partial<AgentToolItem> = {}): AgentToolItem {
  return {
    type: "tool",
    id: "tool-1",
    kind: "bash",
    name: "Bash",
    input: "",
    output: "",
    outputStreams: [],
    status: "passed",
    truncated: false,
    ...overrides,
  };
}

describe("agentToolDisplayText", () => {
  it("uses the command text for bash tools", () => {
    expect(agentToolDisplayText(tool({ input: '{"command":"git status"}' }))).toBe("git status");
  });

  it("uses the tool name once when the command has not arrived", () => {
    expect(agentToolDisplayText(tool())).toBe("Bash");
  });

  it("does not pair the kind with the name", () => {
    const label = agentToolDisplayText(tool({ kind: "bash", name: "Bash", input: "" }));
    expect(label.toLowerCase()).not.toContain("bash bash");
    expect(label).toBe("Bash");
  });

  it("names read, edit, and write rows from the file", () => {
    expect(agentToolDisplayText(tool({ kind: "read", name: "Read", input: '{"path":"pkg/foo.go"}' }))).toBe(
      "pkg/foo.go",
    );
    expect(agentToolDisplayText(tool({ kind: "edit", name: "Edit", input: "web_src/src/lib/agentActivity.ts" }))).toBe(
      "web_src/src/lib/agentActivity.ts",
    );
    expect(agentToolDisplayText(tool({ kind: "write", name: "Write", input: '{"file_path":"tmp/out.txt"}' }))).toBe(
      "tmp/out.txt",
    );
  });

  it("uses the search pattern", () => {
    expect(agentToolDisplayText(tool({ kind: "grep", name: "Grep", input: "rootTriggerRenderer" }))).toBe(
      "rootTriggerRenderer",
    );
  });

  it("reads search and fetch fields from JSON input", () => {
    expect(agentToolDisplayText(tool({ kind: "grep", name: "Grep", input: '{"pattern":"rootTriggerRenderer"}' }))).toBe(
      "rootTriggerRenderer",
    );
    expect(
      agentToolDisplayText(tool({ kind: "glob", name: "Glob", input: '{"glob_pattern":"**/*.go","path":"pkg"}' })),
    ).toBe("**/*.go");
    expect(
      agentToolDisplayText(tool({ kind: "web_search", name: "WebSearch", input: '{"query":"analysis logs"}' })),
    ).toBe("analysis logs");
    expect(
      agentToolDisplayText(tool({ kind: "web_fetch", name: "WebFetch", input: '{"url":"https://example.com/docs"}' })),
    ).toBe("https://example.com/docs");
  });

  it("uses the tool name for incomplete JSON", () => {
    expect(agentToolDisplayText(tool({ kind: "grep", name: "Grep", input: '{"pattern":"rootTrigger' }))).toBe("Grep");
  });

  it("does not show raw JSON when known fields are missing", () => {
    expect(agentToolDisplayText(tool({ kind: "grep", name: "Grep", input: '{"path":"pkg"}' }))).toBe("Grep");
  });

  it("does not expose partial command JSON", () => {
    expect(agentToolDisplayText(tool({ input: '{"command":"ls pkg' }))).toBe("Bash");
  });
});

describe("agentToolOutputPreview", () => {
  it("returns the first three visible lines and a more flag", () => {
    expect(
      agentToolOutputPreview(
        tool({
          output: ["one", "", "two", '{"schema_version":2,"type":"line","activity_id":"a"}', "three", "four"].join(
            "\n",
          ),
        }),
      ),
    ).toEqual({ lines: ["one", "two", "three"], hasMore: true });
  });

  it("flags truncated output even when three or fewer lines exist", () => {
    expect(agentToolOutputPreview(tool({ output: "only line\n", truncated: true }))).toEqual({
      lines: ["only line"],
      hasMore: true,
    });
  });

  it("returns no lines when the tool has no output", () => {
    expect(agentToolOutputPreview(tool())).toEqual({ lines: [], hasMore: false });
  });
});
