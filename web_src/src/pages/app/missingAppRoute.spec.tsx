import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { AccountContext, type AccountContextType } from "@/contexts/accountContextState";
import { setupApiClientErrorInterceptor } from "@/lib/api-client-error-interceptor";

import { OrganizationScope } from "@/App";
import { AppDefaultTabGate } from "./AppDefaultTabGate";

const mounts = vi.hoisted(() => ({ appPage: 0 }));

vi.mock("./index", () => ({
  AppPage: () => {
    mounts.appPage += 1;
    return <div data-testid="app-page" />;
  },
}));

const accountContextValue: AccountContextType = {
  account: { id: "account-1" } as unknown as AccountContextType["account"],
  loading: false,
  setupRequired: false,
  refreshAccount: async () => undefined,
};

const organizationBody = {
  organization: {
    metadata: { id: "org-id", slug: "native-teams", name: "Native Teams" },
  },
};

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location" data-pathname={location.pathname} />;
}

function renderMissingApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AccountContext.Provider value={accountContextValue}>
        <MemoryRouter initialEntries={["/native-teams/apps/6f802716-f501-4853-814c-586a11fff59d"]}>
          <Routes>
            <Route path="/" element={<div>Home</div>} />
            <Route path="/:organizationId" element={<OrganizationScope />}>
              <Route index element={<div>Org home</div>} />
              <Route path="apps/:appId" element={<AppDefaultTabGate />} />
            </Route>
          </Routes>
          <LocationProbe />
        </MemoryRouter>
      </AccountContext.Provider>
    </QueryClientProvider>,
  );
}

function notFoundResponse(): Response {
  return new Response("Not Found\n", { status: 404, statusText: "Not Found" });
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("missing app route", () => {
  const rejections: unknown[] = [];
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    mounts.appPage = 0;
    rejections.length = 0;
    originalFetch = globalThis.fetch;
    setupApiClientErrorInterceptor();
    window.addEventListener("unhandledrejection", onUnhandledRejection);
  });

  afterEach(() => {
    window.removeEventListener("unhandledrejection", onUnhandledRejection);
    globalThis.fetch = originalFetch;
  });

  it("leaves a missing organization without mounting the app page", async () => {
    const requested = installFetch(() => notFoundResponse());

    renderMissingApp();

    await waitFor(() => {
      expect(screen.getByTestId("location").getAttribute("data-pathname")).toBe("/");
    });
    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.queryByTestId("app-page")).toBeNull();
    expect(mounts.appPage).toBe(0);
    expect(requested.some((url) => /runs|memory|staging|versions|console\.yaml/.test(url))).toBe(false);
    await expectNoUnhandledRejection();
  });

  it("sends a missing canvas to the organization home", async () => {
    const requested = installFetch((url) => {
      const path = requestPath(url);
      if (path.startsWith("/api/v1/organizations/native-teams") && !path.includes("/integrations")) {
        return jsonResponse(organizationBody);
      }
      if (path.includes("/account/experimental-features")) {
        return jsonResponse({ features: [] });
      }
      return notFoundResponse();
    });

    renderMissingApp();

    await waitFor(() => {
      expect(screen.getByTestId("location").getAttribute("data-pathname")).toBe("/native-teams");
    });
    expect(screen.getByText("Org home")).toBeInTheDocument();
    expect(screen.queryByTestId("app-page")).toBeNull();
    expect(mounts.appPage).toBe(0);
    expect(requested.some((url) => /runs|memory|staging|versions|console\.yaml/.test(url))).toBe(false);
    await expectNoUnhandledRejection();
  });

  function onUnhandledRejection(event: PromiseRejectionEvent) {
    event.preventDefault();
    rejections.push(event.reason);
  }

  async function expectNoUnhandledRejection() {
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(rejections).toEqual([]);
  }
});

function requestPath(url: string): string {
  return new URL(url, "http://localhost").pathname;
}

function installFetch(respond: (url: string) => Response): string[] {
  const requested: string[] = [];
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    requested.push(url);
    return respond(url);
  }) as typeof fetch;
  return requested;
}
