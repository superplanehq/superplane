import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";

import { FileMonacoEditor } from "./FileMonacoEditor";

const { deltaDecorations } = vi.hoisted(() => ({
  deltaDecorations: vi.fn(() => ["decoration-1"]),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({
    value,
    onChange,
    onMount,
    options,
  }: {
    value?: string;
    onChange?: (value: string | undefined) => void;
    onMount?: (editor: unknown, monaco: unknown) => void;
    options?: { readOnly?: boolean };
  }) => {
    onMount?.(
      {
        getModel: () => ({ getLineCount: () => 2, getLineMaxColumn: () => 7 }),
        deltaDecorations,
        onDidDispose: vi.fn(),
      },
      {
        Range: class {
          constructor(
            readonly startLineNumber: number,
            readonly startColumn: number,
            readonly endLineNumber: number,
            readonly endColumn: number,
          ) {}
        },
      },
    );

    return (
      <textarea
        data-testid="monaco-stub"
        data-read-only={options?.readOnly ? "true" : "false"}
        value={value ?? ""}
        onChange={(event) => onChange?.(event.target.value)}
      />
    );
  },
}));

function ControlledEditorHarness({ initialPath, initialContent }: { initialPath: string; initialContent: string }) {
  const [path, setPath] = useState(initialPath);
  const [contentByPath, setContentByPath] = useState<Record<string, string>>({
    [initialPath]: initialContent,
  });
  const [lastEdit, setLastEdit] = useState<string | null>(null);

  return (
    <div>
      <button type="button" onClick={() => setPath("other.md")}>
        Open other
      </button>
      <button type="button" onClick={() => setPath(initialPath)}>
        Open first
      </button>
      <FileMonacoEditor
        path={path}
        content={contentByPath[path] ?? ""}
        readOnly={false}
        onChange={(value) => {
          setLastEdit(value);
          setContentByPath((current) => ({ ...current, [path]: value }));
        }}
      />
      <div data-testid="last-edit">{lastEdit ?? ""}</div>
    </div>
  );
}

describe("FileMonacoEditor", () => {
  it("decorates every line with the added or deleted file color", () => {
    deltaDecorations.mockClear();
    const { rerender } = render(
      <ThemeProvider>
        <FileMonacoEditor path="README.md" content="# readme\n" status="added" readOnly={false} onChange={vi.fn()} />
      </ThemeProvider>,
    );

    expect(deltaDecorations).toHaveBeenLastCalledWith(
      [],
      [
        expect.objectContaining({
          options: expect.objectContaining({ inlineClassName: "!text-green-600 dark:!text-green-400" }),
        }),
      ],
    );

    rerender(
      <ThemeProvider>
        <FileMonacoEditor path="README.md" content="# readme\n" status="deleted" readOnly onChange={vi.fn()} />
      </ThemeProvider>,
    );

    expect(deltaDecorations).toHaveBeenLastCalledWith(
      ["decoration-1"],
      [
        expect.objectContaining({
          options: expect.objectContaining({ inlineClassName: "!text-red-600 dark:!text-red-400" }),
        }),
      ],
    );
    expect(screen.getByTestId("monaco-stub")).toHaveAttribute("data-read-only", "true");
  });

  it("records the first edit after switching back to a file", async () => {
    const user = userEvent.setup();

    render(
      <ThemeProvider>
        <ControlledEditorHarness initialPath="README.md" initialContent="# readme" />
      </ThemeProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Open other" }));
    await user.click(screen.getByRole("button", { name: "Open first" }));

    await user.type(screen.getByTestId("monaco-stub"), "!");

    expect(screen.getByTestId("last-edit")).toHaveTextContent("# readme!");
  });
});
