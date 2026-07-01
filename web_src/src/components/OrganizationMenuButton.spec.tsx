import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { TooltipProvider } from "@/ui/tooltip";

function renderOrganizationMenuButton(ui: React.ReactElement) {
  return render(
    <TooltipProvider>
      <MemoryRouter>{ui}</MemoryRouter>
    </TooltipProvider>,
  );
}

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: {
      id: "user-1",
      name: "Ada Lovelace",
      email: "ada@example.com",
      installation_admin: false,
    },
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: null }),
  useOrganizationUsage: () => ({ data: null, error: null }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/lib/env", () => ({
  isUsagePageForced: () => false,
}));

vi.mock("@/posthog", () => ({
  posthog: { reset: vi.fn() },
}));

describe("OrganizationMenuButton", () => {
  it("links the logo to organization selection when no organization is active", () => {
    renderOrganizationMenuButton(<OrganizationMenuButton />);

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/");
  });

  it("links the logo to the active organization when one is active", () => {
    renderOrganizationMenuButton(<OrganizationMenuButton organizationId="org-123" />);

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/org-123");
  });
});
