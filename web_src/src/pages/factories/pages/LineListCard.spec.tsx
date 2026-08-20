import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { REFUND_FACTORY_LINES, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
import type { LineCardActions } from "./lineCardActions";
import { LINE_LIST_DESCRIPTION_BY_ID, LINE_LIST_METRICS_BY_ID } from "./lineListMetricsMockData";
import { LineListCard, LineListHeroSplit } from "./LineListCard";

const planLine = REFUND_FACTORY_LINES[0];
const planMetrics = LINE_LIST_METRICS_BY_ID[REFUND_LINE_PLAN_ID] ?? null;
const planDescription = LINE_LIST_DESCRIPTION_BY_ID[REFUND_LINE_PLAN_ID];

describe("LineListCard", () => {
  it("shows a purpose description and no phase names", () => {
    render(
      <MemoryRouter>
        <LineListCard line={planLine} href="/lines/x" metrics={planMetrics} description={planDescription} />
      </MemoryRouter>,
    );

    expect(screen.getByText(planDescription)).toBeInTheDocument();
    expect(screen.queryByText("Plan → Implement → Verify")).not.toBeInTheDocument();
  });

  it("splits success sparkline and completion bars", () => {
    render(<LineListHeroSplit metrics={planMetrics} />);

    const split = screen.getByTestId("lines-card-metrics");
    expect(split).toHaveTextContent("82%");
    expect(split).toHaveTextContent("+6 pts");
    expect(split).toHaveTextContent("1.4 per day");
    expect(split).toHaveTextContent("Success rate");
    expect(split).toHaveTextContent("Completions");
    expect(split).toHaveTextContent("Duration");
    expect(split).toHaveTextContent("15m");
    expect(split).toHaveTextContent("−2m");
    expect(split).toHaveTextContent("$3.20");
  });

  it("shows zero success rate and completions when the line has no metrics", () => {
    render(<LineListHeroSplit metrics={null} />);

    const split = screen.getByTestId("lines-card-metrics");
    expect(split).toHaveTextContent("0%");
    expect(split).toHaveTextContent("0 per day");
    expect(split).toHaveTextContent("—");
  });

  it("omits the menu entirely when no actions are passed", () => {
    render(
      <MemoryRouter>
        <LineListCard line={planLine} href="/lines/x" metrics={planMetrics} description={planDescription} />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("lines-card-menu")).not.toBeInTheDocument();
  });
});

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="current-path">{location.pathname}</div>;
}

function renderCardWithMenu(actions: LineCardActions, href = "/lines/plan-and-implement") {
  return render(
    <MemoryRouter initialEntries={["/start"]}>
      <Routes>
        <Route
          path="/start"
          element={
            <>
              <LineListCard line={planLine} href={href} metrics={planMetrics} actions={actions} />
              <LocationProbe />
            </>
          }
        />
        <Route path={href} element={<div data-testid="line-detail-page">Line detail</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("LineListCard menu", () => {
  function makeActions(overrides: Partial<LineCardActions> = {}): LineCardActions {
    return {
      onEdit: vi.fn(),
      onDuplicate: vi.fn(),
      canEdit: true,
      canDuplicate: true,
      ...overrides,
    };
  }

  it("hides the menu trigger until interacted with", () => {
    renderCardWithMenu(makeActions());

    expect(screen.getByTestId("lines-card-menu")).toHaveClass("opacity-0");
  });

  it("shows Edit and Duplicate, and no Delete item", async () => {
    const user = userEvent.setup();
    renderCardWithMenu(makeActions());

    await user.click(screen.getByTestId("lines-card-menu"));

    expect(screen.getByTestId("lines-card-edit")).toHaveTextContent("Edit");
    expect(screen.getByTestId("lines-card-duplicate")).toHaveTextContent("Duplicate");
    expect(screen.queryByText("Delete")).not.toBeInTheDocument();
  });

  it("clicking Edit calls onEdit and does not navigate the card to its href", async () => {
    const user = userEvent.setup();
    const actions = makeActions();
    renderCardWithMenu(actions);

    await user.click(screen.getByTestId("lines-card-menu"));
    await user.click(screen.getByTestId("lines-card-edit"));

    expect(actions.onEdit).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("current-path")).toHaveTextContent("/start");
    expect(screen.queryByTestId("line-detail-page")).not.toBeInTheDocument();
  });

  it("clicking Duplicate calls onDuplicate and does not navigate the card", async () => {
    const user = userEvent.setup();
    const actions = makeActions();
    renderCardWithMenu(actions);

    await user.click(screen.getByTestId("lines-card-menu"));
    await user.click(screen.getByTestId("lines-card-duplicate"));

    expect(actions.onDuplicate).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("current-path")).toHaveTextContent("/start");
  });

  it("keyboard activation of the menu trigger does not navigate the card", async () => {
    const user = userEvent.setup();
    renderCardWithMenu(makeActions());

    const trigger = screen.getByTestId("lines-card-menu");
    trigger.focus();
    expect(trigger).toHaveFocus();
    await user.keyboard("{Enter}");

    expect(screen.getByTestId("current-path")).toHaveTextContent("/start");
    expect(screen.queryByTestId("line-detail-page")).not.toBeInTheDocument();
  });

  it("disables Edit and Duplicate without permission", async () => {
    const user = userEvent.setup();
    renderCardWithMenu(makeActions({ canEdit: false, canDuplicate: false }));

    await user.click(screen.getByTestId("lines-card-menu"));

    expect(screen.getByTestId("lines-card-edit")).toHaveAttribute("data-disabled");
    expect(screen.getByTestId("lines-card-duplicate")).toHaveAttribute("data-disabled");
  });
});
