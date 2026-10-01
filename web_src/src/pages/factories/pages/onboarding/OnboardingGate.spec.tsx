import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { MemoryRouter, Outlet, Route, Routes, useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";

import { OnboardingGate } from "./OnboardingGate";
import { holdSetupAfterThisVisitCompletes } from "./onboardingGateState";

const FACTORY_KEY = "PAY";
const ROUTE_SEGMENT = "pay-k7m2xqab";
const BARE_SETUP_PATH = `/org-1/workspaces/${FACTORY_KEY}/setup`;
const BARE_OVERVIEW_PATH = `/org-1/workspaces/${FACTORY_KEY}/overview`;
const BARE_BOARD_PATH = `/org-1/workspaces/${FACTORY_KEY}/lines/line-plan`;
const SETUP_PATH = `/org-1/workspaces/${ROUTE_SEGMENT}/setup`;
const OVERVIEW_PATH = `/org-1/workspaces/${ROUTE_SEGMENT}/overview`;
const BOARD_PATH = `/org-1/workspaces/${ROUTE_SEGMENT}/lines/line-plan`;

let factory: FactoriesFactory;
let factoryId: string;

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId,
    factoryKey: FACTORY_KEY,
    routeSegment: ROUTE_SEGMENT,
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
            id: factoryId,
            lines: [{ id: "line-plan" }],
            onboarding: { completedAt: "2026-08-17T12:00:00Z" },
          };
          setTick((n) => n + 1);
        }}
      >
        complete
      </button>
      <button
        type="button"
        onClick={() => {
          factoryId = "factory-2";
          factory = { id: factoryId, onboarding: {} };
          setTick((n) => n + 1);
        }}
      >
        switch workspace
      </button>
      <button type="button" onClick={() => navigate(BARE_BOARD_PATH)}>
        open board
      </button>
      <button type="button" onClick={() => navigate(BARE_SETUP_PATH)}>
        open setup
      </button>
      <button
        type="button"
        onClick={() => {
          factoryId = "factory-2";
          factory = {
            id: factoryId,
            lines: [{ id: "line-plan" }],
            onboarding: { completedAt: "2026-08-17T12:00:00Z" },
          };
          setTick((n) => n + 1);
          navigate(BARE_BOARD_PATH);
        }}
      >
        switch to completed workspace
      </button>
      <button
        type="button"
        onClick={() => {
          factoryId = "factory-1";
          factory = {
            id: factoryId,
            lines: [{ id: "line-plan" }],
            onboarding: { completedAt: "2026-08-17T12:00:00Z" },
          };
          setTick((n) => n + 1);
          navigate(BARE_SETUP_PATH);
        }}
      >
        open first workspace setup
      </button>
      <Routes>
        <Route path="/:organizationId/workspaces/:factoryKey" element={<Layout />}>
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
        <Route path="/:organizationId/workspaces/:factoryKey" element={<Layout />}>
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
    factoryId = "factory-1";
    factory = { id: "factory-1", key: FACTORY_KEY, urlId: "k7m2xqab", onboarding: {} };
  });

  it("redirects an incomplete workspace to setup", async () => {
    renderRoute(BARE_OVERVIEW_PATH);

    expect(await screen.findByText(SETUP_PATH)).toBeInTheDocument();
  });

  it("sends an incomplete initial workspace from the org path to account onboarding", async () => {
    factory = { id: "factory-1", onboarding: { initial: true } };
    renderRoute(BARE_SETUP_PATH);

    expect(await screen.findByText("/onboarding")).toBeInTheDocument();
  });

  it("keeps a non-initial incomplete workspace on the org setup route", async () => {
    renderRoute(BARE_SETUP_PATH);

    expect(await screen.findByText(BARE_SETUP_PATH)).toBeInTheDocument();
  });

  it("redirects a completed workspace away from setup", async () => {
    factory = { id: "factory-1", onboarding: { completedAt: "2026-08-17T12:00:00Z" } };
    renderRoute(BARE_SETUP_PATH);

    expect(await screen.findByText(OVERVIEW_PATH)).toBeInTheDocument();
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
      <MemoryRouter initialEntries={[BARE_SETUP_PATH]}>
        <CompletionHarness />
      </MemoryRouter>,
    );
    expect(screen.getByText(BARE_SETUP_PATH)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "complete" }));
    expect(screen.getByText(BARE_SETUP_PATH)).toBeInTheDocument();
    expect(screen.queryByText(BARE_BOARD_PATH)).not.toBeInTheDocument();
  });

  it("redirects a completed workspace that returns to setup", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={[BARE_SETUP_PATH]}>
        <CompletionHarness />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "complete" }));
    await user.click(screen.getByRole("button", { name: "open board" }));
    await user.click(screen.getByRole("button", { name: "open setup" }));

    expect(await screen.findByText(BOARD_PATH)).toBeInTheDocument();
  });

  it("holds setup when a second workspace completes onboarding", async () => {
    const user = userEvent.setup();
    factory = {
      id: factoryId,
      lines: [{ id: "line-plan" }],
      onboarding: { completedAt: "2026-08-17T12:00:00Z" },
    };
    render(
      <MemoryRouter initialEntries={[BARE_BOARD_PATH]}>
        <CompletionHarness />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "switch workspace" }));
    expect(await screen.findByText(SETUP_PATH)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "complete" }));

    expect(screen.getByText(SETUP_PATH)).toBeInTheDocument();
  });

  it("does not retain a completed workspace hold after switching workspaces", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={[BARE_SETUP_PATH]}>
        <CompletionHarness />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "complete" }));
    await user.click(screen.getByRole("button", { name: "switch to completed workspace" }));
    await user.click(screen.getByRole("button", { name: "open first workspace setup" }));

    expect(await screen.findByText(BOARD_PATH)).toBeInTheDocument();
  });

  it("opens the line board when a completed workspace leaves setup", async () => {
    factory = {
      id: "factory-1",
      lines: [{ id: "line-plan" }],
      onboarding: { completedAt: "2026-08-17T12:00:00Z" },
    };
    renderRoute(BARE_SETUP_PATH);

    expect(await screen.findByText(BOARD_PATH)).toBeInTheDocument();
  });
});
