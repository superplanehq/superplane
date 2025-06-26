import RunItem from '../components/RunItem';

export default {
  title: 'Components/RunItem',
  component: RunItem,
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    status: {
      control: { type: 'select' },
      options: ['passed', 'failed', 'queued', 'running'],
      description: 'Run status',
    },
    commitTitle: {
      control: 'text',
      description: 'Commit title display',
    },
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
    date: {
      control: 'text',
      description: 'Date display',
    },
    needApproval: {
      control: 'boolean',
      description: 'Whether approval is needed',
    },
    isHightlighted: {
      control: 'boolean',
      description: 'Whether the item is highlighted',
    },
  },
};

export const Passed = {
  args: {
    status: 'passed',
    commitTitle: 'Add new feature',
    commitHash: 'abc123',
    imageVersion: 'v1.2.3',
    extraTags: 'production',
    timestamp: '2 hours ago',
    date: 'Jan 16, 2022',
    needApproval: false,
    isHightlighted: false,
  },
};

export const Failed = {
  args: {
    status: 'failed',
    commitTitle: 'Fix critical bug',
    commitHash: 'def456',
    imageVersion: 'v1.3.0',
    extraTags: 'staging',
    timestamp: '1 hour ago',
    date: 'Jan 16, 2022',
    needApproval: false,
    isHightlighted: false,
  },
};

export const Queued = {
  args: {
    status: 'queued',
    commitTitle: 'Update dependencies',
    commitHash: 'ghi789',
    imageVersion: 'v1.4.0',
    extraTags: 'development',
    timestamp: '30 minutes ago',
    date: 'Jan 16, 2022',
    needApproval: true,
    isHightlighted: false,
  },
};

export const Running = {
  args: {
    status: 'running',
    commitTitle: 'Refactor component',
    commitHash: 'jkl012',
    imageVersion: 'v1.1.0',
    extraTags: 'testing',
    timestamp: '5 minutes ago',
    date: 'Jan 16, 2022',
    needApproval: false,
    isHightlighted: false,
  },
};

export const HighlightedPassed = {
  args: {
    status: 'passed',
    commitTitle: 'Deploy to production',
    commitHash: 'mno345',
    imageVersion: 'v2.0.0',
    extraTags: 'production',
    timestamp: '10 minutes ago',
    date: 'Jan 16, 2022',
    needApproval: false,
    isHightlighted: true,
  },
};

export const HighlightedFailed = {
  args: {
    status: 'failed',
    commitTitle: 'Hot fix deployment',
    commitHash: 'pqr678',
    imageVersion: 'v2.0.1',
    extraTags: 'hotfix',
    timestamp: '15 minutes ago',
    date: 'Jan 16, 2022',
    needApproval: false,
    isHightlighted: true,
  },
};