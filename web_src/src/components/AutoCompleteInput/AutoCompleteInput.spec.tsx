import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ExpressionAdapter } from "@/lib/expression";
import { AutoCompleteInput } from "./AutoCompleteInput";
import { calculateDropdownPosition } from "./dropdownPosition";

describe("calculateDropdownPosition", () => {
  it("anchors the dropdown top to the cursor y coordinate", () => {
    const position = calculateDropdownPosition({
      cursor: { x: 120, y: 240 },
      viewportWidth: 1000,
      dropdownWidth: 350,
      valuePreviewWidth: 200,
      showValuePreview: false,
    });

    expect(position.top).toBe(244);
  });

  it("keeps the dropdown inside the viewport horizontally", () => {
    const position = calculateDropdownPosition({
      cursor: { x: 980, y: 80 },
      viewportWidth: 1000,
      dropdownWidth: 350,
      valuePreviewWidth: 200,
      showValuePreview: false,
    });

    expect(position.left).toBe(630);
  });
});

describe("AutoCompleteInput preview toggle", () => {
  it("shows preview for blank inputs when value preview is enabled", () => {
    render(
      <AutoCompleteInput
        exampleObj={{ __root: { data: { name: "DCO" } } }}
        value=""
        onChange={vi.fn()}
        placeholder="{{ root().data.foo }}"
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        showValuePreview
        quickTip="Tip: type `{{` to start an expression."
      />,
    );

    const previewButton = screen.getByRole("button", { name: "Preview" });
    expect(previewButton).toBeInTheDocument();

    fireEvent.click(previewButton);

    expect(screen.getByRole("button", { name: "Preview" })).toBeInTheDocument();
    expect(screen.queryByText(/error \(/i)).not.toBeInTheDocument();
  });

  it("uses a custom preview label when provided", () => {
    render(
      <AutoCompleteInput
        exampleObj={{ __root: { data: { name: "DCO" } } }}
        value=""
        onChange={vi.fn()}
        placeholder="{{ root().data.foo }}"
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        showValuePreview
        valuePreviewLabel="Preview title"
      />,
    );

    expect(screen.getByRole("button", { name: "Preview title" })).toBeInTheDocument();
  });

  it("rejects mixed templates in path-or-raw mode", () => {
    const adapter: ExpressionAdapter = {
      id: "cel",
      evaluate: () => ({ ok: true, value: "ok", formattedValue: "ok" }),
      evaluatePathLiteral: () => ({ ok: true, value: "path", formattedValue: "path" }),
      resolveSuggestionValue: vi.fn(),
      formatResult: vi.fn(),
    };
    render(
      <AutoCompleteInput
        exampleObj={{ status: "passed" }}
        value="prefix {{ status }}"
        onChange={vi.fn()}
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        pathModeOutsideWrapper
        expressionAdapter={adapter}
        showValuePreview
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Preview" }));

    expect(screen.getByText(/Expected a single full \{\{ … \}\} expression/)).toBeInTheDocument();
  });

  it("rejects empty path-or-raw expressions", () => {
    const adapter: ExpressionAdapter = {
      id: "cel",
      evaluate: () => ({ ok: true, value: "", formattedValue: "" }),
      evaluatePathLiteral: () => ({ ok: true, value: "path", formattedValue: "path" }),
      resolveSuggestionValue: vi.fn(),
      formatResult: vi.fn(),
    };
    render(
      <AutoCompleteInput
        exampleObj={{ status: "passed" }}
        value="{{   }}"
        onChange={vi.fn()}
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        pathModeOutsideWrapper
        expressionAdapter={adapter}
        showValuePreview
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Preview" }));

    expect(screen.getByText(/Expression cannot be empty/)).toBeInTheDocument();
  });

  it("evaluates repeated path-or-raw templates as one runtime expression", () => {
    const evaluate = vi.fn((expression: string) =>
      expression.includes("{{")
        ? { ok: false, error: "Unexpected template delimiter" }
        : { ok: true, value: "ok", formattedValue: "ok" },
    );
    const adapter: ExpressionAdapter = {
      id: "cel",
      evaluate,
      evaluatePathLiteral: () => ({ ok: true, value: "path", formattedValue: "path" }),
      resolveSuggestionValue: vi.fn(),
      formatResult: vi.fn(),
    };
    render(
      <AutoCompleteInput
        exampleObj={{ first: "hello", second: "world" }}
        value="{{ first }} {{ second }}"
        onChange={vi.fn()}
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        pathModeOutsideWrapper
        expressionAdapter={adapter}
        showValuePreview
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Preview" }));

    expect(evaluate).toHaveBeenCalledWith(" first }} {{ second ", expect.any(Object));
    expect(screen.getByText(/Unexpected template delimiter/)).toBeInTheDocument();
  });
});

describe("AutoCompleteInput suggestions", () => {
  const renderRunTitleInput = () =>
    render(
      <AutoCompleteInput
        aria-label="Run title"
        exampleObj={{ __root: { data: { name: "DCO", sha: "d6f3c8a2e8b7" } } }}
        value=""
        onChange={vi.fn()}
        placeholder="{{ root().data.foo }}"
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        showValuePreview
      />,
    );

  it("suggests root data fields inside wrapped expressions", async () => {
    renderRunTitleInput();
    const input = screen.getByRole("textbox", { name: "Run title" });
    const value = "{{ root().data.";

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value, selectionStart: value.length } });

    expect(await screen.findByText("name")).toBeInTheDocument();
    expect(screen.getByText("sha")).toBeInTheDocument();
  });

  it("shows canonical root() syntax in function suggestions", async () => {
    renderRunTitleInput();
    const input = screen.getByRole("textbox", { name: "Run title" });
    const value = "{{ ro";

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value, selectionStart: value.length } });

    expect(await screen.findAllByText("root()")).not.toHaveLength(0);
  });

  it("keeps keyboard highlight when rerendering the same suggestions", async () => {
    const value = "{{ root().data.";
    const renderInput = (exampleObj: Record<string, unknown>) => (
      <AutoCompleteInput
        aria-label="Run title"
        exampleObj={exampleObj}
        value={value}
        onChange={vi.fn()}
        placeholder="{{ root().data.foo }}"
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        showValuePreview
      />
    );

    const { rerender } = render(renderInput({ __root: { data: { name: "DCO", sha: "d6f3c8a2e8b7" } } }));
    const input = screen.getByRole("textbox", { name: "Run title" });

    fireEvent.focus(input);
    (input as HTMLTextAreaElement).setSelectionRange(value.length, value.length);
    fireEvent.select(input);
    expect(await screen.findByText("name")).toBeInTheDocument();

    fireEvent.keyDown(input, { key: "ArrowDown" });
    rerender(renderInput({ __root: { data: { name: "DCO", sha: "d6f3c8a2e8b7" } } }));

    await waitFor(() => {
      expect(document.querySelector('[data-suggestion-index="1"]')).toHaveClass("bg-slate-100");
    });
  });

  it("keeps suggestions open after accepting an expandable suggestion inside an expression", async () => {
    renderRunTitleInput();
    const input = screen.getByRole("textbox", { name: "Run title" });
    const value = "{{ ro";

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value, selectionStart: value.length } });

    expect(await screen.findAllByText("root()")).not.toHaveLength(0);

    fireEvent.keyDown(input, { key: "Enter" });

    expect(screen.getByText("data")).toBeInTheDocument();
    await waitFor(() => {
      expect(input).toHaveValue("{{ root().");
    });
  });

  it("keeps follow-up suggestions visible after the keyboard cursor sync frame", async () => {
    renderRunTitleInput();
    const input = screen.getByRole("textbox", { name: "Run title" });
    const value = "{{ ro";

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value, selectionStart: value.length } });

    expect(await screen.findAllByText("root()")).not.toHaveLength(0);

    fireEvent.keyDown(input, { key: "Enter" });

    await new Promise((resolve) => requestAnimationFrame(resolve));

    expect(screen.getByText("data")).toBeInTheDocument();
    expect(screen.getByTestId("autocomplete-value-preview")).toBeInTheDocument();
  });
});

