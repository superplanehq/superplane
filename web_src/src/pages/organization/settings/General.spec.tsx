import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
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
let deleteResponse: Response;
let queryClient: QueryClient;

beforeEach(() => {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async () => deleteResponse.clone(),
  });
});

afterEach(() => {
  queryClient.clear();
  client.setConfig(originalConfig);
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

it("explains a live subscription and links to Billing", async () => {
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

  await deleteOrganization();

  expect(await screen.findByText(SUBSCRIPTION_LIVE_MESSAGE, { exact: false })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Open Billing" })).toHaveAttribute("href", "/org-1/organization/billing");
});

it("shows the fallback text when the failure has no message", async () => {
  deleteResponse = new Response("", { status: 500 });

  await deleteOrganization();

  expect(await screen.findByText("Failed to delete organization. Please try again.")).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Open Billing" })).not.toBeInTheDocument();
});
