import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ChangeEvent } from "react";

import type { ConfigurationField } from "@/api-client";

import { TextFieldRenderer } from "./TextFieldRenderer";

vi.mock("@monaco-editor/react", () => ({
  default: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea
      aria-label="Monaco editor"
      value={value ?? ""}
      onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onChange?.(event.currentTarget.value)}
    />
  ),
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

function textField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "prompt",
    type: "text",
    label: "Prompt",
    ...overrides,
  } as ConfigurationField;
}

function ControlledText({
  field,
  initialValue,
  onChange,
  allowExpressions,
}: {
  field: ConfigurationField;
  initialValue?: string;
  onChange?: (value: unknown) => void;
  allowExpressions?: boolean;
}) {
  const [value, setValue] = useState<unknown>(initialValue ?? "");
  return (
    <TextFieldRenderer
      field={field}
      value={value}
      onChange={(next) => {
        setValue(next);
        onChange?.(next);
      }}
      allowExpressions={allowExpressions}
    />
  );
}

describe("TextFieldRenderer plain textarea expansion", () => {
  it("shows an accessible expand button next to plain multiline inputs", () => {
    render(<ControlledText field={textField()} />);

    const expand = screen.getByRole("button", { name: /expand prompt editor/i });
    expect(expand).toBeInTheDocument();
    expect(expand).toHaveAttribute("data-testid", "text-field-prompt-expand");
  });

  it("opens a full-page dialog seeded with the current value", () => {
    render(<ControlledText field={textField()} initialValue="Deploy notes" />);

    fireEvent.click(screen.getByRole("button", { name: /expand prompt editor/i }));

    const modalInput = screen.getByTestId("text-field-prompt-modal-input") as HTMLTextAreaElement;
    expect(modalInput.value).toBe("Deploy notes");
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
  });

  it("keeps the draft isolated from the inline field until Save is pressed", () => {
    const onChange = vi.fn();
    render(<ControlledText field={textField()} initialValue="Original" onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: /expand prompt editor/i }));
    const modalInput = screen.getByTestId("text-field-prompt-modal-input") as HTMLTextAreaElement;
    fireEvent.change(modalInput, { target: { value: "In-progress edit" } });

    // Inline value has not been updated yet.
    expect(onChange).not.toHaveBeenCalled();
    const inlineInput = screen.getByTestId("text-field-prompt") as HTMLTextAreaElement;
    expect(inlineInput.value).toBe("Original");

    fireEvent.click(screen.getByRole("button", { name: /save/i }));

    expect(onChange).toHaveBeenCalledWith("In-progress edit");
    expect(screen.queryByTestId("text-field-prompt-modal-input")).not.toBeInTheDocument();
    expect(inlineInput.value).toBe("In-progress edit");
  });

  it("discards edits when Cancel is pressed and starts fresh on reopen", () => {
    const onChange = vi.fn();
    render(<ControlledText field={textField()} initialValue="Original" onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: /expand prompt editor/i }));
    const modalInput = screen.getByTestId("text-field-prompt-modal-input") as HTMLTextAreaElement;
    fireEvent.change(modalInput, { target: { value: "Throwaway" } });
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));

    expect(onChange).not.toHaveBeenCalled();
    const inlineInput = screen.getByTestId("text-field-prompt") as HTMLTextAreaElement;
    expect(inlineInput.value).toBe("Original");

    fireEvent.click(screen.getByRole("button", { name: /expand prompt editor/i }));
    const reopened = screen.getByTestId("text-field-prompt-modal-input") as HTMLTextAreaElement;
    expect(reopened.value).toBe("Original");
  });
});

describe("TextFieldRenderer expression-aware expansion", () => {
  it("uses the expression-aware autocomplete input inside the dialog", () => {
    render(
      <ControlledText
        field={textField({ name: "message", label: "Message" })}
        initialValue="{{ root().data.title }}"
        allowExpressions
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /expand message editor/i }));

    const modalInput = screen.getByTestId("text-field-message-modal-input") as HTMLTextAreaElement;
    expect(modalInput.value).toBe("{{ root().data.title }}");
    // The expression-aware editor exposes the preview toggle inside the modal.
    expect(screen.getAllByRole("button", { name: "Preview" }).length).toBeGreaterThan(0);
  });
});

describe("TextFieldRenderer code editor expansion", () => {
  const codeField = textField({
    name: "script",
    label: "Script",
    typeOptions: { text: { language: "shell" } },
  });

  it("stages Monaco edits and only propagates them after Save", () => {
    const onChange = vi.fn();
    render(<ControlledText field={codeField} initialValue={"echo hi"} onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: /expand script editor/i }));

    const editors = screen.getAllByLabelText("Monaco editor") as HTMLTextAreaElement[];
    expect(editors.length).toBe(2);
    const modalEditor = editors[1];
    expect(modalEditor.value).toBe("echo hi");

    fireEvent.change(modalEditor, { target: { value: "echo staged" } });
    expect(onChange).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /save/i }));
    expect(onChange).toHaveBeenLastCalledWith("echo staged");
  });

  it("discards Monaco edits when Cancel is pressed", () => {
    const onChange = vi.fn();
    render(<ControlledText field={codeField} initialValue={"echo hi"} onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: /expand script editor/i }));
    const modalEditor = (screen.getAllByLabelText("Monaco editor") as HTMLTextAreaElement[])[1];
    fireEvent.change(modalEditor, { target: { value: "echo staged" } });
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));

    expect(onChange).not.toHaveBeenCalled();
    const inlineEditor = screen.getByLabelText("Monaco editor") as HTMLTextAreaElement;
    expect(inlineEditor.value).toBe("echo hi");
  });
});
