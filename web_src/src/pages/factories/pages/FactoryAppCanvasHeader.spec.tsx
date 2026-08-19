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

  it("shows Edit in view mode", async () => {
    const user = userEvent.setup();
    const onOpenVisualEditor = vi.fn();

    render(
      <MemoryRouter>
        <FactoryAppCanvasHeader
          backHref="/back"
          backLabel="Plan and Implement"
          title="Refund Implementer"
          subtitle="Canvas"
          isConfigure={false}
          configureBusy={false}
          onDiscard={vi.fn()}
          onSave={vi.fn()}
          onOpenVisualEditor={onOpenVisualEditor}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("factory-app-view-yaml")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("factory-app-edit"));
    expect(onOpenVisualEditor).toHaveBeenCalledTimes(1);
  });

  it("marks edit mode with workspace toggles and more options", async () => {
    const user = userEvent.setup();
    const onAgentOpenChange = vi.fn();
    const onComponentsOpenChange = vi.fn();
    const onViewYaml = vi.fn();
    const onEditWithLocalAgent = vi.fn();

    render(
      <MemoryRouter>
        <FactoryAppCanvasHeader
          backHref="/back"
          backLabel="Plan and Implement"
          title="Refund Implementer"
          subtitle="Drag steps"
          isConfigure
          configureBusy={false}
          onDiscard={vi.fn()}
          onSave={vi.fn()}
          workspace={{
            agentOpen: false,
            componentsOpen: true,
            onAgentOpenChange,
            onComponentsOpenChange,
            onViewYaml,
            onEditWithLocalAgent,
          }}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("factory-app-canvas-header")).toHaveAttribute("data-editing", "true");
    expect(screen.getByTestId("factory-app-editing-badge")).toHaveTextContent("Editing");
    expect(screen.getByTestId("factory-app-workspace-components")).toHaveAttribute("aria-pressed", "true");

    const save = screen.getByTestId("factory-app-save");
    const discard = screen.getByTestId("factory-app-discard");
    const toggles = screen.getByTestId("factory-app-workspace-toggles");
    expect(save.compareDocumentPosition(toggles) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(discard.compareDocumentPosition(toggles) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(screen.getByTestId("factory-app-more-options"));
    await user.click(screen.getByTestId("factory-app-view-yaml"));
    expect(onViewYaml).toHaveBeenCalledTimes(1);
    await user.click(screen.getByTestId("factory-app-more-options"));
    await user.click(screen.getByTestId("factory-app-edit-local-agent"));
    expect(onEditWithLocalAgent).toHaveBeenCalledTimes(1);
  });
});
