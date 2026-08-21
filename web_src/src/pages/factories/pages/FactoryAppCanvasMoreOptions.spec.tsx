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
});
