import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FactoryAppCanvasWorkspaceToggles } from "./FactoryAppCanvasWorkspaceToggles";

describe("FactoryAppCanvasWorkspaceToggles", () => {
  it("toggles Agent and Components independently", async () => {
    const user = userEvent.setup();
    const onAgentOpenChange = vi.fn();
    const onComponentsOpenChange = vi.fn();

    render(
      <FactoryAppCanvasWorkspaceToggles
        agentOpen={false}
        componentsOpen
        onAgentOpenChange={onAgentOpenChange}
        onComponentsOpenChange={onComponentsOpenChange}
      />,
    );

    expect(screen.getByTestId("factory-app-workspace-agent")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByTestId("factory-app-workspace-components")).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByTestId("factory-app-workspace-yaml")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("factory-app-workspace-agent"));
    expect(onAgentOpenChange).toHaveBeenCalledWith(true);

    await user.click(screen.getByTestId("factory-app-workspace-components"));
    expect(onComponentsOpenChange).toHaveBeenCalledWith(false);
  });
});
