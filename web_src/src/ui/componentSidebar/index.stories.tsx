import type { Meta, StoryObj } from '@storybook/react';
import { ComponentSidebar } from './';
import GithubIcon from "@/assets/icons/integrations/github.svg"

const meta: Meta<typeof ComponentSidebar> = {
  title: 'ui/ComponentSidebar',
  component: ComponentSidebar,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

const mockMetadata = [
  {
    icon: "book",
    label: "monarch-app",
  },
  {
    icon: "filter",
    label: "branch=main",
  },
];

export const Default: Story = {
  args: {
    metadata: mockMetadata,
    title: "Listen to code changes",
    iconSrc: GithubIcon,
    iconBackground: "bg-black",
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const WithDifferentIcon: Story = {
  args: {
    metadata: [
      {
        icon: "database",
        label: "api-service",
      },
      {
        icon: "filter",
        label: "env=production",
      },
    ],
    title: "Database Changes",
    iconSlug: "database",
    iconColor: "text-blue-500",
    iconBackground: "bg-blue-200",
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const MinimalMetadata: Story = {
  args: {
    metadata: [
      {
        icon: "code",
        label: "simple-app",
      },
    ],
    title: "Code Updates",
    iconSlug: "code",
    iconColor: "text-green-500",
    iconBackground: "bg-green-200",
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};

export const ExtendedMetadata: Story = {
  args: {
    metadata: [
      {
        icon: "book",
        label: "large-enterprise-app",
      },
      {
        icon: "filter",
        label: "branch=main",
      },
      {
        icon: "tag",
        label: "v2.1.0",
      },
      {
        icon: "users",
        label: "team=backend",
      },
    ],
    title: "Enterprise Application Monitoring",
    iconSlug: "github",
    iconColor: "#7c3aed",
    iconBackground: "#ede9fe",
    onExpandChildEvents: () => console.log("Expand child events"),
    onReRunChildEvents: () => console.log("Re-run child events"),
  },
};