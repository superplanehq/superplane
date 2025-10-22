import type { Meta, StoryObj } from '@storybook/react';
import { Composite, type CompositeProps } from './';
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

const DeployToEu: CompositeProps = {
  title: "Deploy to EU",
  description: "Deploy your application to the EU region",
  iconSrc: KubernetesIcon,
  headerColor: "bg-blue-100",
  iconBackground: "bg-blue-500",
  parameters: ["eu-global-1", "eu-global-2"],
  parametersIcon: "map",
  lastRunItem: {
    title: "fix: open rejected events",
    subtitle: "ef758d40",
    receivedAt: new Date(),
    childEventsInfo: {
      count: 2,
      state: "running",
      waitingInfos: [
        {
          icon: "calendar",
          info: "Wait if it's weekend",
          futureTimeDate: new Date(new Date().getTime() + 200000000),
        },
        {
          icon: "calendar",
          info: "Haloween Holiday",
          futureTimeDate: new Date(new Date().getTime() + 300000000),
        },
      ],
    },
    state: "running",
    values: {
      "Author": "Bart Willems",
      "Commit": "FEAT-1234",
      "Sha": "ef758d40",
      "Image": "v3.18.217",
      "Size": "971.5 MB"
    },
  },
  nextInQueue: {
    title: "Deploy to EU",
    subtitle: "ef758d40",
    receivedAt: new Date(),
  },
  collapsed: false
}

const meta: Meta<typeof Composite> = {
  title: 'Components/Composite',
  component: Composite,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const DeployToEUExpanded: Story = {
  args: DeployToEu,
};

export const DeployToEUCollapsed: Story = {
  args: {
    ...DeployToEu,
    collapsed: true,
    collapsedBackground: "bg-blue-500",
  },
};