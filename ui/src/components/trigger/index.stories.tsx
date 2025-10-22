import type { Meta, StoryObj } from '@storybook/react';
import { Trigger, type TriggerProps } from './';
import githubIcon from '@/assets/icons/integrations/github.svg';
import dockerIcon from '@/assets/icons/integrations/docker.svg';

const GithubProps: TriggerProps = {
  title: "GitHub",
  iconSrc: githubIcon,
  iconBackground: "bg-black",
  headerColor: "bg-gray-100",
  metadata: [
    {
      icon: "book",
      label: "monarch-app",
    },
    {
      icon: "filter",
      label: "branch=main",
    },
  ],
  lastEventData: {
    title: "refactor: update README.md",
    sizeInMB: 1,
    receivedAt: new Date(),
    state: "processed",
  },
}

const meta: Meta<typeof Trigger> = {
  title: 'Components/Trigger',
  component: Trigger,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

const DockerHubProps: TriggerProps = {
  title: "DockerHub",
  iconSrc: dockerIcon,
  headerColor: "bg-sky-100",
  metadata: [
    {
      icon: "box",
      label: "monarch-app-base-image",
    },
    {
      icon: "filter",
      label: "push",
    },
  ],
  lastEventData: {
    title: "v3.18.217",
    sizeInMB: 972.5,
    receivedAt: new Date(),
    state: "processed",
  },
}

export const GitHub: Story = {
  args: GithubProps,
};

export const DockerHub: Story = {
  args: DockerHubProps,
};

export const GitHubCollapsed: Story = {
  args: {
    ...GithubProps,
    collapsed: true,
    collapsedBackground: "bg-black",
  },
};

export const DockerHubCollapsed: Story = {
  args: {
    ...DockerHubProps,
    collapsed: true,
    collapsedBackground: "bg-sky-100",
  },
};