import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { ThemeProvider } from "@/contexts/ThemeProvider";

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

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [], refetch: vi.fn() }),
}));

const experimentalFeatureMocks = vi.hoisted(() => ({
  has: vi.fn((_featureId: string) => false),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: experimentalFeatureMocks.has,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
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
  beforeEach(() => {
    experimentalFeatureMocks.has.mockReset();
    experimentalFeatureMocks.has.mockReturnValue(false);
  });

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

  it("links the logo to workspaces when factories are on", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === "factories");
    render(
      <MemoryRouter>
        <OrganizationMenuButton organizationId="org-123" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/org-123/workspaces");
  });

  it("opens the shared feedback screen from Report issue and Send feedback", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <ThemeProvider>
          <OrganizationMenuButton organizationId="org-123" />
        </ThemeProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Open organization menu" }));
    await user.click(screen.getByRole("button", { name: "Report issue" }));

    expect(screen.getByRole("heading", { name: "Send feedback" })).toBeInTheDocument();
    expect(screen.getByTestId("feedback-category-bug")).toHaveAttribute("aria-pressed", "true");

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.click(screen.getByRole("button", { name: "Open organization menu" }));
    await user.click(screen.getByRole("button", { name: "Send feedback" }));

    expect(screen.getByRole("heading", { name: "Send feedback" })).toBeInTheDocument();
    expect(screen.getByTestId("feedback-category-other")).toHaveAttribute("aria-pressed", "true");
  });
});
