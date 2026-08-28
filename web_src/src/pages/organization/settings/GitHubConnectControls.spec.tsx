import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { CREATE_PRIVATE_GITHUB_APP_LABEL, githubPrivateAppSetupPath } from "@/lib/privateGitHubApp";
import { GitHubConnectControls } from "./GitHubConnectControls";

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location-path">{location.pathname}</span>;
}

function renderControls(onConnect = vi.fn()) {
  return render(
    <MemoryRouter initialEntries={["/org-1/settings/integrations"]}>
      <TooltipProvider>
        <Routes>
          <Route
            path="*"
            element={
              <>
                <GitHubConnectControls
                  organizationId="org-1"
                  definition={{ name: "github", label: "GitHub", hostedAppInstall: true }}
                  canCreateIntegrations
                  permissionsLoading={false}
                  onConnect={onConnect}
                />
                <LocationProbe />
              </>
            }
          />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("GitHubConnectControls", () => {
  it("keeps Connect as the hosted SuperPlane App path", async () => {
    const user = userEvent.setup();
    const onConnect = vi.fn();
    renderControls(onConnect);

    await user.click(screen.getByTestId("integrations-connect-github"));
    expect(onConnect).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("location-path")).toHaveTextContent("/org-1/settings/integrations");
  });

  it("opens the customer GitHub App wizard from the private-app link", async () => {
    const user = userEvent.setup();
    renderControls();

    const link = screen.getByTestId("integrations-create-private-github-app");
    expect(link).toHaveTextContent(CREATE_PRIVATE_GITHUB_APP_LABEL);
    await user.click(link);
    expect(screen.getByTestId("location-path")).toHaveTextContent(githubPrivateAppSetupPath("org-1"));
  });
});
