import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OrganizationSettingsOverviewPage } from "./OrganizationSettingsOverviewPage";

const mutateAsync = vi.fn();
let canUpdateOrg = true;

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({
    data: {
      metadata: { id: "org-1", name: "Acme", slug: "acme" },
    },
  }),
  useUpdateOrganization: () => ({
    mutateAsync,
    isPending: false,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdateOrg, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage(initialPath = "/org-1/organization/general") {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/:organizationId/organization/general" element={<OrganizationSettingsOverviewPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OrganizationSettingsOverviewPage", () => {
  beforeEach(() => {
    canUpdateOrg = true;
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({});
  });

  it("renders the current slug and keeps the name read-only", () => {
    renderPage();

    expect(screen.getByTestId("organization-settings-overview-name")).toHaveTextContent("Acme");
    expect(screen.getByTestId("organization-settings-overview-slug-input")).toHaveValue("acme");
  });

  it("disables the slug input and Save button without update permission", () => {
    canUpdateOrg = false;
    renderPage();

    expect(screen.getByTestId("organization-settings-overview-slug-input")).toBeDisabled();
    expect(screen.getByTestId("organization-settings-overview-save")).toBeDisabled();
  });

  it("shows a validation error for an invalid slug and does not call the API", async () => {
    const user = userEvent.setup();
    renderPage();

    const input = screen.getByTestId("organization-settings-overview-slug-input");
    await user.clear(input);
    await user.type(input, "Not A Slug");
    await user.click(screen.getByTestId("organization-settings-overview-save"));

    expect(await screen.findByText("Use lowercase letters, numbers, and dashes only.")).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("saves a valid slug change and surfaces a backend error inline", async () => {
    const user = userEvent.setup();
    renderPage();

    const input = screen.getByTestId("organization-settings-overview-slug-input");
    await user.clear(input);
    await user.type(input, "new-slug");
    await user.click(screen.getByTestId("organization-settings-overview-save"));

    expect(mutateAsync).toHaveBeenCalledWith({ name: "Acme", slug: "new-slug" });

    mutateAsync.mockRejectedValueOnce({ error: { message: "Slug is already in use" } });
    await user.click(screen.getByTestId("organization-settings-overview-save"));

    expect(await screen.findByText("Slug is already in use")).toBeInTheDocument();
  });
});
