import type { Meta, StoryObj } from '@storybook/react';
import { Semaphore } from './';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';

const meta: Meta<typeof Semaphore> = {
  title: 'ui/Semaphore',
  component: Semaphore,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Success: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    integration: "semaphore-prod",
    project: "my-microservice",
    ref: "main",
    pipelineFile: ".semaphore/deployment.yml",
    parameters: [
      { name: "ENVIRONMENT", value: "production" },
      { name: "REGION", value: "us-west-2" },
      { name: "VERSION", value: "v1.2.3" }
    ],
    lastExecution: {
      workflowId: "abc-123-def-456",
      receivedAt: new Date(Date.now() - 60000), // 1 minute ago
      state: "success",
    }
  },
};

export const Failed: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    project: "my-microservice",
    ref: "feature/new-deploy",
    pipelineFile: ".semaphore/ci.yml",
    parameters: [
      { name: "ENVIRONMENT", value: "staging" }
    ],
    lastExecution: {
      workflowId: "xyz-789-ghi-012",
      receivedAt: new Date(Date.now() - 120000), // 2 minutes ago
      state: "failed",
    }
  },
};

export const Running: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    project: "backend-api",
    ref: "develop",
    pipelineFile: ".semaphore/pipeline.yml",
    parameters: [
      { name: "BUILD_TYPE", value: "debug" },
      { name: "RUN_TESTS", value: "true" }
    ],
    lastExecution: {
      receivedAt: new Date(),
      state: "running",
    }
  },
};

export const NoExecution: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    project: "frontend-app",
    ref: "main",
    pipelineFile: ".semaphore/build.yml",
  },
};

export const NoParameters: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    project: "data-pipeline",
    ref: "main",
    pipelineFile: ".semaphore/workflow.yml",
    parameters: [],
    lastExecution: {
      workflowId: "def-456-ghi-789",
      receivedAt: new Date(Date.now() - 300000), // 5 minutes ago
      state: "success",
    }
  },
};

export const Collapsed: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    collapsed: true,
    collapsedBackground: "bg-gray-100",
    project: "my-microservice",
    ref: "main",
    pipelineFile: ".semaphore/deployment.yml",
    parameters: [
      { name: "ENVIRONMENT", value: "production" }
    ],
    lastExecution: {
      workflowId: "abc-123-def-456",
      receivedAt: new Date(),
      state: "success",
    }
  },
};
