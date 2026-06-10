import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { FileMonacoEditor } from "./FileMonacoEditor";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea data-testid="monaco-stub" value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
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
  it("records the first edit after switching back to a file", async () => {
    const user = userEvent.setup();

    render(<ControlledEditorHarness initialPath="README.md" initialContent="# readme" />);

    await user.click(screen.getByRole("button", { name: "Open other" }));
    await user.click(screen.getByRole("button", { name: "Open first" }));

    await user.type(screen.getByTestId("monaco-stub"), "!");

    expect(screen.getByTestId("last-edit")).toHaveTextContent("# readme!");
  });
});
