import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";

import { meListCodexUsage, type MeCodexUsageConnection } from "@/api-client";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { CodexUsageConnections, FactorySettingsAccountCodexUsagePage } from "./FactorySettingsAccountCodexUsagePage";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";

vi.mock("@/api-client", () => ({ meListCodexUsage: vi.fn() }));

function renderUsageRoute(queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-1/workspaces/refund/settings/account/codex-usage"]}>
        <FactorySettingsLayoutContext.Provider
          value={{ organizationId: "org-1", factoryId: "factory-1", factory: REFUND_FACTORY }}
        >
          <Routes>
            <Route
              path="/:organizationId/workspaces/:factoryKey/settings/account/codex-usage"
              element={<FactorySettingsAccountCodexUsagePage />}
            />
            <Route
              path="/:organizationId/workspaces/:factoryKey/settings/account/profile"
              element={<h1>Account settings</h1>}
            />
          </Routes>
        </FactorySettingsLayoutContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Codex usage", () => {
  it("hides a saved percentage when its reset time passes while the page stays open", () => {
    vi.useFakeTimers();
    const now = Date.now();
    const view = render(
      <CodexUsageConnections
        connections={[
          {
            id: "connection-1",
            observedAt: new Date(now).toISOString(),
            limits: [{ limitId: "codex", primary: { usedPercent: 25, resetsAt: new Date(now + 1000).toISOString() } }],
          },
        ]}
      />,
    );
    try {
      expect(screen.getByText("75% remaining")).toBeInTheDocument();
      act(() => vi.advanceTimersByTime(1000));
      expect(screen.getByText("Update needed")).toBeInTheDocument();
      expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
      expect(screen.queryByText("75% remaining")).not.toBeInTheDocument();
    } finally {
      view.unmount();
      vi.useRealTimers();
    }
  });

  it("shows separate quota windows and hides percentages after the reset time", () => {
    const now = Date.now();
    const connections: MeCodexUsageConnection[] = [
      {
        id: "connection-1",
        accountLabel: "ada@example.com",
        observedAt: new Date(now).toISOString(),
        limits: [
          {
            limitId: "codex",
            primary: { usedPercent: 25, windowDurationMins: "300", resetsAt: new Date(now + 3600000).toISOString() },
            secondary: {
              usedPercent: 90,
              windowDurationMins: "10080",
              resetsAt: new Date(now - 3600000).toISOString(),
            },
          },
        ],
      },
    ];
    render(<CodexUsageConnections connections={connections} />);
    expect(screen.getByText("75% remaining")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "5-hour window remaining" })).toHaveAttribute("aria-valuenow", "75");
    expect(screen.getByText("7-day window")).toBeInTheDocument();
    expect(screen.getByText("Update needed")).toBeInTheDocument();
    expect(screen.queryByText("10% remaining")).not.toBeInTheDocument();
    expect(screen.getByText(/Last updated/)).toBeInTheDocument();
  });

  it("does not invent a percentage when no quota snapshot exists", () => {
    const view = render(
      <CodexUsageConnections connections={[{ id: "connection-1", accountLabel: "ada@example.com" }]} />,
    );
    expect(screen.getByText(/Usage unavailable/)).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    view.rerender(<CodexUsageConnections connections={[]} />);
    expect(screen.getByText("No ChatGPT subscription connected")).toBeInTheDocument();
  });

  it("loads saved usage for the organization and lets the user retry a failed request", async () => {
    const load = vi.mocked(meListCodexUsage);
    load.mockRejectedValueOnce(new Error("offline"));
    load.mockResolvedValue({
      data: { connections: [{ id: "connection-1", accountLabel: "ada@example.com" }] },
      error: undefined,
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId: "org-1", factoryId: "factory-1", factory: REFUND_FACTORY }}
          >
            <FactorySettingsAccountCodexUsagePage />
          </FactorySettingsLayoutContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByText("Could not load Codex usage.")).toBeInTheDocument());
    screen.getByRole("button", { name: "Try again" }).click();
    await waitFor(() => expect(screen.getByText("ada@example.com")).toBeInTheDocument());
    expect(load).toHaveBeenCalledWith(
      expect.objectContaining({ headers: expect.objectContaining({ "x-organization-id": "org-1" }) }),
    );
    view.unmount();
    queryClient.clear();
  });

  it("opens Account settings when a direct usage URL has no connection", async () => {
    vi.mocked(meListCodexUsage).mockResolvedValue({ data: { connections: [] }, error: undefined });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = renderUsageRoute(queryClient);
    try {
      expect(await screen.findByRole("heading", { name: "Account settings" })).toBeInTheDocument();
      expect(screen.queryByText("Codex usage")).not.toBeInTheDocument();
    } finally {
      view.unmount();
      queryClient.clear();
    }
  });

  it("keeps the usage page open when a cached empty result refreshes to a connection", async () => {
    vi.mocked(meListCodexUsage).mockResolvedValue({
      data: { connections: [{ id: "connection-1", accountLabel: "ada@example.com" }] },
      error: undefined,
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(["me", "org-1", "codex-usage", undefined], []);
    const view = renderUsageRoute(queryClient);
    try {
      expect(await screen.findByText("ada@example.com")).toBeInTheDocument();
      expect(screen.queryByRole("heading", { name: "Account settings" })).not.toBeInTheDocument();
    } finally {
      view.unmount();
      queryClient.clear();
    }
  });
});
