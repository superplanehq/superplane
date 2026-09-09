import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { MemoryRouter, Outlet, Route, Routes, useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { OnboardingGate } from "./OnboardingGate";
import { holdSetupAfterThisVisitCompletes } from "./onboardingGateState";

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

function CompletionHarness() {
  const [, setTick] = useState(0);
  const navigate = useNavigate();
  return (
    <>
      <button
        type="button"
        onClick={() => {
          factory = {
            id: "factory-1",
            lines: [{ id: "line-plan" }],
            onboarding: { completedAt: "2026-08-17T12:00:00Z" },
          };
          setTick((n) => n + 1);
        }}
      >
        complete
      </button>
      <button type="button" onClick={() => navigate("/org-1/workspaces/PAY/lines/line-plan")}>
        open board
      </button>
      <button type="button" onClick={() => navigate("/org-1/workspaces/PAY/setup")}>
        open setup
      </button>
      <Routes>
        <Route path="/org-1/workspaces/PAY" element={<Layout />}>
          <Route element={<OnboardingGate />}>
            <Route path="setup" element={<CurrentPath />} />
            <Route path="lines/:lineId" element={<CurrentPath />} />
          </Route>
        </Route>
      </Routes>
    </>
  );
}

function renderRoute(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/onboarding" element={<CurrentPath />} />
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

    expect(await screen.findByText("/org-1/workspaces/pay/setup")).toBeInTheDocument();
  });

  it("sends an incomplete initial workspace from the org path to account onboarding", async () => {
    factory = { id: "factory-1", onboarding: { initial: true } };
    renderRoute("/org-1/workspaces/PAY/setup");

    expect(await screen.findByText("/onboarding")).toBeInTheDocument();
  });

  it("keeps a non-initial incomplete workspace on the org setup route", async () => {
    renderRoute("/org-1/workspaces/PAY/setup");

    expect(await screen.findByText("/org-1/workspaces/PAY/setup")).toBeInTheDocument();
  });

  it("redirects a completed workspace away from setup", async () => {
    factory = { id: "factory-1", onboarding: { completedAt: "2026-08-17T12:00:00Z" } };
    renderRoute("/org-1/workspaces/PAY/setup");

    expect(await screen.findByText("/org-1/workspaces/pay/overview")).toBeInTheDocument();
  });

  // Setup finishes with its own redirect to the line board. This redirect can
  // land after it, so both must open the same board.
  // Finish writes completedAt before analysis mounts. This visit must stay
  // on setup; a later visit still leaves (the tests above).
  it("holds setup when this visit started incomplete", () => {
    expect(holdSetupAfterThisVisitCompletes(true, true)).toBe(true);
    expect(holdSetupAfterThisVisitCompletes(false, true)).toBe(false);
    expect(holdSetupAfterThisVisitCompletes(true, false)).toBe(false);
  });

  it("keeps setup mounted after this visit marks onboarding complete", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup"]}>
        <CompletionHarness />
      </MemoryRouter>,
    );
    expect(screen.getByText("/org-1/workspaces/PAY/setup")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "complete" }));
    expect(screen.getByText("/org-1/workspaces/PAY/setup")).toBeInTheDocument();
    expect(screen.queryByText("/org-1/workspaces/PAY/lines/line-plan")).not.toBeInTheDocument();
  });

  it("redirects a completed workspace that returns to setup", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup"]}>
        <CompletionHarness />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "complete" }));
    await user.click(screen.getByRole("button", { name: "open board" }));
    await user.click(screen.getByRole("button", { name: "open setup" }));

    expect(await screen.findByText("/org-1/workspaces/pay/lines/line-plan")).toBeInTheDocument();
  });

  it("opens the line board when a completed workspace leaves setup", async () => {
    factory = {
      id: "factory-1",
      lines: [{ id: "line-plan" }],
      onboarding: { completedAt: "2026-08-17T12:00:00Z" },
    };
    renderRoute("/org-1/workspaces/PAY/setup");

    expect(await screen.findByText("/org-1/workspaces/pay/lines/line-plan")).toBeInTheDocument();
  });
});
