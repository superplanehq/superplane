import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { SidebarEventItem } from "./SidebarEventItem";
import { SidebarEvent } from "../types";
import { DEFAULT_EVENT_STATE_MAP } from "../../componentBase";

const meta: Meta<typeof SidebarEventItem> = {
  title: "UI/SidebarEventItem",
  component: SidebarEventItem,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
  argTypes: {
    variant: {
      control: "select",
      options: ["latest", "queue"],
    },
    isOpen: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const mockEvent: SidebarEvent = {
  id: "event-123",
  title: "Process Payment",
  subtitle: "Stripe API",
  state: "success" as const,
  values: {
    "Transaction ID": "txn_1234567890",
    Amount: "$99.99",
    Currency: "USD",
    Status: "completed",
  },
};

const ComponentWrapper = ({ children }: { children: React.ReactNode }) => {
  const [openEvents, setOpenEvents] = useState<string[]>([]);

  const handleToggleOpen = (eventId: string) => {
    setOpenEvents((prev) => (prev.includes(eventId) ? prev.filter((id) => id !== eventId) : [...prev, eventId]));
  };

  return (
    <div style={{ width: "400px" }}>
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child as any, {
            isOpen: openEvents.includes((child.props as any).event?.id || "event-123"),
            onToggleOpen: handleToggleOpen,
            onEventClick: (event: SidebarEvent) => console.log("Event clicked:", event),
          });
        }
        return child;
      })}
    </div>
  );
};

export const success: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: mockEvent,
    index: 0,
    variant: "latest",
    isOpen: false,
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const Discarded: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      state: "discarded" as const,
      title: "Failed Authentication",
      subtitle: "Invalid credentials",
    },
    index: 0,
    variant: "latest",
    isOpen: false,
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "failed" as const,
    }),
  },
};

export const Running: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      state: "running" as const,
      title: "Deploying Application",
      subtitle: "AWS EC2",
    },
    index: 0,
    variant: "latest",
    isOpen: false,
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "running" as const,
    }),
  },
};

export const WaitingLatest: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      state: "waiting" as const,
      title: "Pending Approval",
      subtitle: "Manual review",
    },
    index: 0,
    variant: "latest",
    isOpen: false,
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const WaitingQueue: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      state: "waiting" as const,
      title: "Queued Task",
      subtitle: "In queue",
    },
    index: 0,
    variant: "queue",
    isOpen: false,
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const WithTabData: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "Database Query",
      subtitle: "PostgreSQL",
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      current: {
        Query: "SELECT * FROM users",
        Duration: "125ms",
        Rows: "1,247",
        "Cache Hit": "true",
      },
      root: {
        Connection: "postgres://localhost:5432",
        Database: "production",
        "Pool Size": "10",
        "Active Connections": "3",
      },
      payload: {
        sql: "SELECT id, name, email FROM users WHERE active = true ORDER BY created_at DESC",
        parameters: [],
        metadata: {
          executionTime: 125,
          planningTime: 2,
          bufferHits: 1247,
          bufferMisses: 0,
        },
      },
    },
  },
};

export const WithTabDataAndChildEvents: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "Workflow Execution",
      subtitle: "Multi-step process",
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      current: {
        Status: "Running",
        Progress: "3/5 steps",
        Started: "2 minutes ago",
        ETA: "30 seconds",
      },
      root: {
        "Workflow ID": "wf_abc123def456",
        Trigger: "API Request",
        Environment: "production",
        Version: "v1.2.3",
      },
    },
  },
};

export const PayloadTabOnly: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "API Request",
      subtitle: "External service",
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      payload: {
        method: "POST",
        url: "https://api.example.com/v1/users",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer ***",
        },
        body: {
          name: "John Doe",
          email: "john@example.com",
          role: "user",
        },
        response: {
          status: 201,
          data: {
            id: 12345,
            created_at: "2024-01-15T10:30:00Z",
          },
        },
      },
    },
  },
};

export const LargePayload: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "Data Processing",
      subtitle: "Large dataset",
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      current: {
        "Records success": "10,000",
        "Success Rate": "99.8%",
        Errors: "20",
        Duration: "5m 32s",
      },
      payload: JSON.stringify(
        {
          config: {
            batchSize: 1000,
            retries: 3,
            timeout: 30000,
            validation: true,
          },
          source: {
            type: "database",
            connection: "postgres://prod-db:5432/analytics",
            query: "SELECT * FROM events WHERE created_at > ?",
          },
          destination: {
            type: "elasticsearch",
            cluster: "search-prod",
            index: "events-2024",
          },
          transformations: [
            { type: "normalize", fields: ["timestamp", "user_id"] },
            { type: "enrich", source: "user_metadata" },
            { type: "filter", condition: "status = active" },
          ],
          results: {
            totalRecords: 10000,
            successfulRecords: 9980,
            failedRecords: 20,
            errors: [
              { record: 1543, error: "Invalid timestamp format" },
              { record: 2891, error: "Missing required field: user_id" },
            ],
          },
        },
        null,
        2,
      ),
    },
  },
};

