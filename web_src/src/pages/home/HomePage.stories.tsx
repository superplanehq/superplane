import type { Meta, StoryObj } from "@storybook/react-vite";

import { HomePage } from "./index";
import { HomePageHarness } from "./__fixtures__/HomePageHarness";
import { emptyHomePageFixture } from "./__fixtures__/homePageResponses";

/**
 * Mounts the real org home routes against an in-process fixture backend.
 * Use **Current** for the populated homepage baseline, and **FreshOrg** for the
 * empty-org create screen.
 *
 * **Current** shares a router with AppPage: clicking Software Factory opens the
 * live canvas surface (same as Pages/AppPage → Live Canvas).
 *
 * **Fresh Org (Prototype)** mounts the factory-first landing for Storybook only;
 * production `/apps/new` still uses `ZeroStatePage`.
 *
 * Networking is faked by overriding `window.fetch` rather than MSW — same
 * approach and rationale as `AppPage.stories.tsx`.
 */
const meta = {
  title: "Pages/HomePage",
  component: HomePage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof HomePage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Populated homepage: Apps header, toolbar, folder sections, and app cards. */
export const Current: Story = {
  render: () => <HomePageHarness />,
};

/**
 * Production empty-org path: HomePage redirects to `/apps/new` → `ZeroStatePage`.
 */
export const FreshOrg: Story = {
  name: "Fresh Org",
  render: () => <HomePageHarness fixture={emptyHomePageFixture} />,
};

/**
 * Storybook-only prototype of the factory-first new-app landing. Does not change
 * the production `NewAppPage` route.
 */
export const FreshOrgPrototype: Story = {
  name: "Fresh Org (Prototype)",
  render: () => <HomePageHarness fixture={emptyHomePageFixture} prototypeNewApp />,
};
