import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router";

import { FactoryAppCanvasHeader } from "./FactoryAppCanvasHeader";

describe("FactoryAppCanvasHeader", () => {
  it("commits a draft title locally without waiting for Save", async () => {
    const user = userEvent.setup();
    const onDraftTitleChange = vi.fn();
    const onSave = vi.fn();

    render(
      <MemoryRouter>
        <FactoryAppCanvasHeader
          backHref="/back"
          backLabel="Automations"
          title="DFFDGD"
          subtitle="Drag steps"
          isConfigure
          configureBusy={false}
          canRename
          onDraftTitleChange={onDraftTitleChange}
          onDiscard={vi.fn()}
          onSave={onSave}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("factory-app-canvas-title"));
    const input = await screen.findByTestId("factory-app-rename-input");
    expect(input).toHaveValue("DFFDGD");
    await user.clear(input);
    await user.type(input, "Renamed automation");
    await user.keyboard("{Enter}");

    expect(onDraftTitleChange).toHaveBeenCalledWith("Renamed automation");
    expect(onSave).not.toHaveBeenCalled();
  });

  it("does not open rename when canRename is false", async () => {
    const user = userEvent.setup();
    const onDraftTitleChange = vi.fn();

    render(
      <MemoryRouter>
        <FactoryAppCanvasHeader
          backHref="/back"
          backLabel="Automations"
          title="DFFDGD"
          subtitle="Drag steps"
          isConfigure
          configureBusy={false}
          canRename={false}
          onDraftTitleChange={onDraftTitleChange}
          onDiscard={vi.fn()}
          onSave={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("factory-app-canvas-title"));
    expect(screen.queryByTestId("factory-app-rename-input")).not.toBeInTheDocument();
    expect(onDraftTitleChange).not.toHaveBeenCalled();
  });

  it("keeps the title static outside configure mode", async () => {
    const user = userEvent.setup();
    const onDraftTitleChange = vi.fn();

    render(
      <MemoryRouter>
        <FactoryAppCanvasHeader
          backHref="/back"
          backLabel="Automations"
          title="DFFDGD"
          subtitle="Canvas"
          isConfigure={false}
          configureBusy={false}
          canRename
          onDraftTitleChange={onDraftTitleChange}
          onDiscard={vi.fn()}
          onSave={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("factory-app-canvas-title"));
    expect(screen.queryByTestId("factory-app-rename-input")).not.toBeInTheDocument();
    expect(onDraftTitleChange).not.toHaveBeenCalled();
  });
});
