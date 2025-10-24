import type { Meta, StoryObj } from '@storybook/react';
import { If } from './';

const meta: Meta<typeof If> = {
  title: 'ui/If',
  component: If,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "If processed events",
    conditions: [
      {
        field: "$.title",
        operator: "contains",
        value: "\"superplane\"",
        logicalOperator: "OR"
      },
      {
        field: "$.author",
        operator: "contains",
        value: "\"pedro\""
      }
    ],
    trueEvent: {
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully"
    },
    falseEvent: {
      receivedAt: new Date(),
      eventState: "failed",
      eventTitle: "Build failed"
    },
    trueSectionLabel: "TRUE",
    falseSectionLabel: "FALSE"
  },
};