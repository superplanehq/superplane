import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ColumnLaneMenu } from "./ColumnLaneMenu";

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="lines-test-location">{location.pathname}</div>;
}

describe("ColumnLaneMenu", () => {
  it("offers colour swatches and applies a selection", async () => {
    const onColorChange = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          onEdit={vi.fn()}
          colorId={null}
          onColorChange={onColorChange}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.getByText("Set color")).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-menu-color-lime")).toHaveAttribute("aria-label", "Lime");

    await user.click(screen.getByTestId("lines-backlog-menu-color-lime"));
    expect(onColorChange).toHaveBeenCalledWith("lime");
  });

  it("removes the colour when requested", async () => {
    const onColorChange = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu title="Plan" testId="lines-phase-menu-0" colorId="sky" onColorChange={onColorChange} />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-color-sky")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("lines-phase-menu-0-color-remove"));
    expect(onColorChange).toHaveBeenCalledWith(null);
  });

  it("calls the inline Edit action when it is supplied", async () => {
    const onEdit = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          editHref="/canvas-edit"
          onEdit={onEdit}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));
    expect(onEdit).toHaveBeenCalledTimes(1);
  });

  it("offers a separate Edit Agent action", async () => {
    const onEditAgent = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Plan"
          testId="lines-phase-menu-0"
          editHref="/canvas-edit"
          editLabel="Edit Automation"
          onEditAgent={onEditAgent}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    const editAgent = screen.getByTestId("lines-phase-menu-0-edit-agent");
    const editAutomation = screen.getByTestId("lines-phase-menu-0-edit");
    expect(editAgent).toHaveTextContent("Edit Agent");
    expect(editAutomation).toHaveTextContent("Edit Automation");
    // Editing the agent is the common task, so it leads the menu.
    expect(editAgent.compareDocumentPosition(editAutomation) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    await user.click(editAgent);

    expect(onEditAgent).toHaveBeenCalledTimes(1);
  });

  it("uses the supplied Edit label for canvas columns", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Plan"
          testId="lines-phase-menu-0"
          editHref="/canvas-edit"
          editLabel="Edit Automation"
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-edit")).toHaveTextContent("Edit Automation");
  });

  it("offers Set parallelism with the current value", async () => {
    const onSetParallelism = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Implement"
          testId="lines-phase-menu-1"
          editHref="/canvas-edit"
          editLabel="Edit Automation"
          onSetParallelism={onSetParallelism}
          parallelism={10}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-1"));
    expect(screen.getByTestId("lines-phase-menu-1-parallelism")).toHaveTextContent("Set parallelism (10)");
    await user.click(screen.getByTestId("lines-phase-menu-1-parallelism"));
    expect(onSetParallelism).toHaveBeenCalledTimes(1);
  });

  it("hides Edit when the column has no editor", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu title="Done" testId="lines-phase-menu-3" colorId={null} onColorChange={vi.fn()} />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-3"));
    expect(screen.queryByTestId("lines-phase-menu-3-edit")).not.toBeInTheDocument();
  });

  it("offers a separate Edit automation action above Edit for the Backlog shape", async () => {
    const onEdit = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          automationHref="/apps/app-refund-backlog/configure"
          onEdit={onEdit}
          colorId={null}
          onColorChange={vi.fn()}
        />
        <LocationDisplay />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    const editAutomation = screen.getByTestId("lines-backlog-menu-edit-automation");
    const edit = screen.getByTestId("lines-backlog-menu-edit");
    expect(editAutomation).toHaveTextContent("Edit automation");
    expect(edit).toHaveTextContent("Edit");
    // Edit automation leads the menu, ahead of the name/size Edit item.
    expect(editAutomation.compareDocumentPosition(edit) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(editAutomation);
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("/apps/app-refund-backlog/configure");
    expect(onEdit).not.toHaveBeenCalled();
  });

  it("offers Add intake ahead of the other actions when supplied", async () => {
    const onAddIntake = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          onEdit={vi.fn()}
          onAddIntake={onAddIntake}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    const addIntake = screen.getByTestId("lines-backlog-menu-add-intake");
    expect(addIntake).toHaveTextContent("Add intake");
    const edit = screen.getByTestId("lines-backlog-menu-edit");
    expect(addIntake.compareDocumentPosition(edit) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(addIntake);
    expect(onAddIntake).toHaveBeenCalledTimes(1);
  });

  it("hides Add intake when it is not supplied", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          onEdit={vi.fn()}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.queryByTestId("lines-backlog-menu-add-intake")).not.toBeInTheDocument();
  });

  it("hides Edit automation when no automation href is supplied", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          onEdit={vi.fn()}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.queryByTestId("lines-backlog-menu-edit-automation")).not.toBeInTheDocument();
  });

  it("offers Archive Step as the last item after Set color", async () => {
    const onArchiveStep = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Plan"
          testId="lines-phase-menu-0"
          onSetParallelism={vi.fn()}
          onArchiveStep={onArchiveStep}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    const archive = screen.getByTestId("lines-phase-menu-0-archive");
    expect(archive).toHaveTextContent("Archive Step");
    expect(
      screen.getByTestId("lines-phase-menu-0-color-picker").compareDocumentPosition(archive) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.click(archive);
    expect(onArchiveStep).toHaveBeenCalledTimes(1);
  });

  it("hides Archive Step when the callback is not supplied", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu title="Plan" testId="lines-phase-menu-0" colorId={null} onColorChange={vi.fn()} />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.queryByTestId("lines-phase-menu-0-archive")).not.toBeInTheDocument();
  });
});
