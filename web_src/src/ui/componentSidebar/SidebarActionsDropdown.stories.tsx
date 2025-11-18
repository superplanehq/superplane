import type { Meta, StoryObj } from "@storybook/react";
import { SidebarActionsDropdown } from "./SidebarActionsDropdown";
import { useState } from "react";

const meta: Meta<typeof SidebarActionsDropdown> = {
  title: "ui/SidebarActionsDropdown",
  component: SidebarActionsDropdown,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => {
    return (
      <div className="p-8 bg-gray-50 rounded-lg">
        <div className="flex items-center gap-4">
          <h3 className="text-lg font-semibold">Component Actions</h3>
          <SidebarActionsDropdown {...args} />
        </div>
      </div>
    );
  },
  args: {
    onRun: () => console.log("Run action triggered"),
    onDuplicate: () => console.log("Duplicate action triggered"),
    onDocs: () => console.log("Documentation action triggered"),
    onDeactivate: () => console.log("Deactivate action triggered"),
    onToggleView: () => console.log("Toggle view action triggered"),
    onDelete: () => console.log("Delete action triggered"),
    isCompactView: false,
  },
};

export const LimitedActions: Story = {
  render: (args) => {
    return (
      <div className="p-8 bg-gray-50 rounded-lg">
        <div className="flex items-center gap-4">
          <h3 className="text-lg font-semibold">Limited Actions</h3>
          <SidebarActionsDropdown {...args} />
        </div>
      </div>
    );
  },
  args: {
    onRun: () => console.log("Run action"),
    onDocs: () => console.log("Documentation action"),
    onToggleView: () => console.log("Toggle view action"),
    onDelete: () => console.log("Delete action"),
  },
};

export const NoActions: Story = {
  render: (args) => {
    return (
      <div className="p-8 bg-gray-50 rounded-lg">
        <div className="flex items-center gap-4">
          <h3 className="text-lg font-semibold">No Actions Available</h3>
          <SidebarActionsDropdown {...args} />
        </div>
        <p className="text-sm text-gray-500 mt-2">
          When no actions have onAction handlers, the dropdown doesn't render
        </p>
      </div>
    );
  },
  args: {
    // No action handlers provided
  },
};

export const InteractiveToggle: Story = {
  render: () => {
    const [isActive, setIsActive] = useState(true);

    return (
      <div className="p-8 bg-gray-50 rounded-lg">
        <div className="flex items-center justify-between gap-4 mb-4">
          <div>
            <h3 className="text-lg font-semibold">Interactive Component</h3>
            <p className="text-sm text-gray-600">Status: {isActive ? "Active" : "Inactive"}</p>
          </div>
          <SidebarActionsDropdown
            onRun={() => {
              console.log("Run action triggered");
            }}
            onDeactivate={
              isActive
                ? () => {
                    setIsActive(!isActive);
                    console.log(`Component ${isActive ? "deactivated" : "activated"}`);
                  }
                : undefined
            }
            onDocs={() => {
              console.log("Documentation action triggered");
            }}
            onToggleView={() => {
              console.log("Toggle view action triggered");
            }}
            onDuplicate={() => {
              console.log("Duplicate action triggered");
            }}
            onDelete={() => {
              console.log("Delete action triggered");
            }}
          />
        </div>
        <div className="text-sm text-gray-500">Try the different actions to see the interactive behavior.</div>
      </div>
    );
  },
  args: {},
};

export const InContext: Story = {
  render: (args) => {
    return (
      <div className="w-80 bg-white border rounded-lg shadow-sm">
        <div className="flex items-center justify-between gap-3 p-4 border-b bg-gray-50">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center text-white font-semibold">
              A
            </div>
            <h2 className="text-lg font-semibold">API Service</h2>
          </div>
          <SidebarActionsDropdown {...args} />
        </div>
        <div className="p-4">
          <p className="text-sm text-gray-600">
            This shows how the dropdown looks in a typical sidebar header context.
          </p>
        </div>
      </div>
    );
  },
  args: {
    onRun: () => console.log("Run service"),
    onDuplicate: () => console.log("Duplicate service"),
    onDocs: () => console.log("Documentation service"),
    onToggleView: () => console.log("Toggle view service"),
    onDeactivate: () => console.log("Deactivate service"),
    onDelete: () => console.log("Delete service"),
  },
};
