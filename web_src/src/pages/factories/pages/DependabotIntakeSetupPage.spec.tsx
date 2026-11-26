import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext, type ResolvedFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath } from "../lib/factoryPagePaths";
import { DependabotIntakeSetupPage } from "./DependabotIntakeSetupPage";

let canUpdate = true;
const createIntake = vi.fn();

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdate, isLoading: false }),
}));

vi.mock("@/hooks/useFactoryGitHubHost", () => ({
  useFactoryGitHubHost: () => "",
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: createIntake, isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useImportFactoryIntakeItem: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useSearchDependabotIntakeSetupItems: () => ({ data: [], isLoading: false, isError: false }),
}));

const readyFactory = {
  ...REFUND_FACTORY,
  onboarding: { vcsIntegrationId: "github-1", backlogRepository: "acme/payments" },
};
const setupRoute = "/org-1/workspaces/:factoryKey/line/setup/dependabot";

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="dependabot-setup-location">{`${location.pathname}${location.search}`}</div>;
}

function renderSetup(factory: ResolvedFactoriesLayout["factory"] = readyFactory) {
  return render(
    <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/line/setup/dependabot`]}>
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org-1",
          factoryId: PRIMARY_FACTORY_ID,
          factoryKey: PRIMARY_FACTORY_KEY,
          routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
          factory,
          factories: factory ? [factory] : [],
          openCreateWorkOrder: () => {},
        }}
      >
        <Routes>
          <Route path={setupRoute} element={<DependabotIntakeSetupPage />} />
          <Route path="*" element={null} />
        </Routes>
        <LocationProbe />
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
}

describe("DependabotIntakeSetupPage", () => {
  const boardHref = factoryHomePath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT);

  beforeEach(() => {
    canUpdate = true;
    createIntake.mockReset();
    createIntake.mockResolvedValue({ id: "intake-1" });
  });

  it("keeps setup unready when onboarding data is missing", () => {
    renderSetup({ ...readyFactory, onboarding: undefined });

    expect(screen.getByTestId("dependabot-setup-finish")).toBeDisabled();
    expect(screen.getByRole("alert")).toBeVisible();
  });

  for (const field of ["vcsIntegrationId", "backlogRepository"] as const) {
    it.each([undefined, "", " \t\n "])(`keeps setup unready when ${field} is blank (%j)`, (value) => {
      renderSetup({ ...readyFactory, onboarding: { ...readyFactory.onboarding, [field]: value } });

      expect(screen.getByTestId("dependabot-setup-finish")).toBeDisabled();
    });
  }

  it.each([readyFactory.onboarding, { vcsIntegrationId: " \tgithub-1\n ", backlogRepository: " \tacme/payments\n " }])(
    "enables setup with both nonempty connection fields and displays the trimmed repository (%j)",
    (onboarding) => {
      renderSetup({ ...readyFactory, onboarding });

      expect(screen.getByTestId("dependabot-setup-finish")).toBeEnabled();
      expect(screen.getByTestId("dependabot-setup-repository").textContent).toBe("acme/payments");
    },
  );

  it("returns to the board when the user cannot update the factory", () => {
    canUpdate = false;
    renderSetup();

    expect(screen.getByTestId("dependabot-setup-location").textContent).toBe(boardHref);
    expect(screen.queryByTestId("dependabot-intake-setup-page")).not.toBeInTheDocument();
  });

  it.each([null, { ...readyFactory, lines: [] }])(
    "returns to the board when no workspace line is available (%j)",
    (factory) => {
      renderSetup(factory);

      expect(screen.getByTestId("dependabot-setup-location").textContent).toBe(boardHref);
    },
  );

  it("returns to the workspace board when setup is closed", async () => {
    const user = userEvent.setup();
    renderSetup();

    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("dependabot-setup-location").textContent).toBe(boardHref);
  });

  it("returns to the workspace board after intake creation", async () => {
    const user = userEvent.setup();
    renderSetup();

    await user.click(screen.getByTestId("dependabot-setup-finish"));
    await user.click(await screen.findByRole("button", { name: "Turn on Dependabot intake" }));

    expect(screen.getByTestId("dependabot-setup-location").textContent).toBe(boardHref);
  });
});
