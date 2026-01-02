import type { Meta, StoryObj } from "@storybook/react";
import { CollapsedComponent, type CollapsedComponentProps } from "./";
import { resolveIcon } from "@/lib/utils";
import React from "react";

const createCollapsedProps = (
  baseProps: Omit<CollapsedComponentProps, keyof import("../types/componentActions").ComponentActionsProps>,
): CollapsedComponentProps => ({
  ...baseProps,
  onRun: () => console.log("Run clicked!"),
  onDuplicate: () => console.log("Duplicate clicked!"),
  onEdit: () => console.log("Edit clicked!"),
  onDeactivate: () => console.log("Deactivate clicked!"),
  onToggleView: () => console.log("Toggle view clicked!"),
  onDelete: () => console.log("Delete clicked!"),
});

const triggerCollapsed: CollapsedComponentProps = createCollapsedProps({
  title: "Deploy to Production",
  iconSlug: "rocket",
  iconColor: "text-green-700",
  collapsedBackground: "bg-green-100",
  shape: "circle",
});

const approvalCollapsed: CollapsedComponentProps = createCollapsedProps({
  title: "Approve Release",
  iconSlug: "hand",
  iconColor: "text-orange-500",
  collapsedBackground: "bg-orange-100",
});

const compositeCollapsed: CollapsedComponentProps = createCollapsedProps({
  title: "Build/Test/Deploy",
  iconSlug: "git-branch",
  iconColor: "text-purple-700",
  collapsedBackground: "bg-purple-100",
});

const withImageIcon: CollapsedComponentProps = createCollapsedProps({
  title: "Kubernetes",
  iconSrc: "https://cdn.jsdelivr.net/gh/devicons/devicon/icons/kubernetes/kubernetes-plain.svg",
  iconBackground: "bg-blue-500",
  collapsedBackground: "bg-blue-100",
});

const triggerWithMetadata: CollapsedComponentProps = createCollapsedProps({
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
        { icon: "users", label: "3 approvals" },
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
});

const meta: Meta<typeof CollapsedComponent> = {
  title: "ui/CollapsedComponent",
  component: CollapsedComponent,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
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

const ToggleableCollapsedComponent = (args: CollapsedComponentProps) => {
  const [isCollapsed, setIsCollapsed] = React.useState(true);
  const [isCompactView, setIsCompactView] = React.useState(false);

  return (
    <div className="flex flex-col items-center gap-4">
      <div className="text-sm text-gray-500">
        Current state: {isCollapsed ? "Collapsed" : "Expanded"} | View: {isCompactView ? "Compact" : "Detailed"}
      </div>
      <CollapsedComponent
        {...args}
        isCompactView={isCompactView}
        onToggleView={() => {
          setIsCompactView(!isCompactView);
          console.log(`Toggled to ${!isCompactView ? "Compact" : "Detailed"} view!`);
        }}
        onRun={() => console.log("Run action triggered!")}
        onDuplicate={() => console.log("Duplicate action triggered!")}
        onEdit={() => console.log("Edit action triggered!")}
        onDeactivate={() => console.log("Deactivate action triggered!")}
        onDelete={() => console.log("Delete action triggered!")}
      />
      <button onClick={() => setIsCollapsed(!isCollapsed)} className="px-3 py-1 bg-blue-500 text-white rounded text-sm">
        {isCollapsed ? "Expand" : "Collapse"} Component
      </button>
    </div>
  );
};

export const WithToggleActions: Story = {
  render: (args) => <ToggleableCollapsedComponent {...args} />,
  args: {
    title: "Toggleable Component",
    iconSlug: "settings",
    iconColor: "text-blue-700",
    collapsedBackground: "bg-blue-100",
    shape: "rounded",
  },
};

export const WithActionsOnly: Story = {
  args: {
    ...triggerCollapsed,
    title: "Component with Actions",
    onRun: () => console.log("Run clicked!"),
    onDuplicate: () => console.log("Duplicate clicked!"),
    onEdit: () => console.log("Edit clicked!"),
    onDeactivate: () => console.log("Deactivate clicked!"),
    onToggleView: () => console.log("Toggle view clicked!"),
    onDelete: () => console.log("Delete clicked!"),
  },
};
