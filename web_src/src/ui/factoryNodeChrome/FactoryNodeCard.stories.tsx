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
    title: "feat/invite-poc",
    componentLabel: "Create Branch",
    nodeName: "feat/invite-poc",
    iconSlug: "git-branch",
    eventSections: [{ eventId: "1", eventState: "success", eventTitle: "Branch created", eventSubtitle: "7s" }],
  },
};

export const Running: Story = {
  args: {
    title: "Write Plan",
    componentLabel: "Agent",
    nodeName: "Write Plan - plan.md - 12 steps",
    iconSlug: "bot",
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
    title: "Wait for 10 seconds",
    componentLabel: "wait",
    nodeName: "Wait for 10 seconds",
    iconSlug: "alarm-clock",
    eventSections: [{ eventId: "3", eventState: "failed", eventTitle: "Failed", eventSubtitle: "1s" }],
  },
};

export const Pending: Story = {
  args: {
    title: "onRun",
    componentLabel: "On Run",
    nodeName: "onRun",
    iconSlug: "play",
    canvasMode: "live",
    eventSections: [{ eventId: "4", eventState: "neutral", eventTitle: "Idle" }],
  },
};

/** Edit mode: neutral "No run" footer (not Pending). */
export const EditMode: Story = {
  args: {
    title: "Wait for 10 seconds",
    componentLabel: "wait",
    nodeName: "Wait for 10 seconds",
    iconSlug: "alarm-clock",
    canvasMode: "edit",
  },
};

export const Queued: Story = {
  args: {
    title: "wait",
    componentLabel: "wait",
    nodeName: "wait",
    iconSlug: "alarm-clock",
    eventSections: [{ eventId: "5", eventState: "queued", eventTitle: "Queued" }],
  },
};

export const Cancelled: Story = {
  args: {
    title: "Agent",
    componentLabel: "Agent",
    nodeName: "Babysit",
    iconSlug: "bot",
    eventSections: [{ eventId: "6", eventState: "cancelled", eventTitle: "Cancelled", eventSubtitle: "12s" }],
  },
};

export const Error: Story = {
  args: {
    title: "Notify",
    componentLabel: "Send Message",
    nodeName: "Notify",
    iconSlug: "message-square",
    eventSections: [{ eventId: "7", eventState: "error", eventTitle: "Error", eventSubtitle: "2s" }],
  },
};
