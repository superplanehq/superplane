import type { Meta, StoryObj } from '@storybook/react';
import { CollapsedComponent, type CollapsedComponentProps } from './';
import { resolveIcon } from "@/lib/utils";

const triggerCollapsed: CollapsedComponentProps = {
  title: "Deploy to Production",
  iconSlug: "rocket",
  iconColor: "text-green-700",
  collapsedBackground: "bg-green-100",
  shape: "circle",
};

const approvalCollapsed: CollapsedComponentProps = {
  title: "Approve Release",
  iconSlug: "hand",
  iconColor: "text-orange-500",
  collapsedBackground: "bg-orange-100",
};

const compositeCollapsed: CollapsedComponentProps = {
  title: "Build/Test/Deploy",
  iconSlug: "git-branch",
  iconColor: "text-purple-700",
  collapsedBackground: "bg-purple-100",
};

const withImageIcon: CollapsedComponentProps = {
  title: "Kubernetes",
  iconSrc: "https://cdn.jsdelivr.net/gh/devicons/devicon/icons/kubernetes/kubernetes-plain.svg",
  iconBackground: "bg-blue-500",
  collapsedBackground: "bg-blue-100",
};

const triggerWithMetadata: CollapsedComponentProps = {
  title: "Deploy Trigger",
  iconSlug: "rocket",
  iconColor: "text-green-700",
  collapsedBackground: "bg-green-100",
  shape: "circle",
  children: (
    <div className="flex flex-col items-center gap-1">
      {[
        { icon: "clock", label: "5min ago" },
        { icon: "database", label: "1.2GB" },
        { icon: "users", label: "3 approvals" }
      ].map((item, index) => {
        const Icon = resolveIcon(item.icon);
        return (
          <div key={index} className="flex items-center gap-1 text-xs text-gray-500">
            <Icon size={12} />
            <span>{item.label}</span>
          </div>
        );
      })}
    </div>
  ),
};

const meta: Meta<typeof CollapsedComponent> = {
  title: 'Components/CollapsedComponent',
  component: CollapsedComponent,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const TriggerStyle: Story = {
  args: triggerCollapsed,
};

export const ApprovalStyle: Story = {
  args: approvalCollapsed,
};

export const CompositeStyle: Story = {
  args: compositeCollapsed,
};

export const WithImageIcon: Story = {
  args: withImageIcon,
};

export const CircleShape: Story = {
  args: {
    ...triggerCollapsed,
    shape: "circle",
  },
};

export const RoundedShape: Story = {
  args: {
    ...approvalCollapsed,
    shape: "rounded",
  },
};

export const WithChildren: Story = {
  args: triggerWithMetadata,
};