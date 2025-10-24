import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";

import dockerIcon from "@/assets/icons/integrations/docker.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

import { useCallback, useMemo, useState } from "react";
import { LastRunItem } from "../composite";
import type { BreadcrumbItem } from "./Header";
import { CanvasNode, CanvasPage } from "./index";
import { sleep, useSimulator } from "./storybooks/useSimulator";

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

const sampleNodes: CanvasNode[] = [
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
          sizeInMB: 1,
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
          state: "processed",
        },
      },
    },
    _run: async () => {
      return {
        title: "refactor: commit 1234",
        sizeInMB: 1,
        receivedAt: new Date(),
        state: "processed",
      };
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
          sizeInMB: 972.5,
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 3), // 3 hours ago
          state: "processed",
        },
      },
    },
    _run: async () => {
      return {
        title: "v3.18.218",
        sizeInMB: 973.1,
        receivedAt: new Date(),
        state: "processed",
      };
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
      },
    },
    _run: async () => {
      await sleep(5000);

      return {
        title: "FEAT-1234: New feature",
        subtitle: "ef758d40",
        receivedAt: new Date(),
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
      };
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
    const [nodes, setNodes] = useState<CanvasNode[]>(args.nodes!);
    const edges = useMemo(() => args.edges!, [args.edges]);

    const [currentView, setCurrentView] = useState<"main" | "execution">(
      "main"
    );

    const [executionContext, setExecutionContext] = useState<{
      title: string;
      breadcrumbs: BreadcrumbItem[];
      lastRunItem?: LastRunItem;
    } | null>(null);

    const run = useSimulator(nodes, edges, setNodes);

    const handleNodeExpand = useCallback(
      (nodeId: string, nodeData: any) => {
        const nodeTitle = nodeData.composite?.title || nodeData.label;
        const composite = nodeData.composite;

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
          nodes={nodes}
          edges={edges}
          onNodeExpand={handleNodeExpand}
        />
      );
    };

    return (
      <div className="h-[100vh] w-full">
        <div className="absolute z-[999] right-4 top-3">
          <button
            onClick={() => run("listen-code")}
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
