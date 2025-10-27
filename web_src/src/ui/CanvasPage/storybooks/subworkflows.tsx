import { NoopProps } from "@/ui/noop";
import { navigateToStory } from "./navigation";
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

export const MainSubWorkflow = {
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
          parameters: [
            { icon: "code", items: ["POST", "/api/deploy"] }
          ],
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
          parameters: [
            { icon: "list-checks", items: ["health-check", "smoke-test"] }
          ],
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
          parameters: [
            { icon: "server", items: ["temp-storage", "build-cache"] }
          ],
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
          parameters: [
            { icon: "mail", items: ["slack", "email"] }
          ],
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
      onClick: () => navigateToStory("pages-canvaspage-examples--simple-deployment"),
    },
    {
      label: "Build/Test/Deploy Stage",
      iconSlug: "git-branch",
      iconColor: "text-purple-700",
    },
  ],
}

export const DeployToUS = {
  nodes: [
    {
      id: "deploy-action",
      position: { x: 100, y: -100 },
      data: {
        label: `Deploy to US`,
        state: "working",
        type: "composite",
        composite: {
          title: `Deploy to US`,
          description: `Execute deployment to US region`,
          iconSrc: KubernetesIcon,
          headerColor: "bg-blue-100",
          iconBackground: 'bg-blue-500',
          collapsedBackground: 'bg-blue-100',
          parameters: [
            { icon: "globe", items: ["/api/us", "200 OK"] }
          ],
          lastRunItem: {
            title: `Deploy to US`,
            subtitle: "default",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            state: "success",
            values: {},
          },
          collapsed: false,
        },
      },
    }
  ],
  edges: [],
  title: "Deploy to US",
  breadcrumbs: [
    {
      label: "Workflows",
    },
    {
      label: "Simple Deployment",
      onClick: () => navigateToStory("pages-canvaspage-examples--simple-deployment"),
    },
    {
      label: "Deploy to US",
      iconSrc: KubernetesIcon,
      iconBackground: 'bg-blue-500',
    },
  ],
}

export const DeployToEU = {
  nodes: [
    {
      id: "deploy-action",
      position: { x: 100, y: -100 },
      data: {
        label: `Deploy to EU`,
        state: "working",
        type: "composite",
        composite: {
          title: `Deploy to EU`,
          description: `Execute deployment to EU region`,
          iconSrc: KubernetesIcon,
          headerColor: "bg-blue-100",
          iconBackground: 'bg-blue-500',
          collapsedBackground: 'bg-blue-100',
          parameters: [
            { icon: "globe", items: ["/api/eu", "200 OK"] }
          ],
          lastRunItem: {
            title: `Deploy to EU`,
            subtitle: "default",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            state: "success",
            values: {},
          },
          collapsed: false,
        },
      },
    }
  ],
  edges: [],
  title: "Deploy to EU",
  breadcrumbs: [
    {
      label: "Workflows",
    },
    {
      label: "Simple Deployment",
      onClick: () => navigateToStory("pages-canvaspage-examples--simple-deployment"),
    },
    {
      label: "Deploy to EU",
      iconSrc: KubernetesIcon,
      iconBackground: 'bg-blue-500',
    },
  ],
}

export const DeployToAsia = {
  nodes: [
    {
      id: "deploy-action",
      position: { x: 100, y: -100 },
      data: {
        label: `Deploy to Asia`,
        state: "working",
        type: "composite",
        composite: {
          title: `Deploy to Asia`,
          description: `Execute deployment to Asia region`,
          iconSrc: KubernetesIcon,
          headerColor: "bg-blue-100",
          iconBackground: 'bg-blue-500',
          collapsedBackground: 'bg-blue-100',
          parameters: [
            { icon: "globe", items: ["/api/asia", "200 OK"] }
          ],
          lastRunItem: {
            title: `Deploy to Asia`,
            subtitle: "default",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            state: "success",
            values: {},
          },
          collapsed: false,
        },
      },
    }
  ],
  edges: [],
  title: "Deploy to Asia",
  breadcrumbs: [
    {
      label: "Workflows",
    },
    {
      label: "Simple Deployment",
      onClick: () => navigateToStory("pages-canvaspage-examples--simple-deployment"),
    },
    {
      label: "Deploy to Asia",
      iconSrc: KubernetesIcon,
      iconBackground: 'bg-blue-500',
    },
  ],
}

export const Provisioner = {
  nodes: [
    {
      id: "provisioner-action",
      position: { x: 100, y: -100 },
      data: {
        label: `NOOP 1`,
        state: "working",
        type: "noop",
        noop: {
          title: "NOOP 1",
          description: "NOOP 1",
          collapsed: false,
          lastEvent: {
            eventState: "success",
            eventReceivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
          }
        } as NoopProps,
      },
    },
    {
      id: "provisioner-action-2",
      position: { x: 600, y: -100 },
      data: {
        label: `NOOP 2`,
        state: "working",
        type: "noop",
        noop: {
          title: "NOOP 2",
          description: "NOOP 2",
          collapsed: false,
          lastEvent: {
            eventState: "success",
            eventReceivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
          }
        } as NoopProps,
      },
    }
  ],
  edges: [
    { id: "e1", source: "provisioner-action", target: "provisioner-action-2" },
  ],
  title: "Provisioner",
  breadcrumbs: [
    {
      label: "Workflows",
    },
    {
      label: "Ephemeral Environments",
      onClick: () => navigateToStory("pages-canvaspage-examples--ephemeral"),
    },
    {
      label: "Provisioner",
      iconSlug: "boxes",
      iconColor: "text-blue-600",
    },
  ],
}

export const Desprovisioner = {
  nodes: [
    {
      id: "desprovisioner-action",
      position: { x: 100, y: -100 },
      data: {
        label: `NOOP 1`,
        state: "working",
        type: "noop",
        noop: {
          title: "NOOP 1",
          description: "NOOP 1",
          collapsed: false,
          lastEvent: {
            eventState: "success",
            eventReceivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
          }
        } as NoopProps,
      },
    },
    {
      id: "desprovisioner-action-2",
      position: { x: 600, y: -100 },
      data: {
        label: `NOOP 2`,
        state: "working",
        type: "noop",
        noop: {
          title: "NOOP 2",
          description: "NOOP 2",
          collapsed: false,
          lastEvent: {
            eventState: "success",
            eventReceivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
          }
        } as NoopProps,
      },
    }
  ],
  edges: [
    { id: "e1", source: "desprovisioner-action", target: "desprovisioner-action-2" },
  ],
  title: "Provisioner",
  breadcrumbs: [
    {
      label: "Workflows",
    },
    {
      label: "Ephemeral Environments",
      onClick: () => navigateToStory("pages-canvaspage-examples--ephemeral"),
    },
    {
      label: "Provisioner",
      iconSlug: "boxes",
      iconColor: "text-blue-600",
    },
  ],
}


export const SubWorkflowsMap = {
  "Build/Test/Deploy Stage": MainSubWorkflow,
  "Deploy to US": DeployToUS,
  "Deploy to EU": DeployToEU,
  "Deploy to Asia": DeployToAsia,
  "Provisioner": Provisioner,
  "Desprovisioner": Desprovisioner,
}