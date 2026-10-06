import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "bun:test";

vi.mock("@/pages/organization/settings/Members", () => ({
  Members: () => <div data-testid="members-body" />,
}));

import { FactoryOrganizationMembersPage } from "./FactoryOrganizationSettingsPages";

describe("FactoryOrganizationMembersPage", () => {
  it("uses the wide settings column so the member table stays in view", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/settings/organization/members"]}>
        <Routes>
          <Route path="/:organizationId/settings/organization/members" element={<FactoryOrganizationMembersPage />} />
        </Routes>
      </MemoryRouter>,
    );

    // Members is a settings table, so its column matches the wide settings
    // max-width, not the short form measure.
    expect(screen.getByTestId("workspace-page-header").className).toContain("max-w-6xl");
    expect(screen.getByTestId("members-body")).toBeInTheDocument();
  });
});
