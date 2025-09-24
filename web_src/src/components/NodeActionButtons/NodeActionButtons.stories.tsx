import type { Meta, StoryObj } from '@storybook/react';
import { NodeActionButtons } from './NodeActionButtons';

const meta: Meta<typeof NodeActionButtons> = {
  title: 'Components/NodeActionButtons',
  component: NodeActionButtons,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    entityType: {
      control: 'text',
      description: 'The type of entity (e.g., "stage", "event source", "connection group")'
    },
    isEditMode: {
      control: 'boolean',
      description: 'Whether the component is in edit mode or not'
    },
    isNewNode: {
      control: 'boolean',
      description: 'Whether this is a new node being created'
    },
  },
  args: {
    onSave: () => {},
    onCancel: () => {},
    onDiscard: () => {},
    onEdit: () => {},
    onDuplicate: () => {},
    onSend: () => {},
    onSelect: () => {},
    onYamlApply: () => {},
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Edit mode story
export const EditMode: Story = {
  args: {
    isEditMode: true,
    isNewNode: false,
    entityType: 'stage',
    entityData: {
      metadata: {
        name: 'Sample Stage',
        description: 'A sample stage for testing'
      },
      spec: {
        executor: { type: 'http' }
      }
    }
  },
  decorators: [
    (Story) => (
      <div style={{ height: '200px', position: 'relative', padding: '60px' }}>
        <Story />
      </div>
    ),
  ],
};

// Non-edit mode with all buttons
export const NonEditMode: Story = {
  args: {
    isEditMode: false,
    isNewNode: false,
    entityType: 'stage',
  },
  decorators: [
    (Story) => (
      <div style={{ height: '200px', position: 'relative', padding: '60px' }}>
        <Story />
      </div>
    ),
  ],
};
