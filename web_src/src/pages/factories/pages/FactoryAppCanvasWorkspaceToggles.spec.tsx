import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FactoryAppCanvasWorkspaceToggles } from "./FactoryAppCanvasWorkspaceToggles";

describe("FactoryAppCanvasWorkspaceToggles", () => {
  it("toggles Components and hides the managed-agent control", async () => {
    const user = userEvent.setup();
    const onComponentsOpenChange = vi.fn();

    render(<FactoryAppCanvasWorkspaceToggles componentsOpen onComponentsOpenChange={onComponentsOpenChange} />);

    expect(screen.queryByTestId("factory-app-workspace-agent")).not.toBeInTheDocument();
    expect(screen.getByTestId("factory-app-workspace-components")).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByTestId("factory-app-workspace-yaml")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("factory-app-workspace-components"));
    expect(onComponentsOpenChange).toHaveBeenCalledWith(false);
  });
});
