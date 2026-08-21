import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { LineIntakeDrawer } from "./LineIntakeDrawer";

function renderDrawer(props: { onClose?: () => void } = {}) {
  return render(
    <MemoryRouter>
      <LineIntakeDrawer onClose={props.onClose ?? vi.fn()} />
    </MemoryRouter>,
  );
}

describe("LineIntakeDrawer", () => {
  it("lists GitHub, Sentry, and PagerDuty as intake sources", () => {
    renderDrawer();

    const drawer = screen.getByTestId("line-intake-drawer");
    expect(drawer).toHaveAccessibleName("Intake");
    expect(screen.getByRole("heading", { name: "Intake" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-source-github-issues")).toHaveTextContent("GitHub issues");
    expect(screen.getByTestId("line-intake-source-sentry-exceptions")).toHaveTextContent("Sentry exceptions");
    expect(screen.getByTestId("line-intake-source-pagerduty-incidents")).toHaveTextContent("PagerDuty incidents");
  });

  it("closes the drawer from the header control", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderDrawer({ onClose });

    await user.click(screen.getByTestId("line-intake-close"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("opens a searchable picker with six intake templates", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByTestId("line-intake-add"));

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getByRole("heading", { name: "Add intake" })).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-search")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-improve-ci-runtime")).toHaveTextContent(
      "Improve CI runtime",
    );
    expect(within(picker).getByTestId("add-intake-template-improve-page-performance")).toHaveTextContent(
      "Improve page performance",
    );
    expect(within(picker).getAllByTestId(/^add-intake-template-/)).toHaveLength(6);
  });

  it("filters templates from the picker search", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByTestId("line-intake-add"));
    await user.type(screen.getByTestId("add-intake-search"), "runtime");

    expect(screen.getByTestId("add-intake-template-improve-ci-runtime")).toBeInTheDocument();
    expect(screen.queryByTestId("add-intake-template-flaky-tests")).not.toBeInTheDocument();
  });

  it("opens the intake automation popup from a source card", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: "Open GitHub issues intake" }));

    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "GitHub issues" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-listen")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-evaluate")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Accepted events go to Backlog" })).toBeInTheDocument();
  });
});
