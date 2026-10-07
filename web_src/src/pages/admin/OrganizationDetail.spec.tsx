import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
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
        createElement(
          Routes,
          null,
          createElement(Route, { path: "/admin/organizations/:orgId", element: children }),
          createElement(Route, { path: "/admin", element: createElement("p", null, "All organizations page") }),
        ),
      ),
    );
  return { queryClient, ...render(<OrganizationDetail />, { wrapper }) };
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
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url === "/admin/api/accounts/acc-1" && init?.method === "DELETE") {
          return jsonResponse({ status: "deleted", deleted_organization_ids: [ORG_ID] });
        }
        if (url === `/admin/api/organizations/${ORG_ID}`) {
          return jsonResponse({
            id: ORG_ID,
            name: "Acme",
            slug: "acme",
            description: "Builds widgets",
            canvas_count: 2,
            task_count: 4,
            done_task_count: 6,
            member_count: 3,
            created_at: "2024-01-15T12:00:00Z",
            updated_at: "2024-02-20T12:00:00Z",
          });
        }
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
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/integrations`)) {
          return jsonResponse({
            items: [
              {
                id: "integration-1",
                app_name: "sentry",
                installation_name: "acme-sentry",
                state: "error",
                state_description: "Sentry is not sending issue events.",
                details: { installation_uuid: "install-uuid-1", external_organization: "acme-sentry-org" },
                created_at: "2024-01-15T12:00:00Z",
                updated_at: "2024-02-20T12:00:00Z",
              },
            ],
            total: 1,
            limit: 50,
            offset: 0,
          });
        }
        if (url === `/admin/api/organizations/${ORG_ID}/experimental-features`) {
          return jsonResponse({
            features: [{ id: "factories", label: "Factories", description: "Software factories", released: false }],
            enabled: ["factories"],
          });
        }
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/spending-report`)) {
          return jsonResponse({
            kpiTotals: {
              costCents: "100",
              totalTokens: "10",
              durationSeconds: "5",
              hostedCostCents: "100",
              byokCostCents: "0",
            },
            explorerTotals: {
              costCents: "100",
              totalTokens: "10",
              durationSeconds: "5",
              hostedCostCents: "100",
              byokCostCents: "0",
            },
            series: [],
            seriesKeys: [],
            breakdown: [],
            credit: { remainingCreditCents: "5000", grantTotalCents: "5000" },
            catalogs: { workspaces: [], users: [], models: [], machines: [] },
          });
        }
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/velocity`)) {
          return jsonResponse(adminVelocityReport());
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

  it("renders overview values on load", async () => {
    renderPage();

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Overview",
      "Users",
      "Automations",
      "Connections",
      "Features",
      "Spending",
      "Velocity",
      "Credits",
    ]);
    expect(await screen.findByText("Acme")).toBeInTheDocument();
    const panel = screen.getByRole("tabpanel", { name: "Overview" });
    expect(within(panel).getByText("Acme")).toBeInTheDocument();
    expect(within(panel).getByText("acme")).toBeInTheDocument();
    expect(within(panel).getByText(ORG_ID)).toBeInTheDocument();
    expect(within(panel).getByText("Builds widgets")).toBeInTheDocument();
    expect(within(panel).getByText("3")).toBeInTheDocument();
    expect(within(panel).getByText("2")).toBeInTheDocument();
    expect(within(panel).getByText("4")).toBeInTheDocument();
    expect(within(panel).getByText("6")).toBeInTheDocument();
    expect(within(panel).getByText("Name")).toBeInTheDocument();
    expect(within(panel).getByText("Slug")).toBeInTheDocument();
    expect(within(panel).getByText("Organization ID")).toBeInTheDocument();
    expect(within(panel).getByText("Description")).toBeInTheDocument();
    expect(within(panel).getByText("Created")).toBeInTheDocument();
    expect(within(panel).getByText("Updated")).toBeInTheDocument();
    expect(within(panel).getByText("Members")).toBeInTheDocument();
    expect(within(panel).getByText("Automations")).toBeInTheDocument();
    expect(within(panel).getByText("Tasks")).toBeInTheDocument();
    expect(within(panel).getByText("Done Tasks")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search users...")).not.toBeInTheDocument();
  });

  it("loads users after a tab switch", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search users...")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Users" }));

    expect(await screen.findByPlaceholderText("Search users...")).toBeInTheDocument();
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("user-1")).toBeInTheDocument();
    expect(screen.getByText("acc-1")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy user id" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy account id" })).toBeInTheDocument();
  });

  it("deletes a user account and leaves the deleted organization", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("tab", { name: "Users" }));
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(await screen.findByText(/You cannot undo this action/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete Account" }));

    expect(await screen.findByText("All organizations page")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(
      "/admin/api/accounts/acc-1",
      expect.objectContaining({ method: "DELETE", credentials: "include" }),
    );
  });

  it("shows overview first and opens automations after a tab click", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search automations...")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(await screen.findByPlaceholderText("Search automations...")).toBeInTheDocument();
    expect(await screen.findByText("Deploy pipeline")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search users...")).not.toBeInTheDocument();
  });

  it("loads spending from the admin API after the spending tab opens", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    const urlsBefore = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
    expect(urlsBefore.some((url) => url.includes("spending-report"))).toBe(false);

    await user.click(screen.getByRole("tab", { name: "Spending" }));

    expect(await screen.findByTestId("spending-redesign-page")).toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-hosted")).toBeInTheDocument();

    const urls = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
    const spendingCalls = urls.filter((url) => url.includes("spending-report"));
    expect(spendingCalls.length).toBeGreaterThan(0);
    expect(spendingCalls.every((url) => url.startsWith(`/admin/api/organizations/${ORG_ID}/spending-report`))).toBe(
      true,
    );
    expect(spendingCalls.some((url) => url.includes("usageKind=model"))).toBe(true);
    expect(spendingCalls.some((url) => url.includes("usageKind=compute"))).toBe(true);
    expect(urls.some((url) => url.includes("/api/v1/organizations/"))).toBe(false);
  });

  it("opens velocity from the admin API and hides people, sync, and cycle time", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    const urlsBefore = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
    expect(urlsBefore.some((url) => url.includes("/velocity"))).toBe(false);

    await user.click(screen.getByRole("tab", { name: "Velocity" }));

    expect(await screen.findByTestId("velocity-summary")).toBeInTheDocument();
    expect(screen.getByText("Tasks closed")).toBeInTheDocument();
    expect(screen.getByText("Task waste")).toBeInTheDocument();
    expect(screen.getByText("Cost per task")).toBeInTheDocument();
    expect(screen.getByTestId("velocity-delivery")).toBeInTheDocument();
    expect(screen.getByTestId("velocity-cost")).toBeInTheDocument();
    expect(screen.getByTestId("velocity-task-cost")).toBeInTheDocument();
    const automations = screen.getByTestId("velocity-automations");
    expect(within(automations).getByText("Nightly close")).toBeInTheDocument();
    expect(within(automations).queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("Refunds")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-people")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "People" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("velocity-overflow-menu")).not.toBeInTheDocument();
    expect(screen.queryByTestId("velocity-refresh-data")).not.toBeInTheDocument();
    expect(screen.queryByTestId("velocity-sync-progress")).not.toBeInTheDocument();
    expect(screen.queryByText("Median cycle time")).not.toBeInTheDocument();
    expect(screen.queryByTestId("velocity-task-time")).not.toBeInTheDocument();
    expect(screen.queryByText("Manual work")).not.toBeInTheDocument();

    const urls = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
    const velocityCalls = urls.filter((url) => url.includes("/velocity"));
    expect(velocityCalls).toEqual([`/admin/api/organizations/${ORG_ID}/velocity?period_days=30`]);
    expect(urls.some((url) => url.includes("/api/v1/"))).toBe(false);
  });

  it("says when the organization has no workspaces", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organizationOverview());
      }
      if (url.startsWith(`/admin/api/organizations/${ORG_ID}/velocity`)) {
        return jsonResponse({ factories: [] });
      }
      return new Response("not found", { status: 404 });
    });
    renderPage();

    await user.click(await screen.findByRole("tab", { name: "Velocity" }));

    expect(await screen.findByText("This organization has no workspaces.")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-summary")).not.toBeInTheDocument();
  });

  it("says when the selected workspace has no velocity in the period", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organizationOverview());
      }
      if (url.startsWith(`/admin/api/organizations/${ORG_ID}/velocity`)) {
        return jsonResponse({
          factories: [{ id: "factory-1", name: "Quiet" }],
          factoryId: "factory-1",
          periodDays: 30,
          hasPeopleCohort: false,
          hasPreviousWindow: false,
          totals: {},
          previousTotals: {},
          points: [],
          intakeSources: [],
          automations: [],
        });
      }
      return new Response("not found", { status: 404 });
    });
    renderPage();

    await user.click(await screen.findByRole("tab", { name: "Velocity" }));

    expect(await screen.findByText("There is no velocity in this period.")).toBeInTheDocument();
    expect(screen.getByText("Quiet")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "30d" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByTestId("velocity-summary")).not.toBeInTheDocument();
  });

  it("shows a failed refresh next to the saved velocity report", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderPage();

    await user.click(await screen.findByRole("tab", { name: "Velocity" }));
    expect(await screen.findByTestId("velocity-summary")).toBeInTheDocument();

    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.startsWith(`/admin/api/organizations/${ORG_ID}/velocity`)) {
        return new Response("error", { status: 500 });
      }
      return new Response("not found", { status: 404 });
    });
    await queryClient.invalidateQueries({ queryKey: ["admin", "organizations", ORG_ID, "velocity"] });

    expect(await screen.findByTestId("admin-velocity-refresh-error")).toBeInTheDocument();
    expect(screen.getByText("Could not refresh velocity.")).toBeInTheDocument();
    expect(screen.getByText(/^Last loaded /)).toBeInTheDocument();
    expect(screen.getByTestId("velocity-summary")).toBeInTheDocument();
    expect(screen.getByText("Refunds")).toBeInTheDocument();
  });

  it("does not load credits until the credits tab opens", async () => {
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();

    const urls = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
    expect(urls.some((url) => url.includes("/llm-credit"))).toBe(false);
    expect(urls.some((url) => url.includes("/billing-plan"))).toBe(false);
  });

  it("opens features and credits after tab clicks", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Features" }));

    expect(await screen.findByText("Factories")).toBeVisible();
    expect(screen.getByRole("switch", { name: "Toggle Factories" })).toBeVisible();
    expect(screen.queryByPlaceholderText("Search users...")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Credits" }));

    expect(await screen.findByText("Hosted credit")).toBeVisible();
    expect(await screen.findByTestId("admin-org-credit-topup-target")).toBeVisible();
    expect(screen.queryByText("Factories")).not.toBeInTheDocument();
  });

  it("keeps unsaved credit edits after switching tabs", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("tab", { name: "Credits" }));

    const amount = await screen.findByTestId("admin-org-credit-topup-target");
    await user.clear(amount);
    await user.type(amount, "12.50");

    await user.click(screen.getByRole("tab", { name: "Users" }));
    expect(await screen.findByText("Ada Lovelace")).toBeVisible();
    expect(screen.getByRole("tabpanel", { name: "Credits" })).toHaveAttribute("data-state", "inactive");
    expect(screen.getByTestId("admin-org-credit-topup-target")).toHaveValue("12.50");

    await user.click(screen.getByRole("tab", { name: "Credits" }));
    expect(screen.getByRole("tabpanel", { name: "Credits" })).toHaveAttribute("data-state", "active");
    expect(await screen.findByTestId("admin-org-credit-topup-target")).toHaveValue("12.50");
  });

  it("retries overview load after an error", async () => {
    const user = userEvent.setup();
    const overview = {
      id: ORG_ID,
      name: "Acme",
      slug: "acme",
      description: "Builds widgets",
      canvas_count: 2,
      task_count: 4,
      done_task_count: 6,
      member_count: 3,
      created_at: "2024-01-15T12:00:00Z",
      updated_at: "2024-02-20T12:00:00Z",
    };
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === `/admin/api/organizations/${ORG_ID}`) {
        if (vi.mocked(fetch).mock.calls.filter(([call]) => String(call) === url).length === 1) {
          return new Response("error", { status: 500 });
        }
        return jsonResponse(overview);
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();

    expect(await screen.findByText("Could not load this organization.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Acme")).toBeInTheDocument();
  });
});

function organizationOverview() {
  return {
    id: ORG_ID,
    name: "Acme",
    slug: "acme",
    description: "Builds widgets",
    canvas_count: 2,
    task_count: 4,
    done_task_count: 6,
    member_count: 3,
    created_at: "2024-01-15T12:00:00Z",
    updated_at: "2024-02-20T12:00:00Z",
  };
}

function adminVelocityReport() {
  return {
    factories: [{ id: "factory-1", name: "Refunds" }],
    factoryId: "factory-1",
    periodDays: 30,
    hasPeopleCohort: false,
    hasPreviousWindow: true,
    totals: {
      superplaneMerged: 2,
      peopleMerged: 0,
      waste: 1,
      costCents: 500,
      modelCostCents: 400,
      computeCostCents: 100,
      tokens: 1000,
      wasteCostCents: 50,
      tasksClosed: 3,
      tasksWaste: 1,
    },
    previousTotals: {
      superplaneMerged: 1,
      peopleMerged: 0,
      waste: 0,
      costCents: 100,
      tasksClosed: 1,
      tasksWaste: 0,
    },
    points: [
      {
        day: "Mon 1",
        superplaneMerged: 2,
        peopleMerged: 0,
        waste: 1,
        costCents: 500,
        modelCostCents: 400,
        computeCostCents: 100,
        tokens: 1000,
        wasteCostCents: 50,
        medianTaskModelCostCents: 200,
        medianTaskComputeCostCents: 50,
        intake: [{ key: "manual", merged: 2 }],
      },
    ],
    intakeSources: [{ key: "manual", label: "Manually created", merged: 2 }],
    automations: [
      {
        id: "canvas-9",
        name: "Nightly close",
        runs: 4,
        failed: 1,
        costCents: 300,
        averageDurationHours: 0.5,
      },
    ],
  };
}
