import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";

import dockerIcon from "@/assets/icons/integrations/docker.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

import { useCallback, useEffect, useMemo, useState } from "react";
import { LastRunItem } from "../composite";
import type { BreadcrumbItem } from "./Header";
import { CanvasPage } from "./index";
import { useSimulationRunner } from "./storybooks/useSimulation";

// Storybook-specific utility functions for data passing
const isInStorybook = () => {
  return (
    typeof window !== "undefined" &&
    (window.location.pathname.includes("storybook") ||
      window.location.search.includes("path=/story/") ||
      (window.parent !== window &&
        window.parent.location.pathname.includes("storybook")))
  );
};

const navigateToStoryWithData = (storyId: string, data?: any) => {
  try {
    // Use parent window location for Storybook iframe navigation
    const targetWindow = window.parent !== window ? window.parent : window;
    let newUrl = `${targetWindow.location.origin}${targetWindow.location.pathname}?path=/story/${storyId}`;

    // Add data as query parameters for Storybook
    if (data) {
      const encodedData = encodeURIComponent(JSON.stringify(data));
      newUrl += `&args=nodeData:${encodedData}`;
    }

    // Navigate using the correct window
    targetWindow.location.href = newUrl;
  } catch (error) {
    console.error("❌ Navigation failed:", error);
    // Ultimate fallback - try direct URL construction
    try {
      const fallbackUrl = `${window.location.protocol}//${window.location.host}${window.location.pathname}?path=/story/${storyId}`;
      if (window.top?.location) {
        window.top.location.href = fallbackUrl;
      }
    } catch (fallbackError) {
      console.error("❌ Fallback also failed:", fallbackError);
    }
  }
};

const getStorybookData = () => {
  if (typeof window === "undefined") return null;

  try {
    const urlParams = new URLSearchParams(window.location.search);

    const args = urlParams.get("args");

    if (args) {
      // Parse args string like "nodeData:encodedJson"
      const nodeDataMatch = args.match(/nodeData:([^&]+)/);

      if (nodeDataMatch) {
        const decodedData = decodeURIComponent(nodeDataMatch[1]);
        const parsedData = JSON.parse(decodedData);
        return parsedData;
      }
    }
  } catch (error) {
    console.error("❌ Failed to parse Storybook data:", error);
  }

  return null;
};

// Legacy navigation function for simple navigation
const navigateToStory = (storyId: string) => {
  navigateToStoryWithData(storyId);
};

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

