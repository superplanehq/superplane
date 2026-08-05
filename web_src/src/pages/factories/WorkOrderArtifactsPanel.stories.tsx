import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { OPEN_WORK_ORDER_ARTIFACTS } from "./__fixtures__/factoryPageResponses";
import { WorkOrderArtifactsPanel } from "./WorkOrderArtifactsPanel";

/**
 * Work-order detail sidebar section: lists attached PRs / markdown notes and
 * exposes the "Attach" trigger that opens the artifact dialog.
 */
const meta = {
  title: "Factories/WorkOrderArtifactsPanel",
  component: WorkOrderArtifactsPanel,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[280px] max-w-sm bg-white p-4 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderArtifactsPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

const noop = () => undefined;

/** Populated — PR + markdown note, both clickable if a URL is present. */
export const Populated: Story = {
  args: {
    artifacts: OPEN_WORK_ORDER_ARTIFACTS,
    isLoading: false,
    canAttach: true,
    onOpenAttach: noop,
  },
};

/** Empty state — encourages attaching the first artifact. */
export const Empty: Story = {
  args: {
    artifacts: [],
    isLoading: false,
    canAttach: true,
    onOpenAttach: noop,
  },
};

/** Loading — placeholder text while the artifacts query resolves. */
export const Loading: Story = {
  args: {
    artifacts: [],
    isLoading: true,
    canAttach: true,
    onOpenAttach: noop,
  },
};

/** Error — inline failure message. */
export const ErrorState: Story = {
  name: "Error",
  args: {
    artifacts: [],
    isLoading: false,
    error: new Error("Failed to load artifacts"),
    canAttach: true,
    onOpenAttach: noop,
  },
};

/** Read-only viewer — Attach button disabled. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    artifacts: OPEN_WORK_ORDER_ARTIFACTS,
    isLoading: false,
    canAttach: false,
    onOpenAttach: noop,
  },
};
