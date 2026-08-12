import type { Meta, StoryObj } from "@storybook/react-vite";
import { FactoryNodeCard } from "./FactoryNodeCard";

const meta = {
  title: "Factories/UI/FactoryNodeCard",
  component: FactoryNodeCard,
  parameters: { layout: "centered" },
} satisfies Meta<typeof FactoryNodeCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Passed: Story = {
  args: {
    title: "Create Branch",
    iconSlug: "git-branch",
    metadata: [{ icon: "git-branch", label: "feat/invite-poc-from-branch" }],
    eventSections: [{ eventId: "1", eventState: "success", eventTitle: "Branch created", eventSubtitle: "7s" }],
  },
};

export const Running: Story = {
  args: {
    title: "Agent",
    iconSlug: "bot",
    metadata: [{ icon: "file-text", label: "Write Plan - plan.md - 12 steps" }],
    eventSections: [
      {
        eventId: "2",
        eventState: "running",
        eventTitle: "Running",
        receivedAt: new Date(Date.now() - 9 * 60_000),
      },
    ],
  },
};

export const Failed: Story = {
  args: {
    title: "wait",
    iconSlug: "alarm-clock",
    metadata: [{ icon: "clock", label: "Wait for 10 seconds" }],
    eventSections: [{ eventId: "3", eventState: "failed", eventTitle: "Failed", eventSubtitle: "1s" }],
  },
};

export const Pending: Story = {
  args: {
    title: "onRun",
    iconSlug: "play",
    eventSections: [{ eventId: "4", eventState: "queued", eventTitle: "Queued" }],
  },
};