const sampleNodes: Node[] = [
  {
    id: "listen-code",
    position: { x: -500, y: -200 },
    data: {
      label: "Listen to code changes",
      state: "working",
      type: "trigger",
      trigger: {
        title: "GitHub",
        iconSrc: githubIcon,
        iconBackground: "bg-black",
        headerColor: "bg-gray-100",
        collapsedBackground: "bg-black",
        metadata: [
          { icon: "book", label: "monarch-app" },
          { icon: "filter", label: "branch=main" },
        ],
        lastEventData: {
          title: "refactor: update README.md",
          subtitle: "ef53adfa",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "listen-image",
    position: { x: -500, y: 200 },
    data: {
      label: "Listen to Docker image updates",
      state: "pending",
      type: "trigger",
      trigger: {
        title: "DockerHub",
        iconSrc: dockerIcon,
        headerColor: "bg-sky-100",
        collapsedBackground: "bg-sky-100",
        metadata: [
          { icon: "box", label: "monarch-app-base-image" },
          { icon: "filter", label: "push" },
        ],
        lastEventData: {
          title: "v3.18.217",
          subtitle: "972.5 MB",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 3), // 3 hours ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "build-stage",
    position: { x: 0, y: 0 },
    data: {
      label: "Build/Test/Deploy to Stage",
      state: "pending",
      type: "composite",
      composite: {
        title: "Build/Test/Deploy Stage",
        description:
          "Build new release of the monarch app and runs all required tests",
        iconSlug: "git-branch",
        iconColor: "text-purple-700",
        headerColor: "bg-purple-100",
        collapsedBackground: "bg-purple-100",
        parameters: [],
        parametersIcon: "map",
        lastRunItem: {
          title: "fix: open rejected events tabs",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 2), // 2 hours ago
          childEventsInfo: {
            count: 3,
            waitingInfos: [],
          },
          state: "failed",
          values: {
            Author: "Bart Willems",
            Commit: "FEAT-1234",
            Sha: "ef758d40",
            Image: "v3.18.217",
            Size: "971.5 MB",
          },
        },
        nextInQueue: {
          title: "FEAT-1234: New feature",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30), // 30 minutes ago
        },
        collapsed: true,
      },
    },
  },
  {
    id: "approve",
    position: { x: 500, y: 0 },
    data: {
      label: "Approve release",
      state: "pending",
      type: "approval",
      approval: {
        title: "Approve Release",
        description:
          "New releases are deployed to staging for testing and require approvals.",
        iconSlug: "hand",
        iconColor: "text-orange-500",
        headerColor: "bg-orange-100",
        collapsedBackground: "bg-orange-100",
        approvals: [
          {
            title: "Security",
            approved: false,
            interactive: true,
            requireArtifacts: [
              {
                label: "CVE Report",
              },
            ],
            onApprove: (artifacts) =>
              console.log("Security approved with artifacts:", artifacts),
            onReject: (comment) =>
              console.log("Security rejected with comment:", comment),
          },
          {
            title: "Compliance",
            approved: true,
            artifactCount: 1,
            artifacts: {
              "Security Audit Report": "https://example.com/audit-report.pdf",
              "Compliance Certificate": "https://example.com/cert.pdf",
            },
            href: "#",
          },
          {
            title: "Engineering",
            rejected: true,
            approverName: "Lucas Pinheiro",
            rejectionComment:
              "Security vulnerabilities need to be addressed before approval",
          },
          {
            title: "Josh Brown",
            approved: true,
          },
          {
            title: "Admin",
            approved: false,
          },
        ],
        awaitingEvent: {
          title: "fix: open rejected events tab",
          subtitle: "ef758d40",
        },
        receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 24),
        collapsed: false,
      },
    },
  },
  {
    id: "deploy-us",
    position: { x: 1250, y: -600 },
    data: {
      label: "Deploy to US",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deploy to US",
        iconSrc: KubernetesIcon,
        headerColor: "bg-blue-100",
        iconBackground: "bg-blue-500",
        collapsedBackground: "bg-blue-500",
        metadata: [
          { icon: "user", label: "Author: Bart Willems" },
          { icon: "git-commit", label: "Commit: FEAT-1234" },
          { icon: "git-commit", label: "Sha: ef758d40" },
          { icon: "package", label: "Image: v3.18.217" },
          { icon: "package", label: "Size: 971.5 MB" },
        ],
        parameters: ["us-west-1", "us-east-1"],
        parametersIcon: "map",
        lastRunItem: {
          title: "FEAT-984: Autocomplete",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 5), // 5 hours ago
          childEventsInfo: {
            count: 2,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            Author: "Bart Willems",
            Commit: "FEAT-1234",
            Sha: "ef758d40",
            Image: "v3.18.217",
            Size: "971.5 MB",
          },
        },
        nextInQueue: {
          title: "FEAT-983: Better run names",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60), // 1 hour ago
        },
        collapsed: false,
      },
    },
  },
  {
    id: "deploy-eu",
    position: { x: 1250, y: 0 },
    data: {
      label: "Deploy to EU",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deploy to EU",
        description: "Deploy your application to the EU region",
        iconSrc: KubernetesIcon,
        headerColor: "bg-blue-100",
        iconBackground: "bg-blue-500",
        collapsedBackground: "bg-blue-500",
        metadata: [
          { icon: "user", label: "Author: Bart Willems" },
          { icon: "git-commit", label: "Commit: FEAT-1234" },
          { icon: "git-commit", label: "Sha: ef758d40" },
          { icon: "package", label: "Image: v3.18.217" },
          { icon: "package", label: "Size: 971.5 MB" },
        ],
        parameters: ["eu-global-1", "eu-global-2"],
        parametersIcon: "map",
        lastRunItem: {
          title: "fix: open rejected events",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 8), // 8 hours ago
          childEventsInfo: {
            count: 2,
            state: "running",
            waitingInfos: [
              {
                icon: "calendar",
                info: "Wait if it's weekend",
                futureTimeDate: new Date(new Date().getTime() + 200000000),
              },
              {
                icon: "calendar",
                info: "Halloween Holiday",
                futureTimeDate: new Date(new Date().getTime() + 300000000),
              },
            ],
          },
          state: "running",
          values: {
            Author: "Bart Willems",
            Commit: "FEAT-1234",
            Sha: "ef758d40",
            Image: "v3.18.217",
            Size: "971.5 MB",
          },
        },
        nextInQueue: {
          title: "Deploy to EU",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 4), // 4 hours ago
        },
        collapsed: false,
      },
    },
  },
  {
    id: "deploy-asia",
    position: { x: 1250, y: 600 },
    data: {
      label: "Deploy to Asia",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deploy to Asia",
        iconSrc: KubernetesIcon,
        headerColor: "bg-blue-100",
        iconBackground: "bg-blue-500",
        collapsedBackground: "bg-blue-500",
        metadata: [
          { icon: "user", label: "Author: Bart Willems" },
          { icon: "git-commit", label: "Commit: FEAT-1234" },
          { icon: "git-commit", label: "Sha: ef758d40" },
          { icon: "package", label: "Image: v3.18.217" },
          { icon: "package", label: "Size: 971.5 MB" },
        ],
        parameters: ["asia-east-1"],
        parametersIcon: "map",
        lastRunItem: {
          title: "fix: open rejected events",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 12), // 12 hours ago
          childEventsInfo: {
            count: 1,
            state: "processed",
            waitingInfos: [],
          },
          state: "fail",
          values: {
            Author: "Bart Willems",
            Commit: "FEAT-1234",
            Sha: "ef758d40",
            Image: "v3.18.217",
            Size: "971.5 MB",
          },
        },
        startLastValuesOpen: false,
        collapsed: false,
      },
    },
  },
];

