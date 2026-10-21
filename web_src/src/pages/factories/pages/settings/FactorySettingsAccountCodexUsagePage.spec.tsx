import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { meListCodexUsage, type MeCodexUsageConnection } from "@/api-client";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { CodexUsageConnections, FactorySettingsAccountCodexUsagePage } from "./FactorySettingsAccountCodexUsagePage";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";

vi.mock("@/api-client", () => ({ meListCodexUsage: vi.fn() }));

describe("Codex usage", () => {
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
    load.mockResolvedValue({ data: { connections: [] }, error: undefined });
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
    await waitFor(() => expect(screen.getByText("No ChatGPT subscription connected")).toBeInTheDocument());
    expect(load).toHaveBeenCalledWith(
      expect.objectContaining({ headers: expect.objectContaining({ "x-organization-id": "org-1" }) }),
    );
    view.unmount();
    queryClient.clear();
  });
});
