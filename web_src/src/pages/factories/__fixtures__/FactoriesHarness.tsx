import type { CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { OrgWorkspaceHarness, type OrgWorkspacePageOverrides } from "@/pages/__fixtures__/OrgWorkspaceHarness";
import type { HomePageFixture } from "@/pages/home/__fixtures__/handlers";
import { defaultHomePageFixture } from "@/pages/home/__fixtures__/homePageResponses";

import { defaultFactoriesFixture, FACTORIES_ORGANIZATION_ID, type FactoriesFixture } from "./factoryPageResponses";

interface FactoriesHarnessProps {
  /** Path under the org. Defaults to `workspaces` (list page). */
  pathSuffix?: string;
  /** Fixture backing the factories API. Defaults to the populated Refunds Factory dataset. */
  factoriesFixture?: FactoriesFixture;
  /** Optional canvas fixture for factory-embedded AppPage routes. */
  appFixture?: CanvasAppFixture;
  /** Storybook-only: replace selected factory page elements (e.g. wiki wireframe). */
  pageOverrides?: OrgWorkspacePageOverrides;
}

/**
 * Mounts the org home routes with the factories feature enabled and a fixture
 * backend for factory list/detail/orders/lines/apps endpoints. Shares the same
 * `OrgWorkspaceHarness` shell so links between Home → Workspaces → App work.
 */
export function FactoriesHarness({
  pathSuffix = "workspaces",
  factoriesFixture = defaultFactoriesFixture,
  appFixture,
  pageOverrides,
}: FactoriesHarnessProps) {
  const homeFixture: HomePageFixture = {
    ...defaultHomePageFixture,
    organizationId: factoriesFixture.organizationId ?? FACTORIES_ORGANIZATION_ID,
    enabledExperimentalFeatures: [...(defaultHomePageFixture.enabledExperimentalFeatures ?? []), "factories"],
    factories: factoriesFixture.factories.map((factory) => ({
      id: factory.id ?? "",
      name: factory.name ?? "",
      description: factory.description ?? "",
    })),
  };

  return (
    <OrgWorkspaceHarness
      startAt="home"
      pathSuffix={pathSuffix}
      homeFixture={homeFixture}
      factoriesFixture={factoriesFixture}
      appFixture={appFixture}
      pageOverrides={pageOverrides}
    />
  );
}
