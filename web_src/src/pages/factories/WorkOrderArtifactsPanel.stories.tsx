import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { OPEN_WORK_ORDER_ARTIFACTS } from "./__fixtures__/factoryPageResponses";
import { WorkOrderArtifactsPanel } from "./WorkOrderArtifactsPanel";

/**
 * Work-order detail sidebar section: lists PRs and markdown notes attached
 * by automation nodes. This panel is read-only from the UI; manual attach
 * was removed and artifacts are added exclusively through
 * `addWorkOrderArtifact` canvas components.
 */
const meta = {
  title: "Factories/WorkOrderArtifactsPanel",
  component: WorkOrderArtifactsPanel,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-70 max-w-sm bg-white p-4 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderArtifactsPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Populated — PR + markdown note, both clickable if a URL is present. */
export const Populated: Story = {
  args: {
    artifacts: OPEN_WORK_ORDER_ARTIFACTS,
    isLoading: false,
  },
};

/** Empty state — text prompt while no automation has attached artifacts yet. */
export const Empty: Story = {
  args: {
    artifacts: [],
    isLoading: false,
  },
};

/** Loading — placeholder text while the artifacts query resolves. */
export const Loading: Story = {
  args: {
    artifacts: [],
    isLoading: true,
  },
};

/** Error — inline failure message. */
export const ErrorState: Story = {
  name: "Error",
  args: {
    artifacts: [],
    isLoading: false,
    error: new Error("Failed to load artifacts"),
  },
};
