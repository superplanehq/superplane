import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { SPLIT_RUN_SUPER503 } from "./splitRunSuper503Fixture";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";

const STORY_CLIENT = new QueryClient({
  defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
});

/**
 * The task popup as it ships today, on the Automations tab. Rendered from
 * SUPER-503. No organization id, so live canvas and runner log hooks stay
 * off and the fixture stream is what you see.
 */
const meta = {
  title: "Factories/Pages/Task Split Run/Current design",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <QueryClientProvider client={STORY_CLIENT}>
        <ComponentStoryShell className="relative min-h-svh bg-muted/30 p-0">
          <Story />
        </ComponentStoryShell>
      </QueryClientProvider>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

/** SUPER-503, Automations tab. Analysis, Implement, PR #7771, then closure. */
export const AutomationsTab: Story = {
  name: "Automations tab",
  render: () => <WorkOrderSplitRunPopup fixture={SPLIT_RUN_SUPER503} initialTab="log" />,
};
