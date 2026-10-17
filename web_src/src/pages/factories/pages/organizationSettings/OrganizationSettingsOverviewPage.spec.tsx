import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { OrganizationSettingsOverviewPage } from "./OrganizationSettingsOverviewPage";

const mutateAsync = vi.fn();
const deleteOrganization = vi.fn();
let canUpdateOrg = true;
let canDeleteOrg = true;

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
  useDeleteOrganization: () => ({
    mutateAsync: deleteOrganization,
    isPending: false,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => {
      if (resource === "org" && action === "delete") return canDeleteOrg;
      return canUpdateOrg;
    },
    isLoading: false,
  }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

import { showErrorToast } from "@/lib/toast";

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
    canDeleteOrg = true;
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({});
    deleteOrganization.mockReset();
    deleteOrganization.mockResolvedValue({});
    vi.mocked(showErrorToast).mockReset();
  });

  it("shows a character avatar, editable name, and slug in a profile-style card", () => {
    renderPage();

    expect(screen.getByTestId("organization-settings-overview-avatar")).toHaveTextContent("A");
    expect(screen.getByTestId("organization-settings-overview-name")).toHaveValue("Acme");
    expect(screen.getByLabelText("Slug")).toHaveValue("acme");
    expect(screen.getByTestId("organization-settings-overview-slug-input")).toHaveValue("acme");
  });

  it("disables the name, slug, and Save controls without update permission", () => {
    canUpdateOrg = false;
    renderPage();

    expect(screen.getByTestId("organization-settings-overview-name")).toBeDisabled();
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

  it("saves a name change and updates the avatar initials", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.clear(screen.getByTestId("organization-settings-overview-name"));
    await user.type(screen.getByTestId("organization-settings-overview-name"), "Acme Labs");
    await user.click(screen.getByTestId("organization-settings-overview-save"));

    expect(mutateAsync).toHaveBeenCalledWith({ name: "Acme Labs" });
    expect(screen.getByTestId("organization-settings-overview-avatar")).toHaveTextContent("AL");
  });

  it("shows a Danger zone with a Delete organization button", () => {
    renderPage();

    expect(screen.getByTestId("organization-settings-danger-zone")).toHaveTextContent("Danger zone");
    expect(screen.getByTestId("organization-settings-delete-button")).toHaveTextContent("Delete organization");
    expect(
      screen.getByText(
        "You lose access now. SuperPlane keeps this organization for at least 30 days, then removes it.",
      ),
    ).toBeInTheDocument();
  });

  it("disables Delete organization when the user cannot delete the organization", () => {
    canDeleteOrg = false;
    renderPage();

    expect(screen.getByTestId("organization-settings-delete-button")).toBeDisabled();
    expect(screen.getByTestId("organization-settings-overview-name")).toBeEnabled();
  });

  it("keeps the dialog delete action disabled until the typed name matches", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("organization-settings-delete-button"));
    const confirm = screen.getByTestId("organization-delete-confirm-button");
    expect(confirm).toBeDisabled();
    expect(screen.getByText("Removal can take longer when workspaces or integrations remain.")).toBeInTheDocument();
    expect(screen.getByText("If this organization has a Business plan, SuperPlane cancels it.")).toBeInTheDocument();

    const confirmation = screen.getByLabelText('Type "Acme" to confirm');
    await user.type(confirmation, "Acm");
    expect(confirm).toBeDisabled();

    await user.type(confirmation, "e");
    expect(confirm).toBeEnabled();
  });

  it("deletes the organization once when the typed name matches", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("organization-settings-delete-button"));
    await user.type(screen.getByLabelText('Type "Acme" to confirm'), "  Acme  ");
    await user.click(screen.getByTestId("organization-delete-confirm-button"));

    expect(deleteOrganization).toHaveBeenCalledTimes(1);
  });

  it("shows an error toast and keeps the dialog open when delete fails", async () => {
    const user = userEvent.setup();
    deleteOrganization.mockRejectedValueOnce({ status: 500 });
    renderPage();

    await user.click(screen.getByTestId("organization-settings-delete-button"));
    await user.type(screen.getByLabelText('Type "Acme" to confirm'), "Acme");
    await user.click(screen.getByTestId("organization-delete-confirm-button"));

    expect(showErrorToast).toHaveBeenCalledWith("Failed to delete organization.");
    expect(screen.getByTestId("organization-delete-confirm-button")).toBeInTheDocument();
  });

  it("clears the confirmation when the dialog closes", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("organization-settings-delete-button"));
    await user.type(screen.getByLabelText('Type "Acme" to confirm'), "Ac");
    await user.click(screen.getByRole("button", { name: "Keep organization" }));
    await user.click(screen.getByTestId("organization-settings-delete-button"));

    expect(screen.getByLabelText('Type "Acme" to confirm')).toHaveValue("");
  });
});
