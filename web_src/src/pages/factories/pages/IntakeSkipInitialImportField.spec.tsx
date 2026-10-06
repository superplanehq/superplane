import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";

describe("IntakeSkipInitialImportField", () => {
  it("shows a pointer cursor while the choice can change", () => {
    render(
      <IntakeSkipInitialImportField
        checked={false}
        helper="SuperPlane does not import existing issues."
        onCheckedChange={vi.fn()}
        testId="import-existing"
      />,
    );

    expect(screen.getByText(INTAKE_SKIP_INITIAL_IMPORT_COPY.label).closest("label")).toHaveClass("cursor-pointer");
  });

  it("matches the disabled checkbox so the label does not look clickable", async () => {
    const user = userEvent.setup();
    const onCheckedChange = vi.fn();
    render(
      <IntakeSkipInitialImportField
        checked
        disabled
        helper="SuperPlane imports the 30 newest open issues and scores them."
        onCheckedChange={onCheckedChange}
        testId="import-existing"
      />,
    );

    const label = screen.getByText(INTAKE_SKIP_INITIAL_IMPORT_COPY.label).closest("label");
    expect(label).toHaveClass("cursor-not-allowed");
    expect(label).toHaveClass("opacity-50");
    expect(screen.getByRole("checkbox", { name: /Import existing items/ })).toBeDisabled();

    await user.click(label!);
    expect(onCheckedChange).not.toHaveBeenCalled();
  });
});
