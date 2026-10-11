import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
} from "../__fixtures__/factoryPageResponses";
import { readLastVisitedFactory, recordLastVisitedFactory } from "../lib/lastVisitedFactory";
import { MobileFactoriesLayout } from "./MobileFactoriesLayout";

const factoryQuery = vi.hoisted(() => ({ error: undefined as Error | undefined }));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada" } }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => false, currentUserId: "account-1", isLoading: false }),
}));

vi.mock("@/hooks/useFactoryWebsocket", () => ({
  useFactoryWebsocket: () => undefined,
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => ({ data: [REFUND_FACTORY], isLoading: false, isFetching: false }),
  useFactory: () => ({
    data: factoryQuery.error ? undefined : REFUND_FACTORY,
    error: factoryQuery.error,
  }),
}));

vi.mock("./MobileBottomBar", () => ({
  MobileBottomBar: () => <nav data-testid="mobile-bottom-bar" />,
}));

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/velocity`]}>
      <Routes>
        <Route path="/:organizationId/workspaces/:factoryKey/*" element={<MobileFactoriesLayout />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("MobileFactoriesLayout", () => {
  beforeEach(() => {
    window.localStorage.clear();
    factoryQuery.error = undefined;
  });

  it("pins the shell to the visible screen instead of the layout viewport", async () => {
    renderLayout();

    const shell = await screen.findByTestId("mobile-factories-layout");
    expect(shell.className).not.toContain("h-dvh");
    expect(shell).toHaveStyle({ position: "fixed", left: "0px", right: "0px" });
    expect(shell.style.bottom).toBe("");
    expect(shell.style.top).not.toBe("");
    expect(shell.style.height).not.toBe("");
    expect(shell.querySelector("main")).toHaveClass("overscroll-none");
  });

  it("saves the workspace a phone visit opens", async () => {
    renderLayout();

    expect(await screen.findByTestId("mobile-factories-layout")).toBeInTheDocument();
    expect(readLastVisitedFactory("account-1", "org-1")).toBe(PRIMARY_FACTORY_ID);
  });

  it("clears a saved workspace when that workspace fails to load", async () => {
    recordLastVisitedFactory("account-1", "org-1", PRIMARY_FACTORY_ID);
    factoryQuery.error = new Error("not found");
    renderLayout();

    expect(await screen.findByTestId("factories-layout-error")).toBeInTheDocument();
    expect(readLastVisitedFactory("account-1", "org-1")).toBeNull();
  });
});
