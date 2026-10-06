import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

vi.mock("./Members", () => ({
  Members: () => <div data-testid="members-body" />,
}));

vi.mock("../../../hooks/useOrganizationData", () => ({
  useOrganization: () => ({
    data: { metadata: { id: "org-1", name: "Acme" } },
    isLoading: false,
    error: null,
  }),
}));

vi.mock("../../../contexts/useAccount", () => ({
  useAccount: () => ({
    account: { name: "Ada Lovelace", email: "ada@example.com" },
    loading: false,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: () => true,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => false,
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  }),
}));

vi.mock("./General", () => ({ General: () => null }));

import { OrganizationSettings } from "./index";

function settingsColumnClassName(content: HTMLElement) {
  const column = content.closest("[class*='max-w-3xl'], [class*='max-w-6xl']");
  if (!(column instanceof HTMLElement)) {
    throw new Error("settings column not found");
  }
  return column.className;
}

function renderSettings(path: string) {
  return render(
    <ThemeProvider>
      <TooltipProvider>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path="/:organizationId/settings/*" element={<OrganizationSettings />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>
    </ThemeProvider>,
  );
}

describe("OrganizationSettings members column", () => {
  it("uses the wide settings column around the member table", () => {
    renderSettings("/org-1/settings/members");

    const bodyClassName = settingsColumnClassName(screen.getByTestId("members-body"));
    expect(bodyClassName).toContain("max-w-6xl");
    expect(bodyClassName).not.toContain("max-w-3xl");
  });

  it("keeps other settings sections on the short column", () => {
    renderSettings("/org-1/settings/general");

    const bodyClassName = settingsColumnClassName(screen.getByRole("heading", { name: "Settings" }));
    expect(bodyClassName).toContain("max-w-3xl");
    expect(bodyClassName).not.toContain("max-w-6xl");
  });
});
