import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FACTORIES_ORGANIZATION_ID } from "../__fixtures__/factoryPageResponses";
import { SidebarUserMenu } from "./SidebarUserMenu";

vi.mock("@/posthog", () => ({
  posthog: { reset: vi.fn() },
}));

function renderMenu() {
  return render(
    <MemoryRouter>
      <ThemeProvider>
        <SidebarUserMenu
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey="RFSDR"
          userName="Ada Lovelace"
          userEmail="ada@example.com"
          organizationName="SuperPlane"
        />
      </ThemeProvider>
    </MemoryRouter>,
  );
}

describe("SidebarUserMenu", () => {
  it("opens the user menu from the whole profile row", async () => {
    const user = userEvent.setup();
    renderMenu();

    const trigger = screen.getByRole("button", { name: /Ada Lovelace/ });
    expect(trigger).toHaveAccessibleName(/Ada Lovelace.*SuperPlane/s);
    expect(trigger).toHaveAttribute("data-testid", "factories-sidebar-user-menu-trigger");
    expect(within(trigger).queryByRole("button")).not.toBeInTheDocument();

    await user.click(trigger);

    expect(screen.getByText("ada@example.com")).toBeInTheDocument();
    expect(screen.getByTestId("factories-sidebar-back-to-apps")).toHaveTextContent("Back to Apps");
    expect(screen.getByRole("menuitem", { name: "Profile" })).toBeInTheDocument();
    expect(screen.getByTestId("factories-sidebar-appearance")).toHaveTextContent("Appearance");
    expect(screen.getByRole("menuitem", { name: "Sign out" })).toBeInTheDocument();
  });
});
