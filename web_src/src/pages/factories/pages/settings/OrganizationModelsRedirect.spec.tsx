import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { OrganizationModelsRedirect } from "./FactorySettingsRedirects";

function Destination() {
  const location = useLocation();
  return <output>{`${location.pathname}${location.search}${location.hash}`}</output>;
}

describe("organization model settings bookmarks", () => {
  it("redirects to the workspace page with the query and hash", async () => {
    render(
      <MemoryRouter initialEntries={["/org/workspaces/TEAM/settings/organization/models?provider=openai#selected"]}>
        <Routes>
          <Route
            path="/:organizationId/workspaces/:factoryKey/settings/organization/models"
            element={<OrganizationModelsRedirect />}
          />
          <Route path="/:organizationId/workspaces/:factoryKey/settings/workspace/models" element={<Destination />} />
        </Routes>
      </MemoryRouter>,
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "/org/workspaces/team/settings/workspace/models?provider=openai#selected",
    );
  });
});
