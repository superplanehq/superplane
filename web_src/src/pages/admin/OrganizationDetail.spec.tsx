import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";

import OrganizationDetail from "./OrganizationDetail";

const ORG_ID = "org-1";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Admin" } }),
}));

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function renderPage() {
  const queryClient = createQueryClient();
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: [`/admin/organizations/${ORG_ID}`] },
        createElement(Routes, null, createElement(Route, { path: "/admin/organizations/:orgId", element: children })),
      ),
    );
  return render(<OrganizationDetail />, { wrapper });
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("OrganizationDetail", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/users`)) {
          return jsonResponse({
            items: [{ id: "user-1", name: "Ada Lovelace", email: "ada@example.com", account_id: "acc-1" }],
            total: 1,
          });
        }
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/canvases`)) {
          return jsonResponse({
            items: [{ id: "canvas-1", name: "Deploy pipeline", description: "Deploys the app" }],
            total: 1,
          });
        }
        if (url === `/admin/api/organizations/${ORG_ID}/experimental-features`) {
          return jsonResponse({
            features: [{ id: "factories", label: "Factories", description: "Software factories", released: false }],
            enabled: ["factories"],
          });
        }
        if (url.includes("/llm-credit")) {
          return jsonResponse({
            remaining_credit_cents: 5000,
            grant_total_cents: 5000,
            superplane_grant_cents: 5000,
            purchased_credit_cents: 0,
            hosted_billed_cents: 0,
            markup_bps: 2000,
            markup_override_bps: null,
            warning: false,
          });
        }
        if (url.includes("/billing-plan")) {
          return jsonResponse({
            plan: "trial",
            plan_source: "system",
            polar_subscription_status: "",
            polar_managed: false,
            trial_ends_at: null,
            current_period_end: null,
          });
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows users first and opens automations after a tab click", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Users",
      "Automations",
      "Features",
      "Credits",
    ]);
    expect(await screen.findByPlaceholderText("Search users...")).toBeInTheDocument();
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search automations...")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(await screen.findByPlaceholderText("Search automations...")).toBeInTheDocument();
    expect(await screen.findByText("Deploy pipeline")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search users...")).not.toBeInTheDocument();
  });

  it("opens features and credits after tab clicks", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Features" }));

    expect(await screen.findByText("Factories")).toBeVisible();
    expect(screen.getByRole("switch", { name: "Toggle Factories" })).toBeVisible();
    expect(screen.queryByPlaceholderText("Search users...")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Credits" }));

    expect(await screen.findByText("Hosted credit")).toBeVisible();
    expect(await screen.findByTestId("admin-org-credit-amount")).toBeVisible();
    expect(screen.queryByText("Factories")).not.toBeInTheDocument();
  });

  it("keeps unsaved credit edits after switching tabs", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("tab", { name: "Credits" }));

    const amount = await screen.findByTestId("admin-org-credit-amount");
    await user.clear(amount);
    await user.type(amount, "12.50");

    await user.click(screen.getByRole("tab", { name: "Users" }));
    expect(await screen.findByText("Ada Lovelace")).toBeVisible();
    expect(screen.getByRole("tabpanel", { name: "Credits" })).toHaveAttribute("data-state", "inactive");
    expect(screen.getByTestId("admin-org-credit-amount")).toHaveValue("12.50");

    await user.click(screen.getByRole("tab", { name: "Credits" }));
    expect(screen.getByRole("tabpanel", { name: "Credits" })).toHaveAttribute("data-state", "active");
    expect(await screen.findByTestId("admin-org-credit-amount")).toHaveValue("12.50");
  });
});
