import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ChangeEvent } from "react";

import { PanelEditorDialog } from "./PanelEditorDialog";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea
      aria-label="Panel YAML"
      value={value ?? ""}
      onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onChange?.(event.currentTarget.value)}
    />
  ),
}));

vi.mock("../CanvasYamlDiffModal", () => ({
  CanvasYamlDiffModal: ({
    open,
    liveYamlText,
    draftYamlText,
    liveLabel,
    draftLabel,
  }: {
    open: boolean;
    liveYamlText: string;
    draftYamlText: string;
    liveLabel?: string;
    draftLabel?: string;
  }) =>
    open ? (
      <div data-testid="panel-yaml-diff-modal">
        <span>{liveLabel}</span>
        <span>{draftLabel}</span>
        <pre data-testid="saved-yaml">{liveYamlText}</pre>
        <pre data-testid="draft-yaml">{draftYamlText}</pre>
      </div>
    ) : null,
}));

type MarkdownContent = {
  title: string;
  body: string;
};

function renderPanelEditor(initialContent: MarkdownContent = { title: "Before", body: "Original" }) {
  return render(
    <PanelEditorDialog<MarkdownContent>
      open
      onOpenChange={vi.fn()}
      panelId="runbook"
      panelType="markdown"
      initialContent={initialContent}
      onSave={vi.fn()}
      renderForm={({ value, onChange }) => (
        <input
          aria-label="Panel title"
          value={value.title}
          onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...value, title: event.currentTarget.value })}
        />
      )}
    />,
  );
}

describe("PanelEditorDialog", () => {
  it("shows a YAML diff action after panel edits change the draft", async () => {
    const user = userEvent.setup();
    renderPanelEditor();

    expect(screen.queryByRole("button", { name: /view diff/i })).not.toBeInTheDocument();

    await user.clear(screen.getByLabelText("Panel title"));
    await user.type(screen.getByLabelText("Panel title"), "After");

    await user.click(screen.getByRole("button", { name: /view diff/i }));

    expect(await screen.findByTestId("panel-yaml-diff-modal")).toBeInTheDocument();
    expect(screen.getByText("Saved")).toBeInTheDocument();
    expect(screen.getByText("Draft edits")).toBeInTheDocument();
    expect(screen.getByTestId("saved-yaml")).toHaveTextContent("Before");
    expect(screen.getByTestId("draft-yaml")).toHaveTextContent("After");
  });
});
