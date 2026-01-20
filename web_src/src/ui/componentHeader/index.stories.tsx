import type { Meta, StoryObj } from "@storybook/react";
import { ComponentHeader, type ComponentHeaderProps } from "./";

const createHeaderProps = (
  baseProps: Omit<ComponentHeaderProps, keyof import("../types/componentActions").ComponentActionsProps>,
): ComponentHeaderProps => ({
  ...baseProps,
  onRun: () => console.log("Run clicked!"),
  onDuplicate: () => console.log("Duplicate clicked!"),
  onEdit: () => console.log("Edit clicked!"),
  onDeactivate: () => console.log("Deactivate clicked!"),
  onToggleView: () => console.log("Toggle view clicked!"),
  onDelete: () => console.log("Delete clicked!"),
});

const triggerHeader: ComponentHeaderProps = createHeaderProps({
  title: "Deploy to Production",
  description: "Deploy your application to the production environment",
  iconSlug: "rocket",
  iconColor: "text-green-700",
});

const approvalHeader: ComponentHeaderProps = createHeaderProps({
  title: "Approve Release",
  description: "New releases are deployed to staging for testing and require approvals.",
  iconSlug: "hand",
  iconColor: "text-orange-500",
});

const compositeHeader: ComponentHeaderProps = createHeaderProps({
  title: "Build/Test/Deploy Stage",
  description: "Build new release of the monarch app and runs all required tests",
  iconSlug: "git-branch",
  iconColor: "text-purple-700",
});

const headerWithImage: ComponentHeaderProps = createHeaderProps({
  title: "Kubernetes Deployment",
  description: "Deploy to Kubernetes cluster",
  iconSrc: "https://cdn.jsdelivr.net/gh/devicons/devicon/icons/kubernetes/kubernetes-plain.svg",
});

const meta: Meta<typeof ComponentHeader> = {
  title: "ui/ComponentHeader",
  component: ComponentHeader,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
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
  args: createHeaderProps({
    title: "Simple Header",
    iconSlug: "settings",
    iconColor: "text-gray-700",
  }),
};

export const WithActionsDropdown: Story = {
  args: {
    title: "Header with Actions",
    description: "This header includes action dropdown functionality",
    iconSlug: "cog",
    iconColor: "text-blue-700",
    onRun: () => {
      console.log("Run action triggered");
    },
    onDuplicate: () => {
      console.log("Duplicate action triggered");
    },
    onEdit: () => {
      console.log("Edit action triggered");
    },
    onDeactivate: () => {
      console.log("Deactivate action triggered");
    },
    onToggleView: () => {
      console.log("Toggle view action triggered");
    },
    onDelete: () => {
      console.log("Delete action triggered");
    },
    isCompactView: false,
  },
};
