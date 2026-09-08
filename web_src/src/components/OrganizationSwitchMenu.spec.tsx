import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DropdownMenu, DropdownMenuContent } from "@/ui/dropdownMenu";

import { OrganizationSwitchMenu } from "./OrganizationSwitchMenu";

const navigateSpy = vi.fn();

vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof ReactRouterDom>("react-router");
  return {
    ...actual,
    useNavigate: () => navigateSpy,
  };
});

const organizationsState = vi.hoisted(() => ({
  data: [
    { id: "org-1", name: "Acme", slug: "acme" },
    { id: "org-2", name: "Other Co", slug: "other-co" },
    { id: "org-pending", name: "Pending Org", slug: "pending-org", initialOnboardingPending: true },
  ],
  isLoading: false,
  isError: false,
}));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => organizationsState,
}));

function renderMenu(navigateToCurrentOrganization = false) {
  return render(
    <MemoryRouter>
      <DropdownMenu open>
        <DropdownMenuContent>
          <OrganizationSwitchMenu
            currentOrganizationRouteId="acme"
            navigateToCurrentOrganization={navigateToCurrentOrganization}
            testIdPrefix="menu"
          />
        </DropdownMenuContent>
      </DropdownMenu>
    </MemoryRouter>,
  );
}

describe("OrganizationSwitchMenu", () => {
  beforeEach(() => {
    navigateSpy.mockClear();
  });

  it("does not navigate when the selected organization is already current", async () => {
    const user = userEvent.setup();
    renderMenu();

    await user.click(screen.getByTestId("menu-organization-option-org-1"));

    expect(navigateSpy).not.toHaveBeenCalled();
  });

  it("navigates to the current organization when onboarding asks to leave setup", async () => {
    const user = userEvent.setup();
    renderMenu(true);

    await user.click(screen.getByTestId("menu-organization-option-org-1"));

    expect(navigateSpy).toHaveBeenCalledWith("/acme");
  });

  it("hides organizations that did not finish first-run setup", () => {
    renderMenu();

    expect(screen.getByTestId("menu-organization-option-org-1")).toBeInTheDocument();
    expect(screen.getByTestId("menu-organization-option-org-2")).toBeInTheDocument();
    expect(screen.queryByTestId("menu-organization-option-org-pending")).not.toBeInTheDocument();
    expect(screen.getByTestId("menu-organization-create")).toBeInTheDocument();
  });

  it("navigates to another organization", async () => {
    const user = userEvent.setup();
    renderMenu();

    await user.click(screen.getByTestId("menu-organization-option-org-2"));

    expect(navigateSpy).toHaveBeenCalledWith("/other-co");
  });
});
