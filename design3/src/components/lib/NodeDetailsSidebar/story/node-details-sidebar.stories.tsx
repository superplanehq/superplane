import type { Meta, StoryObj } from '@storybook/react';
import { NodeDetailsSidebar } from '../node-details-sidebar';

const meta = {
  title: 'Components/NodeDetailsSidebar',
  component: NodeDetailsSidebar,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'A sidebar component that displays detailed information about a workflow node, including recent runs, queue items, and various configuration options.',
      },
    },
  },
  argTypes: {
    nodeId: {
      control: 'text',
      description: 'Unique identifier for the node',
    },
    nodeTitle: {
      control: 'text',
      description: 'Display title for the node',
    },
    nodeIcon: {
      control: 'text',
      description: 'Material Symbol icon name for the node',
    },
    isOpen: {
      control: 'boolean',
      description: 'Whether the sidebar is open or closed',
    },
    onClose: {
      action: 'closed',
      description: 'Callback function when sidebar is closed',
    },
    className: {
      control: 'text',
      description: 'Additional CSS classes',
    },
  },
} satisfies Meta<typeof NodeDetailsSidebar>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    nodeId: 'sync-cluster-1',
    nodeTitle: 'Sync Cluster',
    nodeIcon: 'sync',
    isOpen: true,
    onClose: () => console.log('Sidebar closed'),
  },
};

export const DeploymentNode: Story = {
  args: {
    nodeId: 'deployment-1',
    nodeTitle: 'Deploy to Production',
    nodeIcon: 'rocket_launch',
    isOpen: true,
    onClose: () => console.log('Sidebar closed'),
  },
};

export const BuildNode: Story = {
  args: {
    nodeId: 'build-1',
    nodeTitle: 'Build & Test',
    nodeIcon: 'build_circle',
    isOpen: true,
    onClose: () => console.log('Sidebar closed'),
  },
};

export const Closed: Story = {
  args: {
    nodeId: 'sync-cluster-1',
    nodeTitle: 'Sync Cluster',
    nodeIcon: 'sync',
    isOpen: false,
    onClose: () => console.log('Sidebar closed'),
  },
};

export const CustomStyling: Story = {
  args: {
    nodeId: 'custom-1',
    nodeTitle: 'Custom Node',
    nodeIcon: 'settings',
    isOpen: true,
    onClose: () => console.log('Sidebar closed'),
    className: 'border-l-4 border-l-blue-500',
  },
};

export const WithNewInputsOutputsStyle: Story = {
  args: {
    nodeId: 'inputs-outputs-demo',
    nodeTitle: 'Inputs/Outputs Demo',
    nodeIcon: 'data_object',
    isOpen: true,
    onClose: () => console.log('Sidebar closed'),
  },
  parameters: {
    docs: {
      description: {
        story: 'This story demonstrates the new renderInputsOutputs2 function that displays inputs and outputs in separate bordered boxes with improved styling and copy buttons.',
      },
    },
  },
};