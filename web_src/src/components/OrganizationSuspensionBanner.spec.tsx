import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter, useNavigate } from "react-router";
import { OrganizationSuspensionBanner } from "./OrganizationSuspensionBanner";

const organization = { suspended: false };
vi.mock("@/api-client/sdk.gen", () => ({
  organizationsDescribeOrganization: async () => ({ data: { organization: { ...organization } } }),
}));

function MemberPage() {
  const navigate = useNavigate();
  return (
    <>
      <OrganizationSuspensionBanner organizationId="org-1" />
      <button onClick={() => navigate("/settings")}>Open settings</button>
      <h1>Existing work</h1>
    </>
  );
}

describe("OrganizationSuspensionBanner", () => {
  it("refreshes on denied writes and navigation, then clears after reactivation", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <MemberPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await waitFor(() => expect(client.isFetching()).toBe(0));
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    organization.suspended = true;
    window.dispatchEvent(new Event("organization-suspended"));
    expect(await screen.findByRole("status")).toHaveTextContent("Configuration changes and new runs are blocked.");
    expect(screen.getByRole("heading", { name: "Existing work" })).toBeInTheDocument();
    organization.suspended = false;
    fireEvent.click(screen.getByRole("button", { name: "Open settings" }));
    await waitFor(() => expect(screen.queryByRole("status")).not.toBeInTheDocument());
    expect(screen.getByRole("heading", { name: "Existing work" })).toBeInTheDocument();
  });
});
