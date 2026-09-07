import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FactoryAppCanvasMoreOptions } from "./FactoryAppCanvasMoreOptions";

describe("FactoryAppCanvasMoreOptions", () => {
  it("opens View YAML and Edit with a local agent from the menu", async () => {
    const user = userEvent.setup();
    const onViewYaml = vi.fn();
    const onEditWithLocalAgent = vi.fn();

    render(<FactoryAppCanvasMoreOptions onViewYaml={onViewYaml} onEditWithLocalAgent={onEditWithLocalAgent} />);

    await user.click(screen.getByTestId("factory-app-more-options"));
    await user.click(screen.getByTestId("factory-app-view-yaml"));
    expect(onViewYaml).toHaveBeenCalledTimes(1);

    await user.click(screen.getByTestId("factory-app-more-options"));
    await user.click(screen.getByTestId("factory-app-edit-local-agent"));
    expect(onEditWithLocalAgent).toHaveBeenCalledTimes(1);
  });

  it("hides Reset to factory defaults when no handler is given", async () => {
    const user = userEvent.setup();

    render(<FactoryAppCanvasMoreOptions onViewYaml={vi.fn()} onEditWithLocalAgent={vi.fn()} />);

    await user.click(screen.getByTestId("factory-app-more-options"));
    expect(screen.queryByTestId("factory-app-reset-defaults")).not.toBeInTheDocument();
  });

  it("renders Reset to factory defaults last and fires the handler", async () => {
    const user = userEvent.setup();
    const onResetToFactoryDefaults = vi.fn();

    render(
      <FactoryAppCanvasMoreOptions
        onViewYaml={vi.fn()}
        onEditWithLocalAgent={vi.fn()}
        onResetToFactoryDefaults={onResetToFactoryDefaults}
      />,
    );

    await user.click(screen.getByTestId("factory-app-more-options"));
    const items = screen.getAllByRole("menuitem");
    expect(items.at(-1)).toHaveAttribute("data-testid", "factory-app-reset-defaults");

    await user.click(screen.getByTestId("factory-app-reset-defaults"));
    expect(onResetToFactoryDefaults).toHaveBeenCalledTimes(1);
  });
});
