import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";

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
 * so the graph, runs sidebar, versions, agent chat, and run inspector render
 * deterministic fake data however Storybook is opened.
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

/**
 * Live canvas view: the ReactFlow graph plus the runs history sidebar.
 * Agent chat is enabled (header Agent toggle) but starts closed.
 */
export const LiveCanvas: Story = {
  render: () => <AppPageHarness />,
};

/**
 * AI chat: same Software Factory canvas with the agent sidebar open and a
 * seeded transcript (user + assistant turns).
 */
export const AIChat: Story = {
  name: "AI Chat",
  render: () => <AppPageHarness openAgentSidebar />,
};

/**
 * Run inspection: a finished (passed) run is selected and the right inspector
 * is opened on the `runner-implement` (Implementation) node, showing that node's
 * execution output for the run.
 */
export const RunInspection: Story = {
  render: () => <AppPageHarness query={`run=${canvasAppIds.publishedRunId}&sidebar=1&node=runner-implement`} />,
};

/**
 * Editing view: the same Software Factory canvas after entering an edit session
 * (the `play` step clicks the header Edit button). Edit mode reveals the canvas
 * zoom controls' editing affordances, including the new layout-direction toggle
 * that flips the canvas between horizontal (freeform) and vertical (top-to-bottom
 * pipeline) auto-layout.
 */
export const EditingLayoutControls: Story = {
  render: () => <AppPageHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const editButton = await canvas.findByTestId("canvas-edit-button", {}, { timeout: 10000 });
    await userEvent.click(editButton);

    // Once the edit session is active the zoom controls expose the layout
    // direction toggle; assert it renders so the story fails loudly if the
    // control is ever accidentally hidden from edit mode.
    await waitFor(() => expect(canvas.getByTestId("canvas-layout-direction-toggle")).toBeInTheDocument(), {
      timeout: 10000,
    });
  },
};