const sampleEdges: Edge[] = [
  { id: "e1", source: "listen-code", target: "build-stage" },
  { id: "e2", source: "listen-image", target: "build-stage" },
  { id: "e3", source: "build-stage", target: "approve" },
  { id: "e4", source: "approve", target: "deploy-us" },
  { id: "e5", source: "approve", target: "deploy-eu" },
  { id: "e6", source: "approve", target: "deploy-asia" },
];

// Mock execution workflow for expanded nodes
const createMockExecutionNodes = (
  title: string,
  lastRunItem: LastRunItem
): Node[] => [
  {
    id: "http-request",
    position: { x: 0, y: 0 },
    data: {
      label: "HTTP Request",
      state: "pending",
      type: "composite",
      composite: {
        title: "HTTP Request",
        description: `Execute HTTP request for ${title}`,
        iconSlug: "globe",
        iconColor: "text-blue-600",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        parameters: ["POST", "/api/deploy"],
        parametersIcon: "code",
        lastRunItem: lastRunItem,
        collapsed: false,
      },
    },
  },
];

const createMockExecutionEdges = (): Edge[] => [];

export const SimpleDeployment: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
    title: "Simple Deployment",
  },
  render: function SimpleDeploymentRender(args) {
    const [simulationNodes, setSimulationNodes] = useState<Node[]>(
      args.nodes ?? []
    );
    const simulationEdges = useMemo(() => args.edges ?? [], [args.edges]);
    const [currentView, setCurrentView] = useState<"main" | "execution">(
      "main"
    );
    const [executionContext, setExecutionContext] = useState<{
      title: string;
      breadcrumbs: BreadcrumbItem[];
      lastRunItem?: LastRunItem;
    } | null>(null);

    const handleNodeExpand = useCallback(
      (nodeId: string, nodeData: any) => {
        const nodeTitle = nodeData.composite?.title || nodeData.label;
        const composite = nodeData.composite;

        // Navigate to BlueprintExecutionPage story with data for specific nodes
        if (
          nodeTitle === "Build/Test/Deploy Stage" ||
          nodeTitle === "Deploy to US" ||
          nodeTitle === "Deploy to EU" ||
          nodeTitle === "Deploy to Asia"
        ) {
          const executionData = {
            title: nodeTitle,
            composite: composite,
            parentWorkflow: args.title || "Simple Deployment",
            nodeId: nodeId,
            timestamp: Date.now(),
          };

          navigateToStoryWithData(
            "pages-canvaspage--blueprint-execution-page",
            executionData
          );
          return;
        }

        const breadcrumbs: BreadcrumbItem[] = [
          {
            label: "Workflows",
          },
          {
            label: args.title || "Simple Deployment",
            onClick: () => setCurrentView("main"),
          },
          {
            label: nodeTitle,
            iconSrc: composite?.iconSrc,
            iconSlug: composite?.iconSlug,
            iconColor: composite?.iconColor,
            iconBackground: composite?.iconBackground,
          },
        ];

        setExecutionContext({
          title: nodeTitle,
          breadcrumbs,
          lastRunItem: composite?.lastRunItem,
        });
        setCurrentView("execution");
      },
      [args.title]
    );

    const renderContent = () => {
      if (currentView === "execution" && executionContext) {
        return (
          <CanvasPage
            nodes={createMockExecutionNodes(
              executionContext.title,
              executionContext.lastRunItem!
            )}
            edges={createMockExecutionEdges()}
            title={executionContext.title}
            breadcrumbs={executionContext.breadcrumbs}
          />
        );
      }

      return (
        <CanvasPage
          {...args}
          nodes={simulationNodes}
          edges={simulationEdges}
          onNodeExpand={handleNodeExpand}
        />
      );
    };

    const runSimulation = useSimulationRunner({
      nodes: simulationNodes,
      edges: setSimulationNodes,
      setNodes: setSimulationNodes,
    });

    return (
      <div className="h-[100vh] w-full ">
        <div className="absolute z-[999] top-3 right-4">
          <button
            onClick={() => runSimulation("listen-code")}
            className="px-3 py-1 rounded bg-blue-600 text-white text-xs shadow hover:bg-blue-700"
          >
            Run
          </button>
        </div>
        {renderContent()}
      </div>
    );
  },
};

export const CollapsedDeployment: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
    startCollapsed: true,
    title: "Simple Deployment",
  },
  render: (args) => {
    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage {...args} />
      </div>
    );
  },
};

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
