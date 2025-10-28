import type { Meta, StoryObj } from '@storybook/react';
import { SidebarActionsDropdown } from './SidebarActionsDropdown';
import { useState } from 'react';

const meta: Meta<typeof SidebarActionsDropdown> = {
  title: 'ui/SidebarActionsDropdown',
  component: SidebarActionsDropdown,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
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
    onRun: () => alert("Run action triggered!"),
    onDuplicate: () => alert("Duplicate action triggered!"),
    onDocs: () => alert("Documentation action triggered!"),
    onDeactivate: () => alert("Deactivate action triggered!"),
    onDelete: () => alert("Delete action triggered!"),
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
  render: (args) => {
    const [isCompactView, setIsCompactView] = useState(false);
    const [isActive, setIsActive] = useState(true);


    return (
      <div className="p-8 bg-gray-50 rounded-lg">
        <div className="flex items-center justify-between gap-4 mb-4">
          <div>
            <h3 className="text-lg font-semibold">Interactive Component</h3>
            <p className="text-sm text-gray-600">
              Status: {isActive ? 'Active' : 'Inactive'}
            </p>
          </div>
          <SidebarActionsDropdown
            onRun={() => {
              console.log("Run action triggered");
              alert("Component started running!");
            }}
            onDeactivate={isActive ? () => {
              setIsActive(!isActive);
              console.log(`Component ${isActive ? 'deactivated' : 'activated'}`);
            } : undefined}
            onDocs={() => {
              console.log("Documentation action triggered");
              alert("Opening documentation!");
            }}
            onDuplicate={() => {
              console.log("Duplicate action triggered");
              alert("Component duplicated!");
            }}
            onDelete={() => {
              console.log("Delete action triggered");
              if (confirm("Are you sure you want to delete this component?")) {
                alert("Component deleted!");
              }
            }}
          />
        </div>
        <div className="text-sm text-gray-500">
          Try the different actions to see the interactive behavior.
        </div>
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
    onDeactivate: () => console.log("Deactivate service"),
    onDelete: () => console.log("Delete service"),
  },
};