import { describe, expect, it } from "bun:test";

import type { AgentToolItem } from "@/lib/agentActivity";
import { agentToolLabel, agentToolLabelText, agentToolOutputPreview, commandDisplayText } from "@/lib/agentToolLabels";

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

describe("agentToolLabel", () => {
  it("uses the command text for bash tools", () => {
    expect(agentToolLabel(tool({ input: '{"command":"git status"}' }))).toEqual({ action: "git status" });
    expect(agentToolLabelText(tool({ input: '{"command":"git status"}' }))).toBe("git status");
  });

  it("uses the tool name once when the command has not arrived", () => {
    expect(agentToolLabel(tool())).toEqual({ action: "Bash" });
    expect(agentToolLabelText(tool())).toBe("Bash");
  });

  it("does not pair the kind with the name", () => {
    const label = agentToolLabelText(tool({ kind: "bash", name: "Bash", input: "" }));
    expect(label.toLowerCase()).not.toContain("bash bash");
    expect(label).toBe("Bash");
  });

  it("names read, edit, and write rows from the file", () => {
    expect(agentToolLabelText(tool({ kind: "read", name: "Read", input: '{"path":"pkg/foo.go"}' }))).toBe(
      "Explored foo.go",
    );
    expect(agentToolLabelText(tool({ kind: "edit", name: "Edit", input: "web_src/src/lib/agentActivity.ts" }))).toBe(
      "Edited agentActivity.ts",
    );
    expect(agentToolLabelText(tool({ kind: "write", name: "Write", input: '{"file_path":"tmp/out.txt"}' }))).toBe(
      "Created out.txt",
    );
  });

  it("puts the search pattern in the detail", () => {
    expect(agentToolLabel(tool({ kind: "grep", name: "Grep", input: "rootTriggerRenderer" }))).toEqual({
      action: "Searched",
      detail: "rootTriggerRenderer",
    });
    expect(agentToolLabelText(tool({ kind: "grep", name: "Grep", input: "rootTriggerRenderer" }))).toBe(
      "rootTriggerRenderer",
    );
  });

  it("reads search and fetch fields from JSON input", () => {
    expect(agentToolLabel(tool({ kind: "grep", name: "Grep", input: '{"pattern":"rootTriggerRenderer"}' }))).toEqual({
      action: "Searched",
      detail: "rootTriggerRenderer",
    });
    expect(
      agentToolLabel(tool({ kind: "glob", name: "Glob", input: '{"glob_pattern":"**/*.go","path":"pkg"}' })),
    ).toEqual({
      action: "Searched",
      detail: "**/*.go",
    });
    expect(agentToolLabel(tool({ kind: "web_search", name: "WebSearch", input: '{"query":"analysis logs"}' }))).toEqual(
      {
        action: "Searched the web",
        detail: "analysis logs",
      },
    );
    expect(
      agentToolLabel(tool({ kind: "web_fetch", name: "WebFetch", input: '{"url":"https://example.com/docs"}' })),
    ).toEqual({
      action: "Fetched page",
      detail: "https://example.com/docs",
    });
  });

  it("reads a search pattern from partial JSON", () => {
    expect(
      agentToolLabel(tool({ kind: "grep", name: "Grep", input: '{"pattern":"rootTrigger', status: "running" })),
    ).toEqual({
      action: "Searching",
      detail: "rootTrigger",
    });
  });

  it("does not show raw JSON when known fields are missing", () => {
    expect(agentToolLabel(tool({ kind: "grep", name: "Grep", input: '{"path":"pkg"}' }))).toEqual({
      action: "Searched",
    });
  });
});

describe("commandDisplayText", () => {
  it("reads a command from partial JSON", () => {
    expect(commandDisplayText('{"command":"ls pkg')).toBe("ls pkg");
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
