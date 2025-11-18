import type { Meta, StoryObj } from "@storybook/react";
import { Semaphore } from "./";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

const meta: Meta<typeof Semaphore> = {
  title: "ui/Semaphore",
  component: Semaphore,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Success: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    integration: "semaphore-prod",
    metadata: [
      { icon: "folder", label: "my-microservice" },
      { icon: "git-branch", label: "main" },
      { icon: "file-code", label: ".semaphore/deployment.yml" },
    ],
    parameters: [
      { name: "ENVIRONMENT", value: "production" },
      { name: "REGION", value: "us-west-2" },
      { name: "VERSION", value: "v1.2.3" },
    ],
    lastExecution: {
      workflowId: "abc-123-def-456",
      receivedAt: new Date(Date.now() - 60000), // 1 minute ago
      state: "success",
    },
  },
};

export const Failed: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    metadata: [
      { icon: "folder", label: "my-microservice" },
      { icon: "git-branch", label: "feature/new-deploy" },
      { icon: "file-code", label: ".semaphore/ci.yml" },
    ],
    parameters: [{ name: "ENVIRONMENT", value: "staging" }],
    lastExecution: {
      workflowId: "xyz-789-ghi-012",
      receivedAt: new Date(Date.now() - 120000), // 2 minutes ago
      state: "failed",
    },
  },
};

export const Running: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    metadata: [
      { icon: "folder", label: "backend-api" },
      { icon: "git-branch", label: "develop" },
      { icon: "file-code", label: ".semaphore/pipeline.yml" },
    ],
    parameters: [
      { name: "BUILD_TYPE", value: "debug" },
      { name: "RUN_TESTS", value: "true" },
    ],
    lastExecution: {
      receivedAt: new Date(),
      state: "running",
    },
  },
};

export const NoExecution: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    metadata: [
      { icon: "folder", label: "frontend-app" },
      { icon: "git-branch", label: "main" },
      { icon: "file-code", label: ".semaphore/build.yml" },
    ],
  },
};

export const NoParameters: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    metadata: [
      { icon: "folder", label: "data-pipeline" },
      { icon: "git-branch", label: "main" },
      { icon: "file-code", label: ".semaphore/workflow.yml" },
    ],
    parameters: [],
    lastExecution: {
      workflowId: "def-456-ghi-789",
      receivedAt: new Date(Date.now() - 300000), // 5 minutes ago
      state: "success",
    },
  },
};

export const Collapsed: Story = {
  args: {
    iconSrc: SemaphoreLogo,
    title: "Run Semaphore Workflow",
    headerColor: "bg-gray-50",
    collapsed: true,
    collapsedBackground: "bg-gray-100",
    metadata: [
      { icon: "folder", label: "my-microservice" },
      { icon: "git-branch", label: "main" },
      { icon: "file-code", label: ".semaphore/deployment.yml" },
    ],
    parameters: [{ name: "ENVIRONMENT", value: "production" }],
    lastExecution: {
      workflowId: "abc-123-def-456",
      receivedAt: new Date(),
      state: "success",
    },
  },
};
