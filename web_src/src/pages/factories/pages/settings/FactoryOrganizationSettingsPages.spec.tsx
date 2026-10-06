import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "bun:test";

vi.mock("@/pages/organization/settings/Members", () => ({
  Members: () => <div data-testid="members-body" />,
}));

import { FactoryOrganizationMembersPage } from "./FactoryOrganizationSettingsPages";

function settingsColumnClassName(content: HTMLElement) {
  const column = content.closest("[class*='max-w-3xl'], [class*='max-w-6xl']");
  if (!(column instanceof HTMLElement)) {
    throw new Error("settings column not found");
  }
  return column.className;
}

describe("FactoryOrganizationMembersPage", () => {
  it("uses the wide settings column so the member table stays in view", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/settings/organization/members"]}>
        <Routes>
          <Route path="/:organizationId/settings/organization/members" element={<FactoryOrganizationMembersPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("workspace-page-header").className).toContain("max-w-6xl");
    const bodyClassName = settingsColumnClassName(screen.getByTestId("members-body"));
    expect(bodyClassName).toContain("max-w-6xl");
    expect(bodyClassName).not.toContain("max-w-3xl");
  });
});
