import type { Meta, StoryObj } from "@storybook/react-vite";

import { AppPage } from "./index";
import { AppPageHarness } from "./__fixtures__/AppPageHarness";
import { canvasAppIds } from "./__fixtures__/handlers";

/**
 * Mounts the real `AppPage` orchestrator against an in-process fixture backend
 * seeded from a live canvas capture (see `__fixtures__/canvasAppResponses.json`).
 * The default capture is **Software Factory** (sourced from the Sentry Exception
 * Solver canvas on app.superplane.com, renamed in-fixture).
 *
 * Shares a router with HomePage: the header Homepage control returns to the
 * populated org home surface (same as Pages/HomePage → Current).
 *
 * Networking is faked by overriding `window.fetch` rather than MSW: MSW relies
 * on a Service Worker, which is silently disabled in non-secure contexts
 * (opening Storybook via a LAN IP instead of `localhost`), causing every
 * request to escape to the live API. The fetch override has no such dependency,
 * so the graph, runs sidebar, versions, and run inspector render deterministic
 * fake data however Storybook is opened.
 */
const meta = {
  title: "Pages/AppPage",
  component: AppPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof AppPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Live canvas view: the ReactFlow graph plus the runs history sidebar. */
export const LiveCanvas: Story = {
  render: () => <AppPageHarness />,
};

/**
 * Run inspection: a finished (passed) run is selected and the right inspector
 * is opened on the `runner-implement` (Implementation) node, showing that node's
 * execution output for the run.
 */
export const RunInspection: Story = {
  render: () => <AppPageHarness query={`run=${canvasAppIds.publishedRunId}&sidebar=1&node=runner-implement`} />,
};
