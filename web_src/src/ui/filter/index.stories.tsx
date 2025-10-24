import type { Meta, StoryObj } from '@storybook/react';
import { Filter } from './';

const meta: Meta<typeof Filter> = {
  title: 'ui/Filter',
  component: Filter,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Filter events based on branch",
    filters: [
      {
        field: "$.monarch_app.branch",
        operator: "is",
        value: "\"main\"",
        logicalOperator: "AND"
      },
      {
        field: "$.monarch_app.branch",
        operator: "contains",
        value: "\"dev\"",
        logicalOperator: "AND"
      },
      {
        field: "$.monarch_app.branch",
        operator: "ends with",
        value: "\"superplane\""
      }
    ],
    lastEvent: {
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully"
    }
  },
};