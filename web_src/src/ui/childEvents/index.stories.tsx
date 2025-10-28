import type { Meta, StoryObj } from '@storybook/react';
import { ChildEvents } from './';

const meta: Meta<typeof ChildEvents> = {
  title: 'ui/ChildEvents',
  component: ChildEvents,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    childEventsInfo: {
      count: 3,
      state: "processed",
      waitingInfos: [],
    },
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const WithWaitingInfos: Story = {
  args: {
    childEventsInfo: {
      count: 5,
      state: "waiting",
      waitingInfos: [
        {
          icon: "clock",
          info: "Waiting for deployment",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 30), // 30 minutes from now
        },
        {
          icon: "shield-check",
          info: "Security scan pending",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 45), // 45 minutes from now
        },
        {
          icon: "database",
          info: "Database migration",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 60 * 2), // 2 hours from now
        },
      ],
    },
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const SingleEvent: Story = {
  args: {
    childEventsInfo: {
      count: 1,
      state: "running",
      waitingInfos: [],
    },
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const ProcessingState: Story = {
  args: {
    childEventsInfo: {
      count: 8,
      state: "processed",
      waitingInfos: [
        {
          icon: "check",
          info: "All tests passed",
          futureTimeDate: new Date(Date.now() - 1000 * 60 * 15), // 15 minutes ago
        },
      ],
    },
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const DiscardedState: Story = {
  args: {
    childEventsInfo: {
      count: 2,
      state: "discarded",
      waitingInfos: [],
    },
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const NoActions: Story = {
  args: {
    childEventsInfo: {
      count: 4,
      state: "processed",
      waitingInfos: [],
    },
  },
};

export const LongWaitingList: Story = {
  args: {
    childEventsInfo: {
      count: 12,
      state: "waiting",
      waitingInfos: [
        {
          icon: "server",
          info: "Server provisioning",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 10),
        },
        {
          icon: "package",
          info: "Package installation",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 20),
        },
        {
          icon: "settings",
          info: "Configuration setup",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 30),
        },
        {
          icon: "database",
          info: "Database initialization",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 40),
        },
        {
          icon: "globe",
          info: "DNS propagation",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 60),
        },
      ],
    },
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};