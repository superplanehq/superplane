import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";

import { OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { splitRunFixtureForWorkOrder } from "../splitRunMocks";
import type { SplitRunSource } from "../splitRunSource";
import { outcomeSummary } from "./automationsViewModel";
import { ConsoleSummaryPanel } from "./ConsoleSummaryPanel";

const automationSource: SplitRunSource = {
  kind: "automation",
  name: "ESLint AutoFix",
  iconSrc: "/superplane-mark.svg",
  iconAlt: "SuperPlane",
  run: { appId: "canvas-eslint", runId: "run-eslint-1" },
};

function renderWithSource(source?: SplitRunSource) {
  const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks: [], demoArtifacts: false });
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/tasks/1"]}>
      <Routes>
        <Route
          path="/:organizationId/workspaces/:factoryKey/*"
          element={
            <ConsoleSummaryPanel fixture={fixture} outcome={outcomeSummary(fixture)} stages={[]} source={source} />
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("ConsoleSummaryPanel source run", () => {
  it("links the task to the run that created it", () => {
    renderWithSource(automationSource);

    const row = screen.getByText("Source run").parentElement;
    expect(row).not.toBeNull();
    const link = within(row as HTMLElement).getByRole("link", { name: "View run" });
    expect(link.getAttribute("href")).toContain("run-eslint-1");
  });

  it("omits the source run row when the task has no source run", () => {
    renderWithSource({ ...automationSource, run: undefined });

    expect(screen.queryByText("Source run")).not.toBeInTheDocument();
  });
});
