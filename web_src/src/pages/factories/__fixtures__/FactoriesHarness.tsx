import type { CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { OrgWorkspaceHarness, type OrgWorkspacePageOverrides } from "@/pages/__fixtures__/OrgWorkspaceHarness";
import type { HomePageFixture } from "@/pages/home/__fixtures__/handlers";
import { defaultHomePageFixture } from "@/pages/home/__fixtures__/homePageResponses";

import { OnboardingStorybookProvider } from "../pages/onboarding/OnboardingStorybookContext";
import { OnboardingWireframe } from "../pages/onboarding/OnboardingWireframe";
import type { OnboardingStorybookSeed } from "../pages/onboarding/onboardingMocks";
import { StorybookOverviewPage } from "../pages/onboarding/StorybookOverviewPage";
import { WikiWireframe } from "../pages/wiki/WikiWireframe";
import { WIKI_DOCUMENTS_DEFAULT, WIKI_DOCUMENTS_REFRESHED } from "../pages/wiki/wikiMocks";
import { defaultFactoriesFixture, FACTORIES_ORGANIZATION_ID, type FactoriesFixture } from "./factoryPageResponses";

interface FactoriesHarnessProps {
  /** Path under the org. Defaults to `workspaces` (list page). */
  pathSuffix?: string;
  /** Fixture backing the factories API. Defaults to the populated Refunds Factory dataset. */
  factoriesFixture?: FactoriesFixture;
  /** Optional canvas fixture for factory-embedded AppPage routes. */
  appFixture?: CanvasAppFixture;
  /**
   * Storybook-only: replace selected factory page elements.
   * Wiki defaults to the wireframe so sidebar navigation shows it; pass
   * `pageOverrides={{ wiki: WikiPage }}` to keep Coming Soon.
   * Onboarding + Get started overview are enabled by default.
   */
  pageOverrides?: OrgWorkspacePageOverrides;
  /** Seed pending providers/repos/tips for onboarding stories. */
  onboardingSeed?: OnboardingStorybookSeed;
  /** When false, skip onboarding provider/routes (app-like create → overview). */
  enableOnboarding?: boolean;
}

function DefaultWikiWireframe() {
  return <WikiWireframe initialDocuments={WIKI_DOCUMENTS_DEFAULT} refreshedDocuments={WIKI_DOCUMENTS_REFRESHED} />;
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
  onboardingSeed,
  enableOnboarding = true,
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

  const harness = (
    <OrgWorkspaceHarness
      startAt="home"
      pathSuffix={pathSuffix}
      homeFixture={homeFixture}
      factoriesFixture={factoriesFixture}
      appFixture={appFixture}
      pageOverrides={
        enableOnboarding
          ? {
              wiki: DefaultWikiWireframe,
              overview: StorybookOverviewPage,
              onboarding: OnboardingWireframe,
              ...pageOverrides,
            }
          : { wiki: DefaultWikiWireframe, ...pageOverrides }
      }
    />
  );

  if (!enableOnboarding) {
    return harness;
  }

  return <OnboardingStorybookProvider initial={onboardingSeed}>{harness}</OnboardingStorybookProvider>;
}
