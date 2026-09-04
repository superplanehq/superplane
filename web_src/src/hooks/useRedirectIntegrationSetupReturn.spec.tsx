import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { MemoryRouter, Outlet, Route, Routes, useSearchParams } from "react-router";

import { peekIntegrationSetupReturn, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { useRedirectIntegrationSetupReturn } from "./useRedirectIntegrationSetupReturn";

const ORGANIZATION_ID = "org-1";
const SETUP_PATH = "/org-1/workspaces/APP/setup?step=vcs&pick=newest";

function SetupLanding() {
  const [searchParams] = useSearchParams();
  return (
    <div>
      workspace setup
      {searchParams.toString() ? <span>{searchParams.toString()}</span> : null}
    </div>
  );
}

function ProviderReturnGuard({ routeOrganizationId = ORGANIZATION_ID }: { routeOrganizationId?: string }) {
  useRedirectIntegrationSetupReturn(routeOrganizationId, ORGANIZATION_ID);
  return <Outlet />;
}

function renderAt(path: string, routeOrganizationId = ORGANIZATION_ID) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<ProviderReturnGuard routeOrganizationId={routeOrganizationId} />}>
          <Route
            path="/:organizationId/settings/integrations/:integrationId"
            element={<div>integration details</div>}
          />
          <Route path="/org-1/workspaces/APP/setup" element={<SetupLanding />} />
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
    expect(peekIntegrationSetupReturn(ORGANIZATION_ID)).toBeNull();
  });

  it("returns a callback that uses the organization UID to a slug route", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    renderAt("/organization-uid/settings/integrations/github-connection", "organization-uid");

    expect(await screen.findByText("workspace setup")).toBeInTheDocument();
    expect(peekIntegrationSetupReturn(ORGANIZATION_ID)).toBeNull();
  });

  it("forwards a GitHub install request onto the stored return path", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    renderAt("/org-1/settings/integrations/github-connection?githubSetup=request");

    expect(await screen.findByText("workspace setup")).toBeInTheDocument();
    expect(screen.getByText(/githubSetup=request/)).toBeInTheDocument();
  });

  it("keeps the hosted-install picker on integration settings", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    renderAt("/org-1/settings/integrations/github-connection?setupStay=1");

    expect(await screen.findByText("integration details")).toBeInTheDocument();
    expect(screen.queryByText("workspace setup")).not.toBeInTheDocument();
  });
});
