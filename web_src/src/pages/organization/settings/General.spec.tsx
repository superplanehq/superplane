import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";
import { client } from "@/api-client/client.gen";
import type { OrganizationsOrganization } from "@/api-client";
import { General } from "./General";

vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/hooks/useReportPageReady", () => ({ useReportPageReady: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

const SUBSCRIPTION_LIVE_MESSAGE =
  "Cancel the subscription in Billing before you delete the organization. You can delete it after the paid period ends.";
const organization: OrganizationsOrganization = { metadata: { id: "org-1", name: "Acme", slug: "acme" } };
const originalConfig = client.getConfig();
const originalAssign = window.location.assign;
let deleteResponse: Response;
let portalResponse: Response;
let queryClient: QueryClient;

beforeEach(() => {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  portalResponse = Response.json({ portalUrl: "https://polar.example/portal/session" });
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input) => {
      const url = input instanceof Request ? input.url : String(input);
      if (url.includes("billing-portal-session")) {
        return portalResponse.clone();
      }
      return deleteResponse.clone();
    },
  });
});

afterEach(() => {
  queryClient.clear();
  client.setConfig(originalConfig);
  window.location.assign = originalAssign;
});

async function deleteOrganization() {
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-1/organization/general"]}>
        <Routes>
          <Route path="/:organizationId/organization/general" element={<General organization={organization} />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  fireEvent.click(screen.getByRole("button", { name: "Delete Organization..." }));
  fireEvent.change(screen.getByLabelText('Type "Acme" to confirm'), { target: { value: "Acme" } });
  fireEvent.click(screen.getByRole("button", { name: "Delete Organization" }));
}

it("explains a live subscription and opens the billing portal", async () => {
  deleteResponse = Response.json(
    {
      code: 9,
      message: SUBSCRIPTION_LIVE_MESSAGE,
      details: [
        {
          "@type": "type.googleapis.com/google.rpc.ErrorInfo",
          domain: "superplane.com",
          reason: "ORGANIZATION_SUBSCRIPTION_LIVE",
        },
      ],
    },
    { status: 400 },
  );
  const assign = vi.fn();
  window.location.assign = assign;

  await deleteOrganization();

  expect(await screen.findByText(SUBSCRIPTION_LIVE_MESSAGE, { exact: false })).toBeInTheDocument();

  // The portal session is organization-scoped, so canceling the subscription
  // must not depend on a workspace.
  fireEvent.click(screen.getByRole("button", { name: "Open Billing" }));
  await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.example/portal/session"));
});

it("shows an error when the billing portal cannot be opened", async () => {
  deleteResponse = Response.json(
    {
      code: 9,
      message: SUBSCRIPTION_LIVE_MESSAGE,
      details: [
        {
          "@type": "type.googleapis.com/google.rpc.ErrorInfo",
          domain: "superplane.com",
          reason: "ORGANIZATION_SUBSCRIPTION_LIVE",
        },
      ],
    },
    { status: 400 },
  );
  portalResponse = new Response("", { status: 500 });
  const assign = vi.fn();
  window.location.assign = assign;

  await deleteOrganization();

  fireEvent.click(await screen.findByRole("button", { name: "Open Billing" }));
  expect(await screen.findByText("Could not open the billing portal. Please try again.")).toBeInTheDocument();
  expect(assign).not.toHaveBeenCalled();
});

it("shows the fallback text when the failure has no message", async () => {
  deleteResponse = new Response("", { status: 500 });

  await deleteOrganization();

  expect(await screen.findByText("Failed to delete organization. Please try again.")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Open Billing" })).not.toBeInTheDocument();
});
