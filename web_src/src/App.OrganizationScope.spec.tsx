import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsOrganization } from "@/api-client";
import { AccountContext, type AccountContextType } from "@/contexts/accountContextState";
import { organizationKeys } from "@/hooks/useOrganizationData";

import { OrganizationScope } from "./App";

const { organizationsDescribeOrganization } = vi.hoisted(() => ({
  organizationsDescribeOrganization: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsDescribeOrganization,
}));

const accountContextValue: AccountContextType = {
  account: { id: "account-1" } as unknown as AccountContextType["account"],
  loading: false,
  setupRequired: false,
  refreshAccount: async () => undefined,
};

const publicLinePath = "/demo/workspaces/newwo/lines/fb0e0e21-8d19-4b3e-ac3f-cfe1cc54f4d7";

let childRenderCount = 0;

function OrgHome() {
  childRenderCount += 1;
  const { organizationId } = useParams<{ organizationId: string }>();
  return <div>Org home: {organizationId}</div>;
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function hangOrganizationRequest() {
  organizationsDescribeOrganization.mockReturnValue(new Promise(() => undefined));
}

function deferOrganizationDenial() {
  let rejectRequest: (error: Error) => void = () => undefined;
  const request = new Promise<never>((_resolve, reject) => {
    rejectRequest = reject;
  });
  organizationsDescribeOrganization.mockReturnValue(request);
  return async () => {
    await act(async () => {
      rejectRequest(new Error("Not Found"));
    });
  };
}

function renderScope(
  initialEntry: string,
  organization?: OrganizationsOrganization,
  queryClient = createQueryClient(),
  account = accountContextValue,
) {
  const segment = initialEntry.split("/")[1];
  if (organization !== undefined) {
    queryClient.setQueryData(organizationKeys.details(segment), organization);
    if (organization.metadata?.slug) {
      queryClient.setQueryData(organizationKeys.details(organization.metadata.slug), organization);
    }
  }

  return render(
    <QueryClientProvider client={queryClient}>
      <AccountContext.Provider value={account}>
        <MemoryRouter initialEntries={[initialEntry]}>
          <Routes>
            <Route path="/:organizationId" element={<OrganizationScope />}>
              <Route index element={<OrgHome />} />
              <Route path="*" element={<OrgHome />} />
            </Route>
          </Routes>
        </MemoryRouter>
      </AccountContext.Provider>
    </QueryClientProvider>,
  );
}

describe("OrganizationScope", () => {
  beforeEach(() => {
    childRenderCount = 0;
    organizationsDescribeOrganization.mockReset();
    hangOrganizationRequest();
  });

  it("waits for the organization before rendering the child route", () => {
    renderScope("/org-uid-123");

    expect(screen.queryByText("Org home: org-uid-123")).not.toBeInTheDocument();
    expect(screen.getByText("Loading organization...")).toBeInTheDocument();
    expect(childRenderCount).toBe(0);
  });

  it("moves a fresh missing organization from loading to not found without the child route", async () => {
    const denyOrganization = deferOrganizationDenial();
    renderScope("/missing-org/apps/canvas-1");

    expect(screen.getByText("Loading organization...")).toBeInTheDocument();
    expect(screen.queryByText("Org home: missing-org")).not.toBeInTheDocument();
    expect(childRenderCount).toBe(0);

    await denyOrganization();

    expect(await screen.findByText("Organization not found")).toBeInTheDocument();
    expect(screen.queryByText("Loading organization...")).not.toBeInTheDocument();
    expect(screen.queryByText("Org home: missing-org")).not.toBeInTheDocument();
    expect(childRenderCount).toBe(0);
  });

  it("renders a signed-in public line when organization access is denied", async () => {
    const denyOrganization = deferOrganizationDenial();
    renderScope(publicLinePath);

    expect(screen.getByText("Loading organization...")).toBeInTheDocument();
    expect(screen.queryByText("Org home: demo")).not.toBeInTheDocument();

    await denyOrganization();

    expect(await screen.findByText("Org home: demo")).toBeInTheDocument();
    expect(screen.queryByText("Organization not found")).not.toBeInTheDocument();
  });

  it("renders the public guest line without waiting for an organization", () => {
    renderScope(publicLinePath, undefined, createQueryClient(), {
      ...accountContextValue,
      account: null,
    });

    expect(screen.getByText("Org home: demo")).toBeInTheDocument();
    expect(organizationsDescribeOrganization).not.toHaveBeenCalled();
  });

  it("redirects a UID URL to the org slug once the organization resolves", () => {
    renderScope("/org-uid-123", {
      metadata: { id: "org-uid-123", slug: "acme" },
    } as OrganizationsOrganization);

    expect(screen.getByText("Org home: acme")).toBeInTheDocument();
    expect(screen.queryByText("Org home: org-uid-123")).not.toBeInTheDocument();
  });

  it("does not redirect when the URL already uses the org slug", () => {
    renderScope("/acme", {
      metadata: { id: "org-uid-123", slug: "acme" },
    } as OrganizationsOrganization);

    expect(screen.getByText("Org home: acme")).toBeInTheDocument();
  });

  it("does not redirect when the organization has no slug yet", () => {
    renderScope("/org-uid-123", {
      metadata: { id: "org-uid-123", slug: "" },
    } as OrganizationsOrganization);

    expect(screen.getByText("Org home: org-uid-123")).toBeInTheDocument();
  });

  it("redirects reserved segments home instead of treating them as an organization", () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <AccountContext.Provider value={accountContextValue}>
          <MemoryRouter initialEntries={["/login"]}>
            <Routes>
              <Route path="/" element={<div>Landing</div>} />
              <Route path="/:organizationId" element={<OrganizationScope />}>
                <Route index element={<div>Org home</div>} />
              </Route>
            </Routes>
          </MemoryRouter>
        </AccountContext.Provider>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Landing")).toBeInTheDocument();
  });
});
