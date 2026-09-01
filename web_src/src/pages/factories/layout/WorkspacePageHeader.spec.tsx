import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { WorkspacePageHeader } from "./WorkspacePageHeader";

function renderHeader(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe("WorkspacePageHeader (section variant)", () => {
  it("renders the title and optional subtitle", () => {
    renderHeader(<WorkspacePageHeader title="Overview" subtitle="Your workspace at a glance." />);
    expect(screen.getByTestId("workspace-page-header-title")).toHaveTextContent("Overview");
    expect(screen.getByTestId("workspace-page-header-subtitle")).toHaveTextContent("Your workspace at a glance.");
    expect(screen.queryByTestId("workspace-page-header-back")).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-page-header-kicker")).not.toBeInTheDocument();
  });

  it("renders leading content next to the title", () => {
    renderHeader(
      <WorkspacePageHeader title="Tasks" leading={<span data-testid="scope-pills">All | Active | My</span>} />,
    );
    expect(screen.getByTestId("scope-pills")).toBeInTheDocument();
  });

  it("renders trailing actions and belowRow content", () => {
    renderHeader(
      <WorkspacePageHeader
        title="Tasks"
        actions={<button data-testid="new-button">New</button>}
        belowRow={<div data-testid="filter-chips">chips</div>}
      />,
    );
    expect(screen.getByTestId("workspace-page-header-actions")).toContainElement(screen.getByTestId("new-button"));
    expect(screen.getByTestId("filter-chips")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-page-header").children).toHaveLength(2);
  });

  it("keeps actions in the title row when actionsAlign is start", () => {
    renderHeader(
      <WorkspacePageHeader
        title="Tasks"
        actionsAlign="start"
        actions={<button data-testid="new-button">New</button>}
      />,
    );
    expect(screen.getByTestId("workspace-page-header").firstElementChild).toHaveClass("justify-start");
    expect(screen.getByTestId("workspace-page-header-actions")).toContainElement(screen.getByTestId("new-button"));
  });

  it("does not add a below-row slot when belowRow is omitted", () => {
    renderHeader(<WorkspacePageHeader title="Tasks" />);
    expect(screen.getByTestId("workspace-page-header").children).toHaveLength(1);
  });
});

describe("WorkspacePageHeader (entity variant)", () => {
  it("renders back link, kicker, title, and subtitle", () => {
    renderHeader(
      <WorkspacePageHeader
        variant="entity"
        title="Reconcile duplicate refunds"
        kicker="SP-42"
        subtitle="Ledger cleanup"
        backHref="/org-1/workspaces/SP/work-orders"
        backLabel="Tasks"
      />,
    );
    const back = screen.getByTestId("workspace-page-header-back");
    expect(back).toHaveAttribute("href", "/org-1/workspaces/SP/work-orders");
    expect(back).toHaveTextContent("Tasks");
    expect(screen.getByTestId("workspace-page-header-kicker")).toHaveTextContent("SP-42");
    expect(screen.getByTestId("workspace-page-header-title")).toHaveTextContent("Reconcile duplicate refunds");
    expect(screen.getByTestId("workspace-page-header-subtitle")).toHaveTextContent("Ledger cleanup");
  });

  it("omits the back link when backHref is not provided", () => {
    renderHeader(<WorkspacePageHeader variant="entity" title="Refund line" />);
    expect(screen.queryByTestId("workspace-page-header-back")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-page-header-title")).toHaveTextContent("Refund line");
  });

  it("omits the kicker when none is provided", () => {
    renderHeader(<WorkspacePageHeader variant="entity" title="Refund line" backHref="/lines" backLabel="Lines" />);
    expect(screen.queryByTestId("workspace-page-header-kicker")).not.toBeInTheDocument();
  });

  it("uses a custom back link test id when provided", () => {
    renderHeader(
      <WorkspacePageHeader
        variant="entity"
        title="Refund line"
        backHref="/lines"
        backLabel="Lines"
        backTestId="lines-back"
      />,
    );
    expect(screen.getByTestId("lines-back")).toBeInTheDocument();
  });
});
