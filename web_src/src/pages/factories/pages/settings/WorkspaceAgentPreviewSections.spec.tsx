import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import { WorkspaceDefaultModelSection } from "./WorkspaceDefaultModelSection";
import { WorkspaceSetupStepsSection } from "./WorkspaceSetupStepsSection";

describe("workspace agent preview fields", () => {
  it("opens the model selector from its label", async () => {
    const user = userEvent.setup();
    render(<WorkspaceDefaultModelSection canUpdate models={[{ value: "claude-sonnet", label: "Claude Sonnet" }]} />);

    expect(screen.getByRole("combobox", { name: "Default model" })).toBeInTheDocument();
    await user.click(screen.getByText("Default model"));

    expect(screen.getByRole("listbox")).toBeInTheDocument();
  });

  it("focuses the setup command field from its label", async () => {
    const user = userEvent.setup();
    render(<WorkspaceSetupStepsSection canUpdate />);

    await user.click(screen.getByText("Setup steps"));

    expect(screen.getByRole("textbox", { name: "Setup steps" })).toHaveFocus();
  });
});
