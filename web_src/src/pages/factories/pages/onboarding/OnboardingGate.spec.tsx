import { render, screen } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { OnboardingGate } from "./OnboardingGate";

let factory: FactoriesFactory;

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    factory,
  }),
}));

function CurrentPath() {
  const { pathname, search } = useLocation();
  return <span>{`${pathname}${search}`}</span>;
}

function Layout() {
  return <Outlet />;
}

function renderRoute(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/org-1/workspaces/PAY" element={<Layout />}>
          <Route element={<OnboardingGate />}>
            <Route path="overview" element={<CurrentPath />} />
            <Route path="setup" element={<CurrentPath />} />
            <Route path="lines" element={<CurrentPath />} />
            <Route path="lines/:lineId" element={<CurrentPath />} />
          </Route>
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("OnboardingGate", () => {
  beforeEach(() => {
    factory = { id: "factory-1", onboarding: {} };
  });

  it("redirects an incomplete workspace to setup", async () => {
    renderRoute("/org-1/workspaces/PAY/overview");

    expect(await screen.findByText("/org-1/workspaces/PAY/setup")).toBeInTheDocument();
  });

  it("redirects a completed workspace away from setup", async () => {
    factory = { id: "factory-1", onboarding: { completedAt: "2026-08-17T12:00:00Z" } };
    renderRoute("/org-1/workspaces/PAY/setup");

    expect(await screen.findByText("/org-1/workspaces/PAY/overview")).toBeInTheDocument();
  });

  // Setup finishes with its own redirect to the line board. This redirect can
  // land after it, so both must open the same board.
  it("opens the line board when a completed workspace leaves setup", async () => {
    factory = {
      id: "factory-1",
      lines: [{ id: "line-plan" }],
      onboarding: { completedAt: "2026-08-17T12:00:00Z" },
    };
    renderRoute("/org-1/workspaces/PAY/setup");

    expect(await screen.findByText("/org-1/workspaces/PAY/lines/line-plan")).toBeInTheDocument();
  });
});
