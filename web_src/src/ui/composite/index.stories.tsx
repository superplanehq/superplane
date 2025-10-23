import type { Meta, StoryObj } from '@storybook/react';
import { Composite, type CompositeProps } from './';
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

const BuildTestDeployStage: CompositeProps = {
  title: "Build/Test/Deploy Stage",
  description: "Build new release of the monarch app and runs all required tests",
  iconSlug: "git-branch",
  iconColor: "text-purple-700",
  headerColor: "bg-purple-100",
  parameters: [],
  parametersIcon: "map",
  lastRunItem: {
    title: "fix: open rejected events tabs",
    subtitle: "ef758d40",
    receivedAt: new Date(),
    childEventsInfo: {
      count: 3,
      waitingInfos: [],
    },
    state: "failed",
    values: {
      "Author": "Bart Willems",
      "Commit": "FEAT-1234",
      "Sha": "ef758d40",
      "Image": "v3.18.217",
      "Size": "971.5 MB"
    },
  },
  nextInQueue: {
    title: "FEAT-1234: New feature",
    subtitle: "ef758d40",
    receivedAt: new Date(),
  },
  collapsed: false
}

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

const DeployToUS: CompositeProps = {
  title: "Deploy to US",
  iconSrc: KubernetesIcon,
  headerColor: "bg-blue-100",
  iconBackground: "bg-blue-500",
  parameters: ["us-west-1", "us-east-1"],
  parametersIcon: "map",
  lastRunItem: {
    title: "FEAT-984: Autocomplete",
    subtitle: "ef758d40",
    receivedAt: new Date(),
    state: "success",
    values: {
      "Author": "Bart Willems",
      "Commit": "FEAT-1234",
      "Sha": "ef758d40",
      "Image": "v3.18.217",
      "Size": "971.5 MB"
    },
  },
  nextInQueue: {
    title: "FEAT-983: Better run names",
    subtitle: "ef758d40",
    receivedAt: new Date(),
  },
  startLastValuesOpen: true,
  collapsed: false
}

const DeployToAsia: CompositeProps = {
  title: "Deploy to Asia",
  iconSrc: KubernetesIcon,
  headerColor: "bg-blue-100",
  iconBackground: "bg-blue-500",
  parameters: ["asia-east-1"],
  parametersIcon: "map",
  lastRunItem: {
    title: "fix: open rejected events",
    subtitle: "ef758d40",
    receivedAt: new Date(),
    state: "success",
    values: {
      "Author": "Bart Willems",
      "Commit": "FEAT-1234",
      "Sha": "ef758d40",
      "Image": "v3.18.217",
      "Size": "971.5 MB"
    },
  },
  startLastValuesOpen: false,
  collapsed: false
}

const meta: Meta<typeof Composite> = {
  title: 'ui/Composite',
  component: Composite,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const BuildTestDeployStageExpanded: Story = {
  args: BuildTestDeployStage,
};

export const DeployToEUExpanded: Story = {
  args: DeployToEu,
};

export const DeployToUSExpanded: Story = {
  args: DeployToUS,
};

export const DeployToAsiaExpanded: Story = {
  args: DeployToAsia,
};

export const DeployToEUCollapsed: Story = {
  args: {
    ...DeployToEu,
    collapsed: true,
    collapsedBackground: "bg-blue-500",
  },
};