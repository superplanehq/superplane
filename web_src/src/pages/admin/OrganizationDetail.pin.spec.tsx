import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes, useNavigate } from "react-router";

import OrganizationDetail from "./OrganizationDetail";

const ORG_ID = "org-1";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Admin" } }),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
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

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("OrganizationDetail pin control", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("pins the organization from the page header on every tab", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === `/admin/api/organizations/${ORG_ID}/pin` && init?.method === "PUT") {
        return jsonResponse({ pinned: true });
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
          pinned: false,
        });
      }
      return jsonResponse({ items: [], total: 0 });
    });
    vi.stubGlobal("fetch", fetchMock);

    renderPage();

    const pinButton = await screen.findByRole("button", { name: "Pin" });
    await user.click(screen.getByRole("tab", { name: "Users" }));
    expect(pinButton).toBeInTheDocument();

    await user.click(pinButton);

    expect(fetchMock).toHaveBeenCalledWith(`/admin/api/organizations/${ORG_ID}/pin`, {
      method: "PUT",
      credentials: "include",
    });
    expect(await screen.findByRole("button", { name: "Unpin" })).toBeInTheDocument();
  });

  it("does not use the previous organization pin state", async () => {
    const user = userEvent.setup();
    let releaseNextOrganization: (response: Response) => void = () => {};
    const nextOrganization = new Promise<Response>((resolve) => {
      releaseNextOrganization = resolve;
    });
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.endsWith("/pin")) {
          return jsonResponse({ pinned: init?.method !== "DELETE" });
        }
        if (url === "/admin/api/organizations/org-2") {
          return nextOrganization;
        }
        if (url === `/admin/api/organizations/${ORG_ID}`) {
          return jsonResponse(organizationDetail(ORG_ID, true));
        }
        return jsonResponse({ items: [], total: 0 });
      }),
    );

    renderOrganizationSwitch();

    expect(await screen.findByRole("button", { name: "Unpin" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open next organization" }));

    expect(screen.queryByRole("button", { name: "Unpin" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Pin" })).not.toBeInTheDocument();

    releaseNextOrganization(jsonResponse(organizationDetail("org-2", false)));

    expect(await screen.findByRole("button", { name: "Pin" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Unpin" })).not.toBeInTheDocument();
  });

  it("shows the pin control after the overview load is retried", async () => {
    const user = userEvent.setup();
    let allowOrganization = false;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === `/admin/api/organizations/${ORG_ID}`) {
          if (!allowOrganization) {
            return new Response("error", { status: 500 });
          }
          return jsonResponse(organizationDetail(ORG_ID, false));
        }
        return jsonResponse({ items: [], total: 0 });
      }),
    );

    renderPage();

    expect(await screen.findByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Pin" })).not.toBeInTheDocument();

    allowOrganization = true;
    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByRole("button", { name: "Pin" })).toBeInTheDocument();
  });
});

function organizationDetail(id: string, pinned: boolean) {
  return {
    id,
    name: id === ORG_ID ? "Acme" : "Beta",
    slug: "acme",
    description: "Builds widgets",
    canvas_count: 2,
    task_count: 4,
    done_task_count: 6,
    member_count: 3,
    pinned,
  };
}

function OrganizationSwitch() {
  const navigate = useNavigate();
  return (
    <>
      <button type="button" onClick={() => navigate("/admin/organizations/org-2")}>
        Open next organization
      </button>
      <OrganizationDetail />
    </>
  );
}

function renderOrganizationSwitch() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
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
  return render(<OrganizationSwitch />, { wrapper });
}
