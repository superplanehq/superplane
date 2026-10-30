import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";

import {
  meDescribeCodexConnection,
  meStartCodexConnection,
  meListCodexUsage,
  type FactoriesFactoryCodexConnection,
  type MeCodexUsageConnection,
} from "@/api-client";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { CodexUsageConnections, FactorySettingsAccountCodexUsagePage } from "./FactorySettingsAccountCodexUsagePage";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";

vi.mock("@/api-client", () => ({
  meListCodexUsage: vi.fn(),
  meDescribeCodexConnection: vi.fn(),
  meStartCodexConnection: vi.fn(),
}));
beforeEach(() => {
  vi.mocked(meDescribeCodexConnection).mockResolvedValue({
    data: { connection: { status: "disconnected", eligible: true } },
  } as never);
  vi.mocked(meStartCodexConnection)
    .mockReset()
    .mockResolvedValue({ data: {} } as never);
});

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

  it("shows unavailable usage without a snapshot and hides usage without a connection", () => {
    const view = render(
      <CodexUsageConnections connections={[{ id: "connection-1", accountLabel: "ada@example.com" }]} />,
    );
    expect(screen.getByText(/Usage unavailable/)).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    view.rerender(<CodexUsageConnections connections={[]} />);
    expect(view.container).toBeEmptyDOMElement();
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

  it("keeps disconnected settings open and starts personal sign-in", async () => {
    vi.mocked(meListCodexUsage).mockResolvedValue({ data: { connections: [] }, error: undefined });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = renderUsageRoute(queryClient);
    try {
      expect(await screen.findByRole("heading", { name: "Codex" })).toBeInTheDocument();
      expect(screen.queryByRole("heading", { name: "Account settings" })).not.toBeInTheDocument();
      const connect = await screen.findByRole("button", { name: "Connect ChatGPT" });
      await waitFor(() => expect(connect).toBeEnabled());
      connect.click();
      await waitFor(() => expect(meStartCodexConnection).toHaveBeenCalledTimes(1));
    } finally {
      view.unmount();
      queryClient.clear();
    }
  });

  it("keeps the connected view busy until the refresh task finishes", async () => {
    let connection: FactoriesFactoryCodexConnection = {
      status: "ready",
      eligible: true,
      accountLabel: "ada@example.com",
      availableModels: ["gpt-5.6-sol"],
      reservationHeld: false,
    };
    vi.mocked(meDescribeCodexConnection).mockImplementation(async () => ({ data: { connection } }) as never);
    vi.mocked(meStartCodexConnection).mockImplementation(async () => {
      connection = { ...connection, reservationHeld: true };
      return { data: { connection } } as never;
    });
    vi.mocked(meListCodexUsage).mockResolvedValue({ data: { connections: [] } } as never);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = renderUsageRoute(queryClient);
    try {
      const refresh = await screen.findByRole("button", { name: "Refresh usage and models" });
      await waitFor(() => expect(refresh).toBeEnabled());
      refresh.click();
      await waitFor(() => expect(meStartCodexConnection).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(queryClient.isMutating()).toBe(0));

      expect(screen.getByRole("status")).toHaveTextContent("Connected");
      expect(screen.getByText("ada@example.com")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Refresh usage and models" })).toBeDisabled();
      expect(screen.getByRole("button", { name: "Refresh usage and models" })).toHaveAttribute("aria-busy", "true");
      expect(screen.getByRole("button", { name: "Disconnect ChatGPT" })).toBeDisabled();
      expect(screen.queryByText(/Wait for connection setup to finish/)).not.toBeInTheDocument();

      connection = { ...connection, reservationHeld: false };
      await act(() => queryClient.invalidateQueries({ queryKey: ["factory-codex-connection", "org-1"] }));

      await waitFor(() => expect(screen.getByRole("button", { name: "Refresh usage and models" })).toBeEnabled());
      expect(screen.getByRole("button", { name: "Refresh usage and models" })).toHaveAttribute("aria-busy", "false");
      expect(screen.getByRole("button", { name: "Disconnect ChatGPT" })).toBeEnabled();
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
