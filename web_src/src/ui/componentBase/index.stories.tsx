import type { Meta, StoryObj } from '@storybook/react';
import { ComponentBase, type ComponentBaseProps } from './';
import dockerIcon from '@/assets/icons/integrations/docker.svg';

const FilterComponentProps: ComponentBaseProps = {
  title: "Filter events based on branch",
  iconSlug: "filter",
  headerColor: "bg-gray-50",
  spec: {
    title: "filter",
    tooltipTitle: "filters applied",
    values: [
      {
        badges: [
          { label: "$.monarch_app.branch", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "is", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"main\"", bgColor: "bg-green-100", textColor: "text-green-700" },
          { label: "AND", bgColor: "bg-gray-500", textColor: "text-white" }
        ]
      },
      {
        badges: [
          { label: "$.monarch_app.branch", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "contains", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"dev\"", bgColor: "bg-green-100", textColor: "text-green-700" },
          { label: "AND", bgColor: "bg-gray-500", textColor: "text-white" }
        ]
      },
      {
        badges: [
          { label: "$.monarch_app.branch", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "ends with", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"superplane\"", bgColor: "bg-green-100", textColor: "text-green-700" },
        ]
      }
    ]
  },
  eventSections: [
    {
      title: "Last Event",
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully"
    }
  ]
}

const IfComponentProps: ComponentBaseProps = {
  title: "If processed events",
  iconSlug: "split",
  headerColor: "bg-gray-50",
  spec: {
    title: "condition",
    tooltipTitle: "conditions applied",
    values: [
      {
        badges: [
          { label: "$.title", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "contains", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"superplane\"", bgColor: "bg-green-100", textColor: "text-green-700" },
          { label: "OR", bgColor: "bg-gray-500", textColor: "text-white" }
        ]
      },
      {
        badges: [
          { label: "$.author", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "contains", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"pedro\"", bgColor: "bg-green-100", textColor: "text-green-700" },
        ]
      }
    ]
  },
  eventSections: [
    {
      title: "TRUE",
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully"
    },
    {
      title: "FALSE",
      receivedAt: new Date(),
      eventState: "failed",
      eventTitle: "Build failed"
    }
  ]
}

const NoopComponentProps: ComponentBaseProps = {
  title: "Don't do anything",
  iconSlug: "circle-off",
  headerColor: "bg-gray-50",
  eventSections: [
    {
      title: "Last Event",
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully"
    }
  ]
}

const SwitchComponentProps: ComponentBaseProps = {
  title: "Branch processed events",
  iconSlug: "git-branch",
  headerColor: "bg-gray-50",
  spec: {
    title: "path",
    tooltipTitle: "paths applied",
    values: [
      {
        badges: [
          { label: "MAIN", bgColor: "bg-gray-500", textColor: "text-white" },
          { label: "$.title", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "contains", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"superplane\"", bgColor: "bg-green-100", textColor: "text-green-700" },
        ]
      },
      {
        badges: [
          { label: "STAGE", bgColor: "bg-gray-500", textColor: "text-white" },
          { label: "$.author", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "contains", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"pedro\"", bgColor: "bg-green-100", textColor: "text-green-700" },
        ]
      },
      {
        badges: [
          { label: "DEV", bgColor: "bg-gray-500", textColor: "text-white" },
          { label: "$.branch", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "is", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "\"dev\"", bgColor: "bg-green-100", textColor: "text-green-700" },
        ]
      }
    ]
  },
  eventSections: [
    {
      title: "MAIN",
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "fix: Branch name contains 'superplane'"
    },
    {
      title: "STAGE",
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "feature: Branch name contains 'dev'"
    },
    {
      title: "DEV",
      receivedAt: new Date(),
      eventState: "failed",
      eventTitle: "Build failed"
    }
  ]
}

const meta: Meta<typeof ComponentBase> = {
  title: 'ui/ComponentBase',
  component: ComponentBase,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Filter: Story = {
  args: FilterComponentProps,
};

export const If: Story = {
  args: IfComponentProps,
};

export const Switch: Story = {
  args: SwitchComponentProps,
};

export const Noop: Story = {
  args: NoopComponentProps,
};
