import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { MarkdownVariablesPanel } from "./MarkdownVariablesPanel";
import type { MarkdownVariable } from "./panelTypes";

vi.mock("./MarkdownVariableSourceControls", () => ({
  MemorySourceControls: () => <div data-testid="memory-controls" />,
  RunSourceControls: ({ source }: { source: { statuses?: readonly string[]; triggers?: readonly string[] } }) => {
    const [open, setOpen] = useState(Boolean(source.statuses?.length || source.triggers?.length));
    return (
      <div data-testid="run-controls">
        <button type="button" onClick={() => setOpen((value) => !value)}>
          Toggle mocked run filters
        </button>
        {open ? <div data-testid="mock-run-filter-content" /> : null}
      </div>
    );
  },
}));

const VARIABLES: MarkdownVariable[] = [
  { name: "deploy", source: { kind: "run", select: "latest", statuses: ["passed"], triggers: ["deploy"] } },
  { name: "memory", source: { kind: "memory", namespace: "ns" } },
];

describe("MarkdownVariablesPanel per-variable loading", () => {
  it("does not reuse collapsed filter state after deleting a preceding variable", () => {
    const unfiltered: MarkdownVariable = { name: "first", source: { kind: "run", select: "latest" } };
    const filtered = VARIABLES[0];
    const props = {
      canvasId: "canvas-1",
      draftBody: "",
      setDraftVariables: () => {},
      previewVars: {},
      errors: [],
      baseLoading: false,
      sideloadLoading: false,
      onInsertSnippet: () => {},
    };
    const { rerender } = render(<MarkdownVariablesPanel {...props} draftVariables={[unfiltered, filtered]} />);

    expect(screen.getAllByTestId("mock-run-filter-content")).toHaveLength(1);

    rerender(<MarkdownVariablesPanel {...props} draftVariables={[filtered]} />);

    expect(screen.getByTestId("mock-run-filter-content")).toBeInTheDocument();
  });

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
