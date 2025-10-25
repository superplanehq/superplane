import type { Meta, StoryObj } from "@storybook/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";

import { useEffect, useState } from "react";
import { CanvasPage } from "./index";
import {
  getStorybookData,
  isInStorybook,
  navigateToStory,
} from "./storybooks/navigation";

const meta = {
  title: "Pages/CanvasPage",
  component: CanvasPage,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {},
} satisfies Meta<typeof CanvasPage>;

export default meta;

type Story = StoryObj<typeof CanvasPage>;

export const BlueprintExecutionPage: Story = {
  args: {
    nodes: [
      {
        id: "http-request",
        position: { x: 0, y: 0 },
        data: {
          label: "HTTP Request",
          state: "success",
          type: "composite",
          composite: {
            title: "HTTP Request",
            description: "Execute HTTP request for deployment",
            iconSlug: "globe",
            iconColor: "text-blue-600",
            headerColor: "bg-blue-100",
            collapsedBackground: "bg-blue-100",
            parameters: ["POST", "/api/deploy"],
            parametersIcon: "code",
            lastRunItem: {
              title: "Deploy to US West",
              subtitle: "ef758d40",
              receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
              childEventsInfo: {
                count: 3,
                state: "processed",
                waitingInfos: [],
              },
              state: "success",
              values: {
                Method: "POST",
                URL: "/api/deploy",
                Region: "us-west-1",
                Status: "200 OK",
              },
            },
            collapsed: false,
          },
        },
      },
      {
        id: "validation",
        position: { x: 500, y: 0 },
        data: {
          label: "Validate Deployment",
          state: "working",
          type: "composite",
          composite: {
            title: "Validate Deployment",
            description: "Run validation checks on deployed services",
            iconSlug: "check-circle",
            iconColor: "text-green-600",
            headerColor: "bg-green-100",
            collapsedBackground: "bg-green-100",
            parameters: ["health-check", "smoke-test"],
            parametersIcon: "list-checks",
            lastRunItem: {
              title: "Validation Suite",
              subtitle: "ef758d40",
              receivedAt: new Date(new Date().getTime() - 1000 * 60 * 15),
              childEventsInfo: {
                count: 5,
                state: "running",
                waitingInfos: [
                  {
                    icon: "clock",
                    info: "Health check in progress",
                    futureTimeDate: new Date(new Date().getTime() + 60000),
                  },
                ],
              },
              state: "running",
              values: {
                "Health Check": "In Progress",
                "Smoke Tests": "Pending",
                "Load Test": "Queued",
              },
            },
            collapsed: false,
          },
        },
      },
      {
        id: "cleanup",
        position: { x: 1000, y: 400 },
        data: {
          label: "Cleanup Resources",
          state: "pending",
          type: "composite",
          composite: {
            title: "Cleanup Resources",
            description: "Clean up temporary resources after deployment",
            iconSlug: "trash",
            iconColor: "text-red-600",
            headerColor: "bg-red-100",
            collapsedBackground: "bg-red-100",
            parameters: ["temp-storage", "build-cache"],
            parametersIcon: "server",
            lastRunItem: {
              title: "Resource Cleanup",
              subtitle: "ef758d40",
              receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60),
              childEventsInfo: {
                count: 2,
                state: "processed",
                waitingInfos: [],
              },
              state: "success",
              values: {
                "Temp Files": "Cleaned",
                Cache: "Cleared",
                Storage: "Released",
              },
            },
            collapsed: false,
          },
        },
      },
      {
        id: "notification",
        position: { x: 1000, y: 0 },
        data: {
          label: "Send Notifications",
          state: "failed",
          type: "composite",
          composite: {
            title: "Send Notifications",
            description: "Notify stakeholders of deployment status",
            iconSlug: "bell",
            iconColor: "text-yellow-600",
            headerColor: "bg-yellow-100",
            collapsedBackground: "bg-yellow-100",
            parameters: ["slack", "email"],
            parametersIcon: "mail",
            lastRunItem: {
              title: "Deployment Notifications",
              subtitle: "ef758d40",
              receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5),
              childEventsInfo: {
                count: 2,
                state: "failed",
                waitingInfos: [],
              },
              state: "failed",
              values: {
                Slack: "Failed",
                Email: "Sent",
                Error: "Rate limit exceeded",
              },
            },
            collapsed: false,
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-request", target: "validation" },
      { id: "e2", source: "validation", target: "cleanup" },
      { id: "e3", source: "validation", target: "notification" },
    ],
    title: "Build/Test/Deploy Stage",
    breadcrumbs: [
      {
        label: "Workflows",
      },
      {
        label: "Simple Deployment",
        onClick: () => navigateToStory("pages-canvaspage--simple-deployment"),
      },
      {
        label: "Build/Test/Deploy Stage",
        iconSlug: "git-branch",
        iconColor: "text-purple-700",
      },
    ],
  },
  render: (args) => {
    // Get data passed from SimpleDeployment story (Storybook only)
    const [executionData, setExecutionData] = useState<any>(null);

    useEffect(() => {
      const data = getStorybookData();

      if (data) {
        setExecutionData(data);
      }
    }, []);

    // Use passed data to customize the story if available
    const dynamicTitle = executionData?.title || args.title;
    const parentWorkflow = executionData?.parentWorkflow || "Simple Deployment";

    const lastBreadCump = executionData
      ? {
          label: dynamicTitle,
          iconSrc: executionData?.composite?.iconSrc,
          iconSlug: executionData?.composite?.iconSlug || "git-branch",
          iconColor: executionData?.composite?.iconColor || "text-purple-700",
          iconBackground: executionData?.composite?.iconBackground,
        }
      : null;

    if (lastBreadCump?.iconSrc) {
      lastBreadCump.iconSlug = "";
      lastBreadCump.iconColor = "";
    }

    // Only override breadcrumbs if we have execution data, otherwise use args.breadcrumbs
    const dynamicBreadcrumbs = executionData
      ? [
          {
            label: "Workflows",
          },
          {
            label: parentWorkflow,
            onClick: () =>
              navigateToStory("pages-canvaspage--simple-deployment"),
          },
          {
            ...lastBreadCump,
          },
        ]
      : args.breadcrumbs;

    // Create different execution nodes based on the source node
    const createDynamicNodes = () => {
      if (!executionData) return args.nodes;

      const baseNodes = [
        {
          id: "pre-deploy",
          position: { x: -400, y: -100 },
          data: {
            label: "Pre-deployment Checks",
            state: "success",
            type: "composite",
            composite: {
              title: "Pre-deployment Checks",
              description: `Validate prerequisites for ${dynamicTitle}`,
              iconSlug: "shield-check",
              iconColor: "text-blue-600",
              headerColor: "bg-blue-100",
              collapsedBackground: "bg-blue-100",
              parameters: ["health", "permissions", "resources"],
              parametersIcon: "check",
              collapsed: false,
              lastRunItem: {
                title: "Pre-deployment Checks",
                subtitle: "default",
                receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
                state: "success",
                values: {},
              },
            },
          },
        },
        {
          id: "deploy-action",
          position: { x: 100, y: -100 },
          data: {
            label: `Deploy ${dynamicTitle.replace("Deploy to ", "")}`,
            state: "working",
            type: "composite",
            composite: {
              title: `Deploy ${dynamicTitle.replace("Deploy to ", "")}`,
              description: `Execute deployment to ${dynamicTitle.replace(
                "Deploy to ",
                ""
              )} region`,
              iconSrc: executionData?.composite?.iconSrc,
              iconSlug: executionData?.composite?.iconSlug,
              iconColor: executionData?.composite?.iconColor,
              headerColor:
                executionData?.composite?.headerColor || "bg-blue-100",
              iconBackground: executionData?.composite?.iconBackground,
              collapsedBackground:
                executionData?.composite?.collapsedBackground,
              parameters: executionData?.composite?.parameters || [],
              parametersIcon: "map",
              lastRunItem: executionData?.composite?.lastRunItem || {
                title: `Deploy ${dynamicTitle.replace("Deploy to ", "")}`,
                subtitle: "default",
                receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
                state: "pending",
                values: {},
              },
              collapsed: false,
            },
          },
        },
        {
          id: "post-deploy",
          position: { x: 600, y: -100 },
          data: {
            label: "Post-deployment Verification",
            state: "pending",
            type: "composite",
            composite: {
              title: "Post-deployment Verification",
              description: `Verify deployment success for ${dynamicTitle}`,
              iconSlug: "check-circle",
              iconColor: "text-green-600",
              headerColor: "bg-green-100",
              collapsedBackground: "bg-green-100",
              parameters: ["health-check", "smoke-test", "monitoring"],
              parametersIcon: "activity",
              collapsed: false,
              lastRunItem: {
                title: "Post-deployment Verification",
                subtitle: "default",
                receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
                state: "pending",
                values: {},
              },
            },
          },
        },
      ];

      return baseNodes;
    };

    const dynamicEdges = executionData
      ? [
          { id: "e1", source: "pre-deploy", target: "deploy-action" },
          { id: "e2", source: "deploy-action", target: "post-deploy" },
        ]
      : args.edges;

    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage
          {...args}
          nodes={createDynamicNodes()}
          edges={dynamicEdges}
          title={dynamicTitle}
          breadcrumbs={dynamicBreadcrumbs}
        />
        {/* Debug info for Storybook (only visible in development) */}
        {isInStorybook() && executionData && (
          <div className="absolute top-16 right-4 z-30 bg-black/80 text-white p-3 rounded text-xs max-w-md">
            <div className="font-bold mb-2">
              📊 Execution Data (Storybook Only)
            </div>
            <div>From: {executionData.parentWorkflow}</div>
            <div>Node: {executionData.nodeId}</div>
            <div>Title: {executionData.title}</div>
            <div>
              Timestamp:{" "}
              {new Date(executionData.timestamp).toLocaleTimeString()}
            </div>
          </div>
        )}
      </div>
    );
  },
};
