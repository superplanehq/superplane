import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";

import OrganizationDetail from "./OrganizationDetail";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: { id: "account-1", name: "Admin", billing_enabled: false },
    loading: false,
  }),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: ["/admin/organizations/org-1"] },
        createElement(Routes, null, createElement(Route, { path: "/admin/organizations/:orgId", element: children })),
      ),
    );
  return render(createElement(OrganizationDetail), { wrapper });
}

describe("OrganizationDetail billing off", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps the Spending report when product billing is off", () => {
    renderPage();

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toContain("Spending");
  });
});
