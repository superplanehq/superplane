import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { OrgExperimentalFeaturesTable } from "./OrgExperimentalFeaturesTable";

const ORG_ID = "org-1";

const registryResponse = {
  features: [
    { id: "factories", label: "Factories", description: "Software factories", released: false },
    { id: "new_canvas", label: "New Canvas", description: "Canvas v2", released: false },
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

function featureTogglePath(featureId: string) {
  return `/admin/api/organizations/${ORG_ID}/experimental-features/${featureId}`;
}

function stubFetch(registry = registryResponse) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === `/admin/api/organizations/${ORG_ID}/experimental-features`) {
        return jsonResponse(registry);
      }
      if (url.startsWith(`/admin/api/organizations/${ORG_ID}/experimental-features/`)) {
        return jsonResponse({ status: init?.method === "DELETE" ? "disabled" : "enabled" });
      }
      return new Response("not found", { status: 404 });
    }),
  );
}

function featureToggleCalls() {
  return vi.mocked(fetch).mock.calls.filter(([, init]) => init?.method === "POST" || init?.method === "DELETE");
}

describe("OrgExperimentalFeaturesTable", () => {
  beforeEach(() => {
    stubFetch();
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

    expect(vi.mocked(fetch)).toHaveBeenCalledWith(featureTogglePath("factories"), {
      method: "DELETE",
      credentials: "include",
    });
  });

  it("shows a master switch when visible flags exist", async () => {
    renderTable();

    expect(await screen.findByRole("switch", { name: "Toggle all experimental features" })).toBeInTheDocument();
  });

  it("shows the master switch off when flags are mixed", async () => {
    renderTable();

    expect(await screen.findByRole("switch", { name: "Toggle all experimental features" })).toHaveAttribute(
      "data-state",
      "unchecked",
    );
  });

  it("shows the master switch on when every visible flag is on", async () => {
    stubFetch({
      ...registryResponse,
      enabled: ["factories", "new_canvas"],
    });
    renderTable();

    expect(await screen.findByRole("switch", { name: "Toggle all experimental features" })).toHaveAttribute(
      "data-state",
      "checked",
    );
  });

  it("enables only visible flags that are off", async () => {
    const user = userEvent.setup();
    renderTable();

    await user.click(await screen.findByRole("switch", { name: "Toggle all experimental features" }));

    await waitFor(() => {
      expect(featureToggleCalls()).toEqual([
        [featureTogglePath("new_canvas"), { method: "POST", credentials: "include" }],
      ]);
    });
  });

  it("disables only visible flags that are on", async () => {
    stubFetch({
      ...registryResponse,
      enabled: ["factories", "new_canvas", "claude_managed_agents"],
    });
    const user = userEvent.setup();
    renderTable();

    await user.click(await screen.findByRole("switch", { name: "Toggle all experimental features" }));

    await waitFor(() => {
      expect(featureToggleCalls()).toEqual([
        [featureTogglePath("factories"), { method: "DELETE", credentials: "include" }],
        [featureTogglePath("new_canvas"), { method: "DELETE", credentials: "include" }],
      ]);
    });
  });

  it("shows an error and server state when a later bulk request fails", async () => {
    let registry = {
      ...registryResponse,
      enabled: ["factories", "new_canvas", "claude_managed_agents"],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url === `/admin/api/organizations/${ORG_ID}/experimental-features`) {
          return jsonResponse(registry);
        }
        if (url === featureTogglePath("factories") && init?.method === "DELETE") {
          registry = { ...registry, enabled: ["new_canvas", "claude_managed_agents"] };
          return jsonResponse({ status: "disabled" });
        }
        if (url === featureTogglePath("new_canvas") && init?.method === "DELETE") {
          return new Response("failed", { status: 500 });
        }
        return new Response("not found", { status: 404 });
      }),
    );
    const user = userEvent.setup();
    renderTable();

    await user.click(await screen.findByRole("switch", { name: "Toggle all experimental features" }));

    expect(await screen.findByText("Failed to disable experimental features")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("switch", { name: "Toggle Factories" })).toHaveAttribute("data-state", "unchecked");
    });
    expect(screen.getByRole("switch", { name: "Toggle New Canvas" })).toHaveAttribute("data-state", "checked");
  });
});
