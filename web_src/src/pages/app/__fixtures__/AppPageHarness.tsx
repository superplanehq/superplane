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
  /**
   * Open the agent chat sidebar on mount (AI Chat story). Live Canvas leaves
   * this false so the Agent toggle is available but the panel starts closed.
   */
  openAgentSidebar?: boolean;
}

/**
 * Mounts `AppPage` against an in-process fixture backend. Shares a workspace
 * router with HomePage so the header Homepage control returns to the org home
 * surface used by the HomePage stories.
 */
export function AppPageHarness({ query = "", fixture, openAgentSidebar = false }: AppPageHarnessProps) {
  return (
    <OrgWorkspaceHarness startAt="app" appFixture={fixture} appQuery={query} openAgentSidebar={openAgentSidebar} />
  );
}
