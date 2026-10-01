import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { describe, expect, it } from "bun:test";

import type { OrganizationsOrganization } from "@/api-client";
import { AccountContext, type AccountContextType } from "@/contexts/accountContextState";
import { organizationKeys } from "@/hooks/useOrganizationData";

import { OrganizationScope } from "./App";

const accountContextValue: AccountContextType = {
  account: { id: "account-1" } as unknown as AccountContextType["account"],
  loading: false,
  setupRequired: false,
  refreshAccount: async () => undefined,
};

function OrgHome() {
  const { organizationId } = useParams<{ organizationId: string }>();
  return <div>Org home: {organizationId}</div>;
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, retryOnMount: false },
      mutations: { retry: false },
    },
  });
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
  it("waits for the organization before rendering the child route", () => {
    renderScope("/org-uid-123");

    expect(screen.queryByText("Org home: org-uid-123")).not.toBeInTheDocument();
    expect(screen.getByText("Loading organization...")).toBeInTheDocument();
  });

  it("shows organization not found without rendering the child app route", async () => {
    const queryClient = createQueryClient();
    await queryClient
      .fetchQuery({
        queryKey: organizationKeys.details("missing-org"),
        queryFn: () => Promise.reject(new Error("Not Found")),
      })
      .catch(() => undefined);

    renderScope("/missing-org/apps/canvas-1", undefined, queryClient);

    expect(screen.getByText("Organization not found")).toBeInTheDocument();
    expect(screen.queryByText("Org home: missing-org")).not.toBeInTheDocument();
  });

  it("renders the public guest line without waiting for an organization", () => {
    renderScope("/demo/workspaces/newwo/lines/fb0e0e21-8d19-4b3e-ac3f-cfe1cc54f4d7", undefined, createQueryClient(), {
      ...accountContextValue,
      account: null,
    });

    expect(screen.getByText("Org home: demo")).toBeInTheDocument();
  });

  it("redirects a UID URL to the org slug once the organization resolves", () => {
    renderScope("/org-uid-123", {
      metadata: { id: "org-uid-123", slug: "acme" },
    } as OrganizationsOrganization);

    // The redirect target (`/acme`) matches the same route pattern, so the
    // outlet renders again — this time with the slug in the URL param.
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
