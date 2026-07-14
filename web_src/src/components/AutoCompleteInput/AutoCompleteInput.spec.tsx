import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
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
});
