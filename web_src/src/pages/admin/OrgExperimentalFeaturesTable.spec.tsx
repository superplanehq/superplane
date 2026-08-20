import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { OrgExperimentalFeaturesTable } from "./OrgExperimentalFeaturesTable";

const ORG_ID = "org-1";

const registryResponse = {
  features: [
    { id: "factories", label: "Factories", description: "Software factories", released: false },
    { id: "claude_managed_agents", label: "Claude Managed Agents", description: "Chat", released: true },
  ],
  enabled: ["factories"],
};

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function renderTable() {
  const queryClient = createQueryClient();
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  return render(<OrgExperimentalFeaturesTable orgId={ORG_ID} />, { wrapper });
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("OrgExperimentalFeaturesTable", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url === `/admin/api/organizations/${ORG_ID}/experimental-features`) {
          return jsonResponse(registryResponse);
        }
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/experimental-features/`)) {
          return jsonResponse({ status: init?.method === "DELETE" ? "disabled" : "enabled" });
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads flags from the admin route, not the member organization API", async () => {
    renderTable();

    expect(await screen.findByText("Factories")).toBeInTheDocument();
    expect(screen.queryByText("Claude Managed Agents")).not.toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "Toggle Factories" })).toHaveAttribute("data-state", "checked");

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock).toHaveBeenCalledWith(`/admin/api/organizations/${ORG_ID}/experimental-features`, {
      credentials: "include",
    });
    expect(fetchMock.mock.calls.every(([url]) => !String(url).includes("/api/v1/organizations"))).toBe(true);
    expect(fetchMock.mock.calls.every(([url]) => !String(url).includes("/account/experimental-features"))).toBe(true);
  });

  it("toggles a flag through the admin route", async () => {
    const user = userEvent.setup();
    renderTable();

    await user.click(await screen.findByRole("switch", { name: "Toggle Factories" }));

    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      `/admin/api/organizations/${ORG_ID}/experimental-features/factories`,
      {
        method: "DELETE",
        credentials: "include",
      },
    );
  });
});