describe("AutoCompleteInput fullHeight mode", () => {
  it("lets the textarea fill its parent instead of auto-resizing", () => {
    render(
      <div style={{ height: 400 }}>
        <AutoCompleteInput
          aria-label="Full height editor"
          exampleObj={{ __root: { data: { name: "DCO" } } }}
          value={"line 1\nline 2\nline 3"}
          onChange={vi.fn()}
          placeholder="Type here"
          startWord="{{"
          prefix="{{ "
          suffix=" }}"
          fullHeight
        />
      </div>,
    );

    const input = screen.getByRole("textbox", { name: "Full height editor" }) as HTMLTextAreaElement;
    // Auto-resize must not set an explicit height style — the parent controls layout.
    expect(input.style.height).toBe("");
    expect(input.className).toContain("h-full");
    expect(input.className).toContain("overflow-y-auto");
  });

  it("keeps auto-resize enabled when fullHeight is not requested", () => {
    render(
      <AutoCompleteInput
        aria-label="Auto sized editor"
        exampleObj={null}
        value={"one\ntwo"}
        onChange={vi.fn()}
        placeholder="Type here"
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        minHeight={80}
      />,
    );

    const input = screen.getByRole("textbox", { name: "Auto sized editor" }) as HTMLTextAreaElement;
    // Auto-resize should still set an inline height.
    expect(input.style.height).not.toBe("");
    expect(input.className).not.toContain("h-full");
  });
});
