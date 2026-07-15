import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { MarkdownVariablesPanel } from "./MarkdownVariablesPanel";
import type { MarkdownVariable } from "./panelTypes";

vi.mock("./MarkdownVariableSourceControls", () => ({
  MemorySourceControls: () => <div data-testid="memory-controls" />,
  RunSourceControls: () => <div data-testid="run-controls" />,
}));

const VARIABLES: MarkdownVariable[] = [
  { name: "deploy", source: { kind: "run", select: "latest", statuses: ["passed"], triggers: ["deploy"] } },
  { name: "memory", source: { kind: "memory", namespace: "ns" } },
];

describe("MarkdownVariablesPanel per-variable loading", () => {
  it("shows Loading only for variables still searching, and surfaces sibling errors", () => {
    render(
      <MarkdownVariablesPanel
        canvasId="canvas-1"
        draftBody=""
        draftVariables={VARIABLES}
        setDraftVariables={() => {}}
        previewVars={{ deploy: null, memory: null }}
        errors={[{ name: "memory", message: 'No memory rows in namespace "ns".' }]}
        baseLoading={false}
        sideloadLoading={false}
        searchingNames={["deploy"]}
        onInsertSnippet={() => {}}
      />,
    );

    expect(screen.getByText("Loading preview…")).toBeInTheDocument();
    expect(screen.getByTestId("markdown-variable-preview-error")).toHaveTextContent(/No memory rows/);
  });

  it("does not hide settled errors when no variable is searching", () => {
    render(
      <MarkdownVariablesPanel
        canvasId="canvas-1"
        draftBody=""
        draftVariables={VARIABLES}
        setDraftVariables={() => {}}
        previewVars={{ deploy: null, memory: null }}
        errors={[
          { name: "deploy", message: "No run matched the configured filters yet." },
          { name: "memory", message: 'No memory rows in namespace "ns".' },
        ]}
        baseLoading={false}
        sideloadLoading={false}
        searchingNames={[]}
        onInsertSnippet={() => {}}
      />,
    );

    expect(screen.queryByText("Loading preview…")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("markdown-variable-preview-error")).toHaveLength(2);
  });

  it("keeps previews loading while run executions are side-loading", () => {
    render(
      <MarkdownVariablesPanel
        canvasId="canvas-1"
        draftBody=""
        draftVariables={VARIABLES}
        setDraftVariables={() => {}}
        previewVars={{ deploy: { $: {} }, memory: null }}
        errors={[]}
        baseLoading={false}
        sideloadLoading
        searchingNames={[]}
        onInsertSnippet={() => {}}
      />,
    );

    expect(screen.getByText("Loading preview…")).toBeInTheDocument();
    expect(screen.getByText("No data resolved yet.")).toBeInTheDocument();
  });
});
