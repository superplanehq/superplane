import type { Meta, StoryObj } from "@storybook/react";
import { type Edge, type Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "../canvas-reset.css";

import dockerIcon from "@/assets/icons/integrations/docker.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

import { useMemo, useState } from "react";
import { Button } from "../../button";
import { CanvasNode, CanvasPage } from "../index";
import { genCommit } from "./commits";
import { genDockerImage } from "./dockerImages";
import { handleNodeExpand } from "./navigation";
import { SimulationEngine, sleep, useSimulationRunner } from "./useSimulation";

const meta = {
  title: "Pages/CanvasPage/Examples",
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
          { icon: "filter", label: "push" },
        ],
        lastEventData: {
          title: "refactor: update README.md",
          subtitle: "ef53adfa",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
          state: "processed",
        },
      },
    },
    __simulation: {
      run: async (_input, update, output) => {
        const commit = genCommit();

        const event = {
          title: commit.message,
          subtitle: commit.sha,
          receivedAt: new Date(),
          state: "processed",
        };

        update("data.trigger.lastEventData", event);
        output(event);
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
      },
    },
    __simulation: {
      run: async (_input, update, output) => {
        const commit = genDockerImage();

        const event = {
          title: commit.message,
          subtitle: commit.size,
          receivedAt: new Date(),
          state: "processed",
        };

        update("data.trigger.lastEventData", event);
        output(event);
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
        nextInQueue: null,
      },
    },
    __simulation: {
      onQueueChange: (_current, next, update) => {
        if (next) {
          update("data.composite.nextInQueue", {
            title: next.title,
            subtitle: next.subtitle,
            receivedAt: new Date(),
          });
        } else {
          update("data.composite.nextInQueue", null);
        }
      },

      run: async (input, update, output) => {
        update("data.state", "working");
        update("data.composite.lastRunItem.title", input.title);
        update("data.composite.lastRunItem.subtitle", input.subtitle);
        update("data.composite.lastRunItem.receivedAt", new Date());

        update("data.composite.lastRunItem.state", "running");
        await sleep(5000);
        update("data.composite.lastRunItem.state", "success");

        output(input);
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
            approved: true,
            requireArtifacts: [
              {
                label: "CVE Report",
              },
            ],
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
            approved: true,
            approverName: "Lucas Pinheiro",
          },
          {
            title: "Josh Brown",
            approved: true,
          },
        ],
        awaitingEvent: null,
        receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 24),
        collapsed: false,
      },
    },
    __simulation: {
      onQueueChange: (current, _next, update) => {
        if (current) {
          update("data.approval.approvals.0.approved", false);
          update("data.approval.approvals.0.interactive", true);
          update("data.approval.awaitingEvent", {
            title: current.title,
            subtitle: current.subtitle,
          });
        } else {
          update("data.approval.awaitingEvent", null);
        }
      },
      run: async (input, _update, output) => {
        output(input);
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
        parameters: [{ icon: "map", items: ["us-west-1", "us-east-1"] }],
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
    __simulation: {
      onQueueChange: (current, next, update) => {
        if (next) {
          update("data.composite.nextInQueue", {
            title: next.title,
            subtitle: next.subtitle,
            receivedAt: new Date(),
          });
        } else {
          update("data.composite.nextInQueue", null);
        }
      },

      run: async (input, update, output) => {
        update("data.state", "working");
        update("data.composite.lastRunItem.title", input.title);
        update("data.composite.lastRunItem.subtitle", input.subtitle);
        update("data.composite.lastRunItem.receivedAt", new Date());

        update("data.composite.lastRunItem.state", "running");
        await sleep(5000);
        update("data.composite.lastRunItem.state", "success");

        output(input);
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
        parameters: [{ icon: "map", items: ["eu-global-1", "eu-global-2"] }],
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
          title: "FEAT-892: Organization level integrations page",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 4), // 4 hours ago
        },
        collapsed: false,
      },
    },
    __simulation: {
      onQueueChange: (current, next, update) => {
        if (next) {
          update("data.composite.nextInQueue", {
            title: next.title,
            subtitle: next.subtitle,
            receivedAt: new Date(),
          });
        } else {
          update("data.composite.nextInQueue", null);
        }
      },

      run: async (input, update, output) => {
        update("data.state", "working");
        update("data.composite.lastRunItem.title", input.title);
        update("data.composite.lastRunItem.subtitle", input.subtitle);
        update("data.composite.lastRunItem.receivedAt", new Date());

        update("data.composite.lastRunItem.state", "running");
        await sleep(5000);
        update("data.composite.lastRunItem.state", "success");

        output(input);
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
        parameters: [{ icon: "map", items: ["asia-east-1"] }],
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
    __simulation: {
      onQueueChange: (_current, next, update) => {
        if (next) {
          update("data.composite.nextInQueue", {
            title: next.title,
            subtitle: next.subtitle,
            receivedAt: new Date(),
          });
        } else {
          update("data.composite.nextInQueue", null);
        }
      },

      run: async (input, update, output) => {
        update("data.state", "working");
        update("data.composite.lastRunItem.title", input.title);
        update("data.composite.lastRunItem.subtitle", input.subtitle);
        update("data.composite.lastRunItem.receivedAt", new Date());

        update("data.composite.lastRunItem.state", "running");
        await sleep(5000);
        update("data.composite.lastRunItem.state", "success");

        output(input);
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

export const SimpleDeployment: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
    title: "Simple Deployment",
  },
  render: function SimpleDeploymentRender(args) {
    const [nodes, setNodes] = useState<Node[]>(args.nodes ?? []);
    const edges = useMemo(() => args.edges ?? [], [args.edges]);
    const simulation = useSimulationRunner({ nodes, edges, setNodes });

    const renderContent = () => {
      return (
        <CanvasPage
          {...args}
          nodes={nodes}
          edges={edges}
          onNodeExpand={handleNodeExpand}
          onApprove={simulation.onApprove.bind(simulation)}
          onReject={simulation.onReject.bind(simulation)}
        />
      );
    };

    return (
      <div className="h-[100vh] w-full ">
        <SimulatorButtons simulation={simulation} />

        {renderContent()}
      </div>
    );
  },
};

SimpleDeployment.storyName = "01 - Simple Deployment";

function SimulatorButtons({ simulation }: { simulation: SimulationEngine }) {
  return (
    <div className="absolute z-[999] bottom-3 left-3 flex gap-2">
      <Button
        onClick={() => simulation.run("listen-code")}
        size="sm"
        variant="outline"
      >
        GitHub Push
      </Button>

      <Button
        onClick={() => simulation.run("listen-image")}
        size="sm"
        variant="outline"
      >
        Docker Image Push
      </Button>
    </div>
  );
}
