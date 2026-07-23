import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import type { HomePageFixture } from "./handlers";

interface HomePageHarnessProps {
  /**
   * Fixture to serve for this story. Defaults to the populated current-homepage
   * seed (apps + folders).
   */
  fixture?: HomePageFixture;
  /**
   * Path under the org (without leading slash), e.g. `apps/new`.
   * Defaults to the org index (`/:organizationId`).
   */
  pathSuffix?: string;
}

/**
 * Mounts org home routes against an in-process fixture backend. Shares a
 * workspace router with AppPage so clicking Software Factory opens the live
 * canvas story surface (and the logo can navigate back home from the app).
 */
export function HomePageHarness({ fixture, pathSuffix = "" }: HomePageHarnessProps) {
  return <OrgWorkspaceHarness startAt="home" homeFixture={fixture} pathSuffix={pathSuffix} />;
}
