import type { Meta, StoryObj } from '@storybook/react';
import { MetadataList } from './';

const meta: Meta<typeof MetadataList> = {
  title: 'ui/MetadataList',
  component: MetadataList,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        icon: "book",
        label: "monarch-app",
      },
      {
        icon: "filter",
        label: "branch=main",
      },
    ],
  },
};

export const DockerHubExample: Story = {
  args: {
    items: [
      {
        icon: "box",
        label: "monarch-app-base-image",
      },
      {
        icon: "filter",
        label: "push",
      },
    ],
  },
};

export const SingleItem: Story = {
  args: {
    items: [
      {
        icon: "globe",
        label: "production environment",
      },
    ],
  },
};

export const ManyItems: Story = {
  args: {
    items: [
      {
        icon: "database",
        label: "postgres:14.5",
      },
      {
        icon: "server",
        label: "us-west-2",
      },
      {
        icon: "shield",
        label: "SSL enabled",
      },
      {
        icon: "clock",
        label: "24h retention",
      },
    ],
  },
};

export const EmptyList: Story = {
  args: {
    items: [],
  },
};

export const CustomStyling: Story = {
  args: {
    items: [
      {
        icon: "star",
        label: "premium feature",
      },
      {
        icon: "zap",
        label: "high performance",
      },
    ],
    className: "px-4 py-2 bg-blue-50 border border-blue-200 rounded-lg text-blue-700 flex flex-col gap-3",
    iconSize: 16,
  },
};