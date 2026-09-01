import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router";

import { peekIntegrationSetupReturn, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { IntegrationSetupReturn } from "./IntegrationSetupReturn";

const ORGANIZATION_ID = "org-1";
const SETUP_PATH = "/org-1/workspaces/APP/setup";

function renderAt(path: string, children: React.ReactNode) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/:organizationId/settings/integrations/:integrationId"
          element={<IntegrationSetupReturn organizationId={ORGANIZATION_ID}>{children}</IntegrationSetupReturn>}
        />
        <Route path="/org-1/workspaces/APP/setup" element={<div>workspace setup</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("IntegrationSetupReturn", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("returns to the stored path even when the provider redirects to an unknown integration", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    renderAt("/org-1/settings/integrations/an-id-created-during-setup", <div>integration details</div>);

    expect(await screen.findByText("workspace setup")).toBeInTheDocument();
    expect(screen.queryByText("integration details")).not.toBeInTheDocument();
  });

  it("keeps the marker so a remount can still return to setup", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    const first = renderAt("/org-1/settings/integrations/an-id-created-during-setup", <div>integration details</div>);
    expect(await screen.findByText("workspace setup")).toBeInTheDocument();
    expect(peekIntegrationSetupReturn(ORGANIZATION_ID)).toBe(SETUP_PATH);
    first.unmount();

    renderAt("/org-1/settings/integrations/an-id-created-during-setup", <div>integration details</div>);
    expect(await screen.findByText("workspace setup")).toBeInTheDocument();
    expect(peekIntegrationSetupReturn(ORGANIZATION_ID)).toBe(SETUP_PATH);
  });

  it("stays on the integration page when setupStay is set", async () => {
    rememberIntegrationSetupReturn(ORGANIZATION_ID, SETUP_PATH);

    renderAt("/org-1/settings/integrations/abc?setupStay=1", <div>integration details</div>);

    expect(await screen.findByText("integration details")).toBeInTheDocument();
    expect(screen.queryByText("workspace setup")).not.toBeInTheDocument();
    expect(peekIntegrationSetupReturn(ORGANIZATION_ID)).toBe(SETUP_PATH);
  });

  it("renders the page when no setup is in progress", () => {
    renderAt("/org-1/settings/integrations/abc", <div>integration details</div>);

    expect(screen.getByText("integration details")).toBeInTheDocument();
  });
});
