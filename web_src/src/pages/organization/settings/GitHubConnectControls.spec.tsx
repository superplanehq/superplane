import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { CREATE_PRIVATE_GITHUB_APP_LABEL } from "@/lib/privateGitHubApp";
import { TooltipProvider } from "@/ui/tooltip";
import { GitHubConnectControls } from "./GitHubConnectControls";

function renderControls(onConnect = vi.fn(), allowPrivateApp = true) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <GitHubConnectControls
          definition={{ name: "github", label: "GitHub", hostedAppInstall: true, legacySetupOnly: false }}
          canCreateIntegrations
          permissionsLoading={false}
          onConnect={onConnect}
          allowPrivateApp={allowPrivateApp}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("GitHubConnectControls", () => {
  it("offers only private GitHub setup from integration settings", async () => {
    const user = userEvent.setup();
    const onConnect = vi.fn();
    renderControls(onConnect);

    const button = screen.getByTestId("integrations-connect-github");
    expect(button).toHaveTextContent(CREATE_PRIVATE_GITHUB_APP_LABEL);
    await user.click(button);
    expect(onConnect).toHaveBeenCalledTimes(1);
  });

  it("leaves the public GitHub App flow to Factory onboarding", () => {
    renderControls(vi.fn(), false);
    expect(screen.queryByTestId("integrations-connect-github")).not.toBeInTheDocument();
  });
});
