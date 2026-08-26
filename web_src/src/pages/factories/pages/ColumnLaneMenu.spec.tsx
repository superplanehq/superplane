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
    expect(screen.getByTestId("lines-phase-menu-0-edit")).toHaveTextContent("Edit Automation");
    expect(screen.getByTestId("lines-phase-menu-0-edit-agent")).toHaveTextContent("Edit Agent");
    await user.click(screen.getByTestId("lines-phase-menu-0-edit-agent"));

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
});
