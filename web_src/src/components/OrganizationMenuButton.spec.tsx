import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";

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
    render(
      <MemoryRouter>
        <OrganizationMenuButton />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/");
  });

  it("links the logo to the active organization when one is active", () => {
    render(
      <MemoryRouter>
        <OrganizationMenuButton organizationId="org-123" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/org-123");
  });
});
