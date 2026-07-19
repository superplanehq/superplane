import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import type { CanvasAppFixture } from "./handlers";

interface AppPageHarnessProps {
  /** Query string appended to the AppPage route (without the leading `?`). */
  query?: string;
  /**
   * Fixture to serve for this story. Defaults to the Software Factory
   * capture used by the original `LiveCanvas`/`RunInspection` stories.
   */
  fixture?: CanvasAppFixture;
}

/**
 * Mounts `AppPage` against an in-process fixture backend. Shares a workspace
 * router with HomePage so the header Homepage control returns to the org home
 * surface used by the HomePage stories.
 */
export function AppPageHarness({ query = "", fixture }: AppPageHarnessProps) {
  return <OrgWorkspaceHarness startAt="app" appFixture={fixture} appQuery={query} />;
}