export const WithExecutionChain: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "Deployment Pipeline",
      subtitle: "Production deploy",
      state: "running" as const,
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      current: {
        Status: "Deploying",
        Progress: "60%",
        Started: "3 minutes ago",
        ETA: "2 minutes",
      },
      root: {
        "Pipeline ID": "deploy_prod_2024_001",
        Branch: "main",
        Commit: "a1b2c3d",
        Environment: "production",
      },
      executionChain: [
        { name: "Build Docker Image", state: "completed" },
        { name: "Run Unit Tests", state: "completed" },
        { name: "Security Scan", state: "completed" },
        { name: "Deploy to Staging", state: "completed" },
        { name: "Integration Tests", state: "running" },
        { name: "Deploy to Production", state: "running" },
        { name: "Health Checks", state: "running" },
        { name: "Update DNS", state: "running" },
      ],
    },
  },
};

export const WithExecutionChainFailed: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "Failed Deployment",
      subtitle: "Build failed",
      state: "discarded" as const,
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      current: {
        Status: "Failed",
        Error: "Build compilation error",
        Started: "5 minutes ago",
        "Failed at": "3 minutes ago",
      },
      root: {
        "Pipeline ID": "deploy_prod_2024_002",
        Branch: "feature/new-api",
        Commit: "x9y8z7w",
        Environment: "production",
      },
      executionChain: [
        { name: "Build Docker Image", state: "completed" },
        { name: "Run Unit Tests", state: "completed" },
        { name: "Security Scan", state: "failed" },
        { name: "Deploy to Staging", state: "running" },
        { name: "Integration Tests", state: "running" },
        { name: "Deploy to Production", state: "running" },
      ],
    },
  },
};

export const WithNestedExecutionChain: Story = {
  render: (args) => (
    <ComponentWrapper>
      <SidebarEventItem {...args} />
    </ComponentWrapper>
  ),
  args: {
    event: {
      ...mockEvent,
      title: "Complex Deployment",
      subtitle: "Multi-service deployment",
      state: "running" as const,
    },
    index: 0,
    variant: "latest",
    isOpen: true,
    tabData: {
      current: {
        Status: "Deploying Services",
        Progress: "4/6 services",
        Started: "8 minutes ago",
        ETA: "5 minutes",
      },
      root: {
        "Pipeline ID": "deploy_multi_2024_003",
        Branch: "release/v2.1.0",
        Commit: "def456gh",
        Environment: "production",
      },
      executionChain: [
        {
          name: "Preparation Phase",
          state: "completed",
          children: [
            { name: "Validate Configuration", state: "completed" },
            { name: "Check Dependencies", state: "completed" },
            { name: "Reserve Resources", state: "completed" },
          ],
        },
        {
          name: "Build Phase",
          state: "completed",
          children: [
            { name: "Build API Service", state: "completed" },
            { name: "Build Frontend", state: "completed" },
            { name: "Build Background Jobs", state: "completed" },
            { name: "Build Database Migrations", state: "completed" },
          ],
        },
        {
          name: "Test Phase",
          state: "completed",
          children: [
            { name: "Unit Tests", state: "completed" },
            { name: "Integration Tests", state: "completed" },
            { name: "Security Scan", state: "completed" },
          ],
        },
        {
          name: "Deploy to Staging",
          state: "running",
          children: [
            { name: "Deploy Database", state: "completed" },
            { name: "Deploy API Service", state: "completed" },
            { name: "Deploy Frontend", state: "running" },
            { name: "Deploy Background Jobs", state: "running" },
          ],
        },
        {
          name: "Staging Tests",
          state: "running",
          children: [
            { name: "Smoke Tests", state: "running" },
            { name: "Performance Tests", state: "running" },
            { name: "User Acceptance Tests", state: "running" },
          ],
        },
        {
          name: "Production Deployment",
          state: "running",
          children: [
            { name: "Blue-Green Switch", state: "running" },
            { name: "Health Checks", state: "running" },
            { name: "Monitor Metrics", state: "running" },
          ],
        },
      ],
    },
  },
};
