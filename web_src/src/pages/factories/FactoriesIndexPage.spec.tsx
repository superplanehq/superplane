import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_LINE_ID,
  EMPTY_FACTORY,
  FACTORIES_ORGANIZATION_ID,
} from "./__fixtures__/factoryPageResponses";
import { FactoriesIndexPage } from "./FactoriesIndexPage";
import { factoryHomePath, newFactoryPath } from "./lib/factoryPagePaths";

const listedFactories: Array<typeof ACME_ONBOARDING_FACTORY> = [];
const permissionsState = vi.hoisted(() => ({ canCreate: true, isLoading: false }));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => ({ data: listedFactories, isLoading: false, error: null }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: null }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) =>
      resource === "factories" && action === "create" && permissionsState.canCreate,
    isLoading: permissionsState.isLoading,
  }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

function CurrentPath() {
  return <span data-testid="location-path">{useLocation().pathname}</span>;
}

function renderIndex() {
  return render(
    <MemoryRouter initialEntries={[`/${FACTORIES_ORGANIZATION_ID}/workspaces`]}>
      <Routes>
        <Route path="/:organizationId/workspaces" element={<FactoriesIndexPage />} />
        <Route path="/:organizationId/workspaces/new" element={<CurrentPath />} />
        <Route path="/:organizationId/workspaces/:factoryKey/*" element={<CurrentPath />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("FactoriesIndexPage", () => {
  beforeEach(() => {
    listedFactories.length = 0;
    permissionsState.canCreate = true;
    permissionsState.isLoading = false;
  });

  it("opens the unique line when the workspace has one line", () => {
    listedFactories.splice(0, listedFactories.length, ACME_ONBOARDING_FACTORY);
    renderIndex();

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factoryHomePath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!, ACME_ONBOARDING_LINE_ID),
    );
  });

  it("opens the workspace index when the workspace has no line", () => {
    listedFactories.splice(0, listedFactories.length, EMPTY_FACTORY);
    renderIndex();

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factoryHomePath(FACTORIES_ORGANIZATION_ID, EMPTY_FACTORY.key!),
    );
  });

  it("opens a finished workspace when another workspace is still in setup", () => {
    listedFactories.splice(0, listedFactories.length, EMPTY_FACTORY, ACME_ONBOARDING_FACTORY);
    renderIndex();

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factoryHomePath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!, ACME_ONBOARDING_LINE_ID),
    );
  });

  it("opens workspace setup when the organization has no workspace", () => {
    renderIndex();

    expect(screen.getByTestId("location-path")).toHaveTextContent(newFactoryPath(FACTORIES_ORGANIZATION_ID));
  });

  it("asks an admin to finish setup when the user cannot create a workspace", () => {
    permissionsState.canCreate = false;
    renderIndex();

    expect(screen.getByTestId("factories-empty-state")).toBeInTheDocument();
    expect(screen.getByText("An organization admin must finish setup")).toBeInTheDocument();
    expect(screen.queryByTestId("location-path")).not.toBeInTheDocument();
  });
});
