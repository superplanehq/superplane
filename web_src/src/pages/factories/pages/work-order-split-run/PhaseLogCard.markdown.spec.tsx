import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";

const useLiveLogStreamMock = vi.fn();

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

describe("PhaseLogCard agent note markdown", () => {
  it("renders a fenced agent note as one code block", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        stream={[
          line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue", componentType: "Run Claude Code" }),
          line({ id: "step-write", note: true, componentName: "Write Implementation Plan", componentType: "prompt" }),
          line({
            id: "note-code",
            note: true,
            noteParentId: "step-write",
            componentType: "note",
            componentName: "Here is the change:\n\n```ts\nconst n = 1;\n\nconst m = 2;\n```",
          }),
        ]}
      />,
    );

    const blocks = screen.getAllByTestId("monaco-stub");
    expect(blocks).toHaveLength(1);
    expect(blocks[0]?.textContent).toBe("const n = 1;\n\nconst m = 2;");
    expect(screen.queryByText("undefined")).not.toBeInTheDocument();
    expect(screen.getByText("Here is the change:")).toBeInTheDocument();
  });

  it("renders a streaming unclosed fence as a code block", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        stream={[
          line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue", componentType: "Run Claude Code" }),
          line({ id: "step-write", note: true, componentName: "Write Implementation Plan", componentType: "prompt" }),
          line({
            id: "note-code",
            note: true,
            noteParentId: "step-write",
            componentType: "note",
            componentName: "Streaming:\n\n```ts\nconst n = 1;",
          }),
        ]}
      />,
    );

    expect(screen.getAllByTestId("monaco-stub")).toHaveLength(1);
    expect(screen.getByTestId("monaco-stub").textContent).toBe("const n = 1;");
    expect(screen.queryByText("```")).not.toBeInTheDocument();
    expect(screen.queryByText("```ts")).not.toBeInTheDocument();
  });
});
