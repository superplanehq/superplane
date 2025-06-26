import MessageItem from '../components/MessageItem';

export default {
  title: 'Components/MessageItem',
  component: MessageItem,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    commitHash: {
      control: 'text',
      description: 'Commit hash display',
    },
    imageVersion: {
      control: 'text',
      description: 'Image version display',
    },
    extraTags: {
      control: 'text',
      description: 'Additional tags to display',
    },
    timestamp: {
      control: 'text',
      description: 'Timestamp display',
    },
    approved: {
      control: 'boolean',
      description: 'Whether the item is approved',
    },
    isDragStart: {
      control: 'boolean',
      description: 'Whether drag indicator is visible',
    },
  },
};

export const Default = {
  args: {
    commitHash: 'abc123',
    imageVersion: 'v1.2.3',
    extraTags: 'production',
    timestamp: '2 hours ago',
    approved: false,
    isDragStart: false,
  },
};

export const Approved = {
  args: {
    commitHash: 'def456',
    imageVersion: 'v1.3.0',
    extraTags: 'staging',
    timestamp: '1 hour ago',
    approved: true,
    isDragStart: false,
  },
};

export const WithDragIndicator = {
  args: {
    commitHash: 'ghi789',
    imageVersion: 'v1.4.0',
    extraTags: 'development',
    timestamp: '30 minutes ago',
    approved: false,
    isDragStart: true,
  },
};

export const WithoutExtraTags = {
  args: {
    commitHash: 'jkl012',
    imageVersion: 'v1.1.0',
    extraTags: null,
    timestamp: '3 hours ago',
    approved: false,
    isDragStart: false,
  },
};