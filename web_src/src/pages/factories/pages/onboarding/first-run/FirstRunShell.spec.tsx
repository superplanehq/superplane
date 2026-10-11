import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunShell } from "./FirstRunShell";

function renderShell(ui: ReactNode) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FirstRunShell", () => {
  it("keeps owner setup on the app shell and does not load the art script", () => {
    const scriptsBefore = document.querySelectorAll("script[data-onboarding-src]").length;
    renderShell(
      <FirstRunShell testId="owner-setup" chrome={{ stepIndex: 0, stepCount: 2 }}>
        <p>Create the owner account</p>
      </FirstRunShell>,
    );

    expect(screen.getByTestId("owner-setup")).toHaveAttribute("data-visual", "app");
    expect(screen.queryByTestId("first-run-logo")).not.toBeInTheDocument();
    expect(document.querySelectorAll("script[data-onboarding-src]")).toHaveLength(scriptsBefore);
  });

  it("keeps a tall preview step reachable from the top", () => {
    renderShell(
      <FirstRunShell testId="preview" visual="preview" chrome={{ stepIndex: 1 }} contentSpacing="compact">
        <h1>Connect SuperPlane to GitHub</h1>
      </FirstRunShell>,
    );

    const content = screen.getByTestId("first-run-content");
    expect(content).toHaveClass("flex-col");
    expect(content).not.toHaveClass("items-center");
    expect(content.firstElementChild).toHaveClass("my-auto");
    expect(screen.getByRole("heading", { name: "Connect SuperPlane to GitHub" })).toBeInTheDocument();
  });

  it("gives the organization and workspace menus the preview colors", async () => {
    const user = userEvent.setup();
    renderShell(
      <FirstRunShell
        testId="preview"
        visual="preview"
        chrome={{
          stepIndex: 0,
          organizationSwitch: { currentOrganizationRouteId: "acme" },
          workspaceSwitch: {
            organizationId: "org-1",
            currentFactoryId: "factory-1",
            factories: [
              { id: "factory-1", name: "Payments" },
              { id: "factory-2", name: "Billing" },
            ],
          },
        }}
      >
        <h1>Welcome</h1>
      </FirstRunShell>,
    );

    await user.click(screen.getByTestId("first-run-workspace-switch"));
    const workspaceMenu = screen.getByTestId("first-run-workspace-menu");
    expect(workspaceMenu.style.getPropertyValue("--popover")).toBe("#201f1a");
    expect(workspaceMenu.style.getPropertyValue("--foreground")).toBe("#eeede9");
    expect(workspaceMenu).toHaveTextContent(FIRST_RUN_COPY.chrome.switchWorkspace);

    await user.click(screen.getByTestId("first-run-organization-switch"));
    const organizationMenu = screen.getByTestId("first-run-organization-menu");
    expect(organizationMenu.style.getPropertyValue("--popover")).toBe("#201f1a");
    expect(organizationMenu.style.getPropertyValue("--accent")).toBe("#201f1a");
  });
});
