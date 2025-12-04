import { ComponentBaseProps } from "@/ui/componentBase";
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
          parameters: [{ icon: "code", items: { method: "POST", endpoint: "/api/deploy" } }],
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
          parameters: [{ icon: "list-checks", items: { tests: "health-check, smoke-test" } }],
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
          parameters: [{ icon: "server", items: { cleanup: "temp-storage, build-cache" } }],
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
          parameters: [{ icon: "mail", items: { channels: "slack, email" } }],
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
};

export const DeployToUS = {
  nodes: [
    {
      id: "drain-traffic",
      position: { x: 0, y: 0 },
      data: {
        label: `Drain Traffic`,
        state: "success",
        type: "composite",
        composite: {
          title: `Drain Traffic`,
          description: `Reduce traffic to 0; wait to drain`,
          iconSlug: "traffic-cone",
          iconColor: "text-amber-600",
          headerColor: "bg-amber-100",
          collapsedBackground: "bg-amber-100",
          parameters: [{ icon: "globe", items: { domain: "us.example.com", weight: "0%" } }],
          lastRunItem: {
            title: `Drain complete`,
            subtitle: "ingress/us",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 25),
            state: "success",
            values: { Drained: "OK", Connections: "0" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "argo-rollout",
      position: { x: 480, y: 0 },
      data: {
        label: `Rollout with Argo`,
        state: "working",
        type: "composite",
        composite: {
          title: `Rollout with Argo`,
          description: `Sync and roll out application`,
          iconSlug: "git-branch",
          iconColor: "text-blue-700",
          headerColor: "bg-blue-100",
          collapsedBackground: "bg-blue-100",
          parameters: [{ icon: "boxes", items: { app: "us-api", strategy: "canary" } }],
          lastRunItem: {
            title: `Argo sync`,
            subtitle: "rollout/us-api",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 20),
            state: "success",
            values: { Revision: "12", Status: "Healthy" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "run-migrations",
      position: { x: 960, y: 0 },
      data: {
        label: `Run Migrations`,
        state: "success",
        type: "composite",
        composite: {
          title: `Run Migrations`,
          description: `Apply DB migrations safely`,
          iconSlug: "database",
          iconColor: "text-emerald-700",
          headerColor: "bg-emerald-100",
          collapsedBackground: "bg-emerald-100",
          parameters: [{ icon: "server-cog", items: { job: "migrate", concurrency: "1" } }],
          lastRunItem: {
            title: `Migrations applied`,
            subtitle: "db/main",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 15),
            state: "success",
            values: { Steps: "3", Status: "OK" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "health-check",
      position: { x: 1440, y: 0 },
      data: {
        label: `Health Check`,
        state: "working",
        type: "composite",
        composite: {
          title: `Health Check`,
          description: `Probe readiness and SLOs`,
          iconSlug: "heartbeat",
          iconColor: "text-green-700",
          headerColor: "bg-green-100",
          collapsedBackground: "bg-green-100",
          parameters: [{ icon: "stethoscope", items: { endpoint: "/healthz", threshold: "p95<250ms" } }],
          lastRunItem: {
            title: `Probes`,
            subtitle: "readiness/liveness",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 10),
            state: "running",
            values: { Readiness: "OK", Latency: "210ms" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "enable-traffic",
      position: { x: 1920, y: 0 },
      data: {
        label: `Enable Live Traffic`,
        state: "pending",
        type: "composite",
        composite: {
          title: `Enable Live Traffic`,
          description: `Restore weight to 100%`,
          iconSlug: "toggle-right",
          iconColor: "text-purple-700",
          headerColor: "bg-purple-100",
          collapsedBackground: "bg-purple-100",
          parameters: [{ icon: "globe", items: { domain: "us.example.com", weight: "100%" } }],
          lastRunItem: {
            title: `Cutover pending`,
            subtitle: "promotion",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5),
            state: "pending",
            values: {},
          },
          collapsed: false,
        },
      },
    },
  ],
  edges: [
    { id: "us-e1", source: "drain-traffic", target: "argo-rollout" },
    { id: "us-e2", source: "argo-rollout", target: "run-migrations" },
    { id: "us-e3", source: "run-migrations", target: "health-check" },
    { id: "us-e4", source: "health-check", target: "enable-traffic" },
  ],
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
      iconBackground: "bg-blue-500",
    },
  ],
};

export const DeployToEU = {
  nodes: [
    {
      id: "eu-drain-traffic",
      position: { x: 0, y: 0 },
      data: {
        label: `Drain Traffic`,
        state: "success",
        type: "composite",
        composite: {
          title: `Drain Traffic`,
          description: `Reduce traffic to 0; wait to drain`,
          iconSlug: "traffic-cone",
          iconColor: "text-amber-600",
          headerColor: "bg-amber-100",
          collapsedBackground: "bg-amber-100",
          parameters: [{ icon: "globe", items: { domain: "eu.example.com", weight: "0%" } }],
          lastRunItem: {
            title: `Drain complete`,
            subtitle: "ingress/eu",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 25),
            state: "success",
            values: { Drained: "OK", Connections: "0" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "eu-argo-rollout",
      position: { x: 480, y: 0 },
      data: {
        label: `Rollout with Argo`,
        state: "working",
        type: "composite",
        composite: {
          title: `Rollout with Argo`,
          description: `Sync and roll out application`,
          iconSlug: "git-branch",
          iconColor: "text-blue-700",
          headerColor: "bg-blue-100",
          collapsedBackground: "bg-blue-100",
          parameters: [{ icon: "boxes", items: { app: "eu-api", strategy: "canary" } }],
          lastRunItem: {
            title: `Argo sync`,
            subtitle: "rollout/eu-api",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 20),
            state: "success",
            values: { Revision: "8", Status: "Healthy" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "eu-run-migrations",
      position: { x: 960, y: 0 },
      data: {
        label: `Run Migrations`,
        state: "success",
        type: "composite",
        composite: {
          title: `Run Migrations`,
          description: `Apply DB migrations safely`,
          iconSlug: "database",
          iconColor: "text-emerald-700",
          headerColor: "bg-emerald-100",
          collapsedBackground: "bg-emerald-100",
          parameters: [{ icon: "server-cog", items: { job: "migrate", concurrency: "1" } }],
          lastRunItem: {
            title: `Migrations applied`,
            subtitle: "db/main",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 15),
            state: "success",
            values: { Steps: "2", Status: "OK" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "eu-health-check",
      position: { x: 1440, y: 0 },
      data: {
        label: `Health Check`,
        state: "working",
        type: "composite",
        composite: {
          title: `Health Check`,
          description: `Probe readiness and SLOs`,
          iconSlug: "heartbeat",
          iconColor: "text-green-700",
          headerColor: "bg-green-100",
          collapsedBackground: "bg-green-100",
          parameters: [{ icon: "stethoscope", items: { endpoint: "/healthz", threshold: "p95<250ms" } }],
          lastRunItem: {
            title: `Probes`,
            subtitle: "readiness/liveness",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 10),
            state: "running",
            values: { Readiness: "OK", Latency: "230ms" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "eu-enable-traffic",
      position: { x: 1920, y: 0 },
      data: {
        label: `Enable Live Traffic`,
        state: "pending",
        type: "composite",
        composite: {
          title: `Enable Live Traffic`,
          description: `Restore weight to 100%`,
          iconSlug: "toggle-right",
          iconColor: "text-purple-700",
          headerColor: "bg-purple-100",
          collapsedBackground: "bg-purple-100",
          parameters: [{ icon: "globe", items: { domain: "eu.example.com", weight: "100%" } }],
          lastRunItem: {
            title: "Cutover pending",
            subtitle: "promotion",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5),
            state: "pending",
            values: {},
          },
          collapsed: false,
        },
      },
    },
  ],
  edges: [
    { id: "eu-e1", source: "eu-drain-traffic", target: "eu-argo-rollout" },
    { id: "eu-e2", source: "eu-argo-rollout", target: "eu-run-migrations" },
    { id: "eu-e3", source: "eu-run-migrations", target: "eu-health-check" },
    { id: "eu-e4", source: "eu-health-check", target: "eu-enable-traffic" },
  ],
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
      iconBackground: "bg-blue-500",
    },
  ],
};

export const DeployToAsia = {
  nodes: [
    {
      id: "asia-drain-traffic",
      position: { x: 0, y: 0 },
      data: {
        label: `Drain Traffic`,
        state: "success",
        type: "composite",
        composite: {
          title: `Drain Traffic`,
          description: `Reduce traffic to 0; wait to drain`,
          iconSlug: "traffic-cone",
          iconColor: "text-amber-600",
          headerColor: "bg-amber-100",
          collapsedBackground: "bg-amber-100",
          parameters: [{ icon: "globe", items: { domain: "asia.example.com", weight: "0%" } }],
          lastRunItem: {
            title: `Drain complete`,
            subtitle: "ingress/asia",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 25),
            state: "success",
            values: { Drained: "OK", Connections: "0" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "asia-argo-rollout",
      position: { x: 480, y: 0 },
      data: {
        label: `Rollout with Argo`,
        state: "working",
        type: "composite",
        composite: {
          title: `Rollout with Argo`,
          description: `Sync and roll out application`,
          iconSlug: "git-branch",
          iconColor: "text-blue-700",
          headerColor: "bg-blue-100",
          collapsedBackground: "bg-blue-100",
          parameters: [{ icon: "boxes", items: { app: "asia-api", strategy: "canary" } }],
          lastRunItem: {
            title: `Argo sync`,
            subtitle: "rollout/asia-api",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 20),
            state: "success",
            values: { Revision: "5", Status: "Healthy" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "asia-run-migrations",
      position: { x: 960, y: 0 },
      data: {
        label: `Run Migrations`,
        state: "success",
        type: "composite",
        composite: {
          title: `Run Migrations`,
          description: `Apply DB migrations safely`,
          iconSlug: "database",
          iconColor: "text-emerald-700",
          headerColor: "bg-emerald-100",
          collapsedBackground: "bg-emerald-100",
          parameters: [{ icon: "server-cog", items: { job: "migrate", concurrency: "1" } }],
          lastRunItem: {
            title: `Migrations applied`,
            subtitle: "db/main",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 15),
            state: "success",
            values: { Steps: "3", Status: "OK" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "asia-health-check",
      position: { x: 1440, y: 0 },
      data: {
        label: `Health Check`,
        state: "working",
        type: "composite",
        composite: {
          title: `Health Check`,
          description: `Probe readiness and SLOs`,
          iconSlug: "heartbeat",
          iconColor: "text-green-700",
          headerColor: "bg-green-100",
          collapsedBackground: "bg-green-100",
          parameters: [{ icon: "stethoscope", items: { endpoint: "/healthz", threshold: "p95<250ms" } }],
          lastRunItem: {
            title: `Probes`,
            subtitle: "readiness/liveness",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 10),
            state: "running",
            values: { Readiness: "OK", Latency: "220ms" },
          },
          collapsed: false,
        },
      },
    },
    {
      id: "asia-enable-traffic",
      position: { x: 1920, y: 0 },
      data: {
        label: `Enable Live Traffic`,
        state: "pending",
        type: "composite",
        composite: {
          title: `Enable Live Traffic`,
          description: `Restore weight to 100%`,
          iconSlug: "toggle-right",
          iconColor: "text-purple-700",
          headerColor: "bg-purple-100",
          collapsedBackground: "bg-purple-100",
          parameters: [{ icon: "globe", items: { domain: "asia.example.com", weight: "100%" } }],
          lastRunItem: {
            title: "Cutover pending",
            subtitle: "promotion",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5),
            state: "pending",
            values: {},
          },
          collapsed: false,
        },
      },
    },
  ],
  edges: [
    { id: "asia-e1", source: "asia-drain-traffic", target: "asia-argo-rollout" },
    { id: "asia-e2", source: "asia-argo-rollout", target: "asia-run-migrations" },
    { id: "asia-e3", source: "asia-run-migrations", target: "asia-health-check" },
    { id: "asia-e4", source: "asia-health-check", target: "asia-enable-traffic" },
  ],
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
      iconBackground: "bg-blue-500",
    },
  ],
};

export const Provisioner = {
  nodes: [
    {
      id: "provisioner-action",
      position: { x: 100, y: -100 },
      data: {
        label: `NOOP 1`,
        state: "working",
        type: "component",
        component: {
          title: "NOOP 1",
          description: "NOOP 1",
          headerColor: "bg-gray-50",
          collapsed: false,
          eventSections: [{
            title: "Last Run",
            eventState: "success",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
          }],
        } as ComponentBaseProps,
      },
    },
    {
      id: "provisioner-action-2",
      position: { x: 600, y: -100 },
      data: {
        label: `NOOP 2`,
        state: "working",
        type: "component",
        component: {
          title: "NOOP 2",
          description: "NOOP 2",
          headerColor: "bg-gray-50",
          collapsed: false,
          eventSections: [{
            title: "Last Run",
            eventState: "success",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
          }],
        } as ComponentBaseProps,
      },
    },
  ],
  edges: [{ id: "e1", source: "provisioner-action", target: "provisioner-action-2" }],
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
};

export const Desprovisioner = {
  nodes: [
    {
      id: "desprovisioner-action",
      position: { x: 100, y: -100 },
      data: {
        label: `NOOP 1`,
        state: "working",
        type: "component",
        component: {
          title: "NOOP 1",
          description: "NOOP 1",
          headerColor: "bg-gray-50",
          collapsed: false,
          eventSections: [{
            title: "Last Run",
            eventState: "success",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
          }],
        } as ComponentBaseProps,
      },
    },
    {
      id: "desprovisioner-action-2",
      position: { x: 600, y: -100 },
      data: {
        label: `NOOP 2`,
        state: "working",
        type: "component",
        component: {
          title: "NOOP 2",
          description: "NOOP 2",
          headerColor: "bg-gray-50",
          collapsed: false,
          eventSections: [{
            title: "Last Run",
            eventState: "success",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30),
            eventTitle: "FEAT: Add new feature",
            eventSubtitle: "ef546d40",
          }],
        } as ComponentBaseProps,
      },
    },
  ],
  edges: [{ id: "e1", source: "desprovisioner-action", target: "desprovisioner-action-2" }],
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
};

export const SubWorkflowsMap = {
  "Build/Test/Deploy Stage": MainSubWorkflow,
  "Deploy to US": DeployToUS,
  "Deploy to EU": DeployToEU,
  "Deploy to Asia": DeployToAsia,
  Provisioner: Provisioner,
  Desprovisioner: Desprovisioner,
};
