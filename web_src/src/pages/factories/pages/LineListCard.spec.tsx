import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { REFUND_FACTORY_LINES, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
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
});
