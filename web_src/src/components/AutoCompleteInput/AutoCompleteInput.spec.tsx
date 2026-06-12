import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AutoCompleteInput } from "./AutoCompleteInput";

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
});
