import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ColumnLaneMenu } from "./ColumnLaneMenu";

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

  it("offers Add automation ahead of the other actions when supplied", async () => {
    const onAddAutomation = vi.fn();
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ColumnLaneMenu
          title="Backlog"
          testId="lines-backlog-menu"
          onEdit={vi.fn()}
          onAddAutomation={onAddAutomation}
          colorId={null}
          onColorChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("lines-backlog-menu"));
    const addAutomation = screen.getByTestId("lines-backlog-menu-add-automation");
    expect(addAutomation).toHaveTextContent("Add automation");
    const edit = screen.getByTestId("lines-backlog-menu-edit");
    expect(addAutomation.compareDocumentPosition(edit) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(addAutomation);
    expect(onAddAutomation).toHaveBeenCalledTimes(1);
  });

  it("hides Add automation when it is not supplied", async () => {
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
    expect(screen.queryByTestId("lines-backlog-menu-add-automation")).not.toBeInTheDocument();
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
});
