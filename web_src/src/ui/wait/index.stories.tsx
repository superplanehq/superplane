import type { Meta, StoryObj } from '@storybook/react';
import { Wait } from './';

const meta: Meta<typeof Wait> = {
  title: 'ui/Wait',
  component: Wait,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const WaitSeconds: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 30,
      unit: "seconds",
    },
  },
};

export const WaitMinutes: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 5,
      unit: "minutes",
    },
  },
};

export const WaitHours: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 2,
      unit: "hours",
    },
  },
};

export const WaitCollapsed: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 10,
      unit: "minutes",
    },
    collapsed: true,
    collapsedBackground: "bg-yellow-50",
  },
};

export const WaitSelected: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 15,
      unit: "seconds",
    },
    selected: true,
  },
};

export const WaitNoDuration: Story = {
  args: {
    title: "Wait",
  },
};

export const WaitWithSuccessfulExecution: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 30,
      unit: "seconds",
    },
    lastExecution: {
      receivedAt: new Date(Date.now() - 2 * 60 * 1000), // 2 minutes ago
      state: "success",
    },
  },
};

export const WaitWithFailedExecution: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 5,
      unit: "minutes",
    },
    lastExecution: {
      receivedAt: new Date(Date.now() - 10 * 60 * 1000), // 10 minutes ago
      state: "failed",
    },
  },
};

export const WaitRunning: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 1,
      unit: "hours",
    },
    lastExecution: {
      receivedAt: new Date(Date.now() - 30 * 1000), // 30 seconds ago
      state: "running",
    },
  },
};

export const WaitNoExecution: Story = {
  args: {
    title: "Wait",
    duration: {
      value: 15,
      unit: "seconds",
    },
    lastExecution: undefined,
  },
};
