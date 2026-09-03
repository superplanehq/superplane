import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FACTORIES_ORGANIZATION_ID } from "../__fixtures__/factoryPageResponses";
import { SidebarUserMenu } from "./SidebarUserMenu";

vi.mock("@/posthog", () => ({
  posthog: { reset: vi.fn() },
}));

function renderMenu() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ThemeProvider>
          <SidebarUserMenu
            organizationId={FACTORIES_ORGANIZATION_ID}
            factoryKey="RFSDR"
            userName="Ada Lovelace"
            organizationName="SuperPlane"
          />
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("SidebarUserMenu", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", async (input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      if (url.includes("/organizations")) {
        return new Response(
          JSON.stringify([
            { id: FACTORIES_ORGANIZATION_ID, slug: "superplane", name: "SuperPlane" },
            { id: "org-acme", slug: "acme", name: "Acme" },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response("not found", { status: 404 });
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("opens the user menu from the whole profile row", async () => {
    const user = userEvent.setup();
    renderMenu();

    const trigger = screen.getByRole("button", { name: /Ada Lovelace/ });
    expect(trigger).toHaveAccessibleName(/Ada Lovelace.*SuperPlane/s);
    expect(trigger).toHaveAttribute("data-testid", "factories-sidebar-user-menu-trigger");
    expect(within(trigger).queryByRole("button")).not.toBeInTheDocument();

    await user.click(trigger);

    expect(trigger).toHaveAccessibleName(/Ada Lovelace.*SuperPlane/s);
    const organizationName = screen.getByTestId("factories-sidebar-organization-name");
    expect(organizationName.tagName).toBe("P");
    expect(organizationName).toHaveTextContent("SuperPlane");
    expect(screen.queryByRole("menuitem", { name: "SuperPlane" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Organization settings")).toBeInTheDocument();
    expect(screen.getByLabelText("Switch organization")).toBeInTheDocument();
    expect(screen.getByTestId("factories-sidebar-back-to-apps")).toHaveTextContent("Back to Apps");
    expect(screen.getByRole("menuitem", { name: "Profile" })).toBeInTheDocument();
    expect(screen.getByTestId("factories-sidebar-appearance")).toHaveTextContent("Appearance");
    expect(screen.getByRole("menuitem", { name: "Sign out" })).toBeInTheDocument();
  });

  it("lists organizations and Create new organization last", async () => {
    const user = userEvent.setup();
    renderMenu();

    await user.click(screen.getByRole("button", { name: /Ada Lovelace/ }));
    await user.click(screen.getByLabelText("Switch organization"));

    const menu = await screen.findByTestId("factories-sidebar-organization-switch-menu");
    expect(within(menu).getByText("SuperPlane")).toBeInTheDocument();
    expect(within(menu).getByText("Acme")).toBeInTheDocument();
    const items = within(menu).getAllByRole("menuitem");
    expect(items[items.length - 1]).toHaveTextContent("Create new organization");
  });
});
