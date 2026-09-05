import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import type { VelocityAutomation } from "../lib/factoryVelocityReport";
import { VelocityAutomationsTable } from "./VelocityAutomationsTable";

const AUTOMATIONS: VelocityAutomation[] = [
  {
    id: "app-planner",
    name: "Planner",
    runs: 40,
    failed: 1,
    averageDurationHours: 2,
    averageCostUsd: 0.5,
    totalCostUsd: 20,
  },
  {
    id: "app-verifier",
    name: "Verifier",
    runs: 10,
    failed: 6,
    averageDurationHours: 0.25,
    averageCostUsd: 3,
    totalCostUsd: 30,
  },
];

function renderTable(automations: VelocityAutomation[] = AUTOMATIONS) {
  return render(
    <MemoryRouter>
      <VelocityAutomationsTable
        automations={automations}
        organizationId="org-1"
        factoryKey="refunds"
        periodLabel="Last 14 days"
      />
    </MemoryRouter>,
  );
}

/** The automation name of each body row, top to bottom. */
function rowNames(): string[] {
  const rows = within(screen.getByTestId("velocity-automations")).getAllByRole("row").slice(1);
  return rows.map((row) => within(row).getByRole("link").textContent ?? "");
}

describe("VelocityAutomationsTable", () => {
  it("starts on the busiest automation, which is the order the report arrives in", () => {
    renderTable();

    expect(rowNames()).toEqual(["Planner", "Verifier"]);
  });

  it("sorts on the column the reader clicks", async () => {
    const user = userEvent.setup();
    renderTable();

    await user.click(screen.getByRole("button", { name: "Total cost" }));

    expect(rowNames()).toEqual(["Verifier", "Planner"]);
  });

  it("turns the sort around on a second click of the same column", async () => {
    const user = userEvent.setup();
    renderTable();

    await user.click(screen.getByRole("button", { name: "Runs" }));

    expect(rowNames()).toEqual(["Verifier", "Planner"]);
  });

  it("marks the sorted column for assistive technology", async () => {
    const user = userEvent.setup();
    renderTable();

    // `aria-sort` belongs on the column header, not on the button inside it.
    const failed = screen.getByRole("columnheader", { name: "Failed" });
    expect(failed).toHaveAttribute("aria-sort", "none");

    await user.click(within(failed).getByRole("button"));
    expect(failed).toHaveAttribute("aria-sort", "descending");

    await user.click(within(failed).getByRole("button"));
    expect(failed).toHaveAttribute("aria-sort", "ascending");
  });

  it("links each automation to its detail page", () => {
    renderTable();

    expect(screen.getByRole("link", { name: "Planner" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/refunds/automations/app-planner",
    );
  });
});
