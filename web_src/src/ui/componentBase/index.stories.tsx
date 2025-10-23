import type { Meta, StoryObj } from '@storybook/react';
import { ComponentBase, type ComponentBaseProps } from './';
import githubIcon from '@/assets/icons/integrations/github.svg';
import dockerIcon from '@/assets/icons/integrations/docker.svg';

const BasicProps: ComponentBaseProps = {
  title: "GitHub Action",
  iconSrc: githubIcon,
  iconBackground: "bg-black",
  headerColor: "bg-gray-100",
  description: "Build and test workflow",
  spec: {
    title: "filter",
    values: [
      {
        badges: [
          { label: "branch=main", bgColor: "bg-blue-500", textColor: "text-blue-600" },
          { label: "event=push", bgColor: "bg-green-500", textColor: "text-green-600" }
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

const DockerProps: ComponentBaseProps = {
  title: "DockerHub Image",
  iconSrc: dockerIcon,
  headerColor: "bg-sky-100",
  description: "Container registry trigger",
  spec: {
    title: "filter",
    values: [
      {
        badges: [
          { label: "tag=latest", bgColor: "bg-purple-500", textColor: "text-purple-600" },
          { label: "push", bgColor: "bg-orange-500", textColor: "text-orange-600" }
        ]
      },
      {
        badges: [
          { label: "tag=v1.0", bgColor: "bg-blue-500", textColor: "text-blue-600" }
        ]
      }
    ]
  },
  eventSections: [
    {
      title: "Last Event",
      receivedAt: new Date(Date.now() - 5 * 60 * 1000), // 5 minutes ago
      eventState: "failed",
      eventTitle: "Image build failed"
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

export const Basic: Story = {
  args: BasicProps,
};

export const Docker: Story = {
  args: DockerProps,
};

export const WithoutSpec: Story = {
  args: {
    title: "Simple Component",
    headerColor: "bg-purple-100",
    description: "A component without specifications",
    eventSections: [
      {
        title: "Status",
        receivedAt: new Date(),
        eventState: "success",
        eventTitle: "Ready"
      }
    ]
  },
};

export const MultipleEvents: Story = {
  args: {
    title: "Multi-Event Component",
    iconSlug: "webhook",
    iconColor: "text-blue-600",
    headerColor: "bg-blue-100",
    description: "Component with multiple event sections",
    spec: {
      title: "condition",
      values: [
        {
          badges: [
            { label: "active", bgColor: "bg-green-500", textColor: "text-green-600" },
            { label: "enabled", bgColor: "bg-blue-500", textColor: "text-blue-600" }
          ]
        }
      ]
    },
    eventSections: [
      {
        title: "Build",
        receivedAt: new Date(Date.now() - 2 * 60 * 1000),
        eventState: "success",
        eventTitle: "Build passed"
      },
      {
        title: "Deploy",
        receivedAt: new Date(Date.now() - 1 * 60 * 1000),
        eventState: "success",
        eventTitle: "Deployed to production"
      }
    ]
  },
};

export const Collapsed: Story = {
  args: {
    ...BasicProps,
    collapsed: true,
    collapsedBackground: "bg-black",
  },
};

export const CollapsedDocker: Story = {
  args: {
    ...DockerProps,
    collapsed: true,
    collapsedBackground: "bg-sky-100",
  },
};