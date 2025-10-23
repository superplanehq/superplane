import type { Meta, StoryObj } from '@storybook/react';
import { ComponentHeader, type ComponentHeaderProps } from './';

const triggerHeader: ComponentHeaderProps = {
  title: "Deploy to Production",
  description: "Deploy your application to the production environment",
  iconSlug: "rocket",
  iconColor: "text-green-700",
  headerColor: "bg-green-100",
};

const approvalHeader: ComponentHeaderProps = {
  title: "Approve Release",
  description: "New releases are deployed to staging for testing and require approvals.",
  iconSlug: "hand",
  iconColor: "text-orange-500",
  headerColor: "bg-orange-100",
};

const compositeHeader: ComponentHeaderProps = {
  title: "Build/Test/Deploy Stage",
  description: "Build new release of the monarch app and runs all required tests",
  iconSlug: "git-branch",
  iconColor: "text-purple-700",
  headerColor: "bg-purple-100",
};

const headerWithImage: ComponentHeaderProps = {
  title: "Kubernetes Deployment",
  description: "Deploy to Kubernetes cluster",
  iconSrc: "https://cdn.jsdelivr.net/gh/devicons/devicon/icons/kubernetes/kubernetes-plain.svg",
  iconBackground: "bg-blue-500",
  headerColor: "bg-blue-100",
};

const meta: Meta<typeof ComponentHeader> = {
  title: 'Components/ComponentHeader',
  component: ComponentHeader,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const TriggerStyle: Story = {
  args: triggerHeader,
};

export const ApprovalStyle: Story = {
  args: approvalHeader,
};

export const CompositeStyle: Story = {
  args: compositeHeader,
};

export const WithImageIcon: Story = {
  args: headerWithImage,
};

export const WithoutDescription: Story = {
  args: {
    title: "Simple Header",
    iconSlug: "settings",
    iconColor: "text-gray-700",
    headerColor: "bg-gray-100",
  },
};