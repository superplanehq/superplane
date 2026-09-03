import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { MemoryRouter, Outlet, Route, Routes } from "react-router";

import { peekIntegrationSetupReturn, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { useRedirectIntegrationSetupReturn } from "./useRedirectIntegrationSetupReturn";

const ORGANIZATION_ID = "org-1";
const SETUP_PATH = "/org-1/workspaces/APP/setup?step=vcs&pick=newest";

function ProviderReturnGuard() {
  useRedirectIntegrationSetupReturn(ORGANIZATION_ID);
  return <Outlet />;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<ProviderReturnGuard />}>
          <Route
            path="/:organizationId/settings/integrations/:integrationId"
            element={<div>integration details</div>}
          />
          <Route path="/org-1/workspaces/APP/setup" element={<div>workspace setup</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("useRedirectIntegrationSetupReturn", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("returns a GitHub provider callback to workspace setup before settings can redirect", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);
    expect(peekIntegrationSetupReturn(ORGANIZATION_ID)).toBe(SETUP_PATH);

    renderAt("/org-1/settings/integrations/github-connection");

    expect(await screen.findByText("workspace setup")).toBeInTheDocument();
    expect(screen.queryByText("integration details")).not.toBeInTheDocument();
  });

  it("keeps the hosted-install picker on integration settings", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    renderAt("/org-1/settings/integrations/github-connection?setupStay=1");

    expect(await screen.findByText("integration details")).toBeInTheDocument();
    expect(screen.queryByText("workspace setup")).not.toBeInTheDocument();
  });
});
