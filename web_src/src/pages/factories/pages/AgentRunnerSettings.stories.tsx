import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";
import { HttpResponse, http, delay } from "msw";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { SuperplaneComponentsNode } from "@/api-client";

import { AgentRunnerSettings } from "./AgentRunnerSettings";
import { agentNodeWithCommand, LONG_BASH_COMMAND } from "./planningReviewFindBox.storyHelpers";

const meta = {
  title: "Factories/Pages/Agent Runner Settings",
  component: AgentRunnerSettings,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AgentRunnerSettings>;

export default meta;

type Story = StoryObj<typeof AgentRunnerSettings>;

function panel(command: string) {
  return (
    <MemoryRouter>
      <div className="flex h-svh w-full max-w-3xl flex-col border-r border-border bg-background">
        <AgentRunnerSettings
          node={agentNodeWithCommand(command)}
          onSave={(update) => console.log("save agent runner", update)}
        />
      </div>
    </MemoryRouter>
  );
}

export const OneLine: Story = {
  name: "One line script",
  render: () => panel('git clone --depth 1 "$REPO_URL" repo'),
};

export const LongScript: Story = {
  name: "Long script",
  render: () => panel(LONG_BASH_COMMAND),
};

function FleetPanel({ unavailable = false, readOnly = false }: { unavailable?: boolean; readOnly?: boolean }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [node, setNode] = useState<SuperplaneComponentsNode>(() => {
    const agent = agentNodeWithCommand("git status");
    return {
      ...agent,
      configuration: { ...agent.configuration, machineType: unavailable ? "retired-team-fleet" : "team-linux" },
    };
  });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <div className="flex h-svh max-w-3xl flex-col">
          <AgentRunnerSettings
            node={node}
            organizationId="story-org"
            disabled={readOnly}
            onSave={(update) => {
              console.log("Save agent settings", update);
              setNode((current) => ({ ...current, configuration: update.configuration }));
            }}
          />
        </div>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

const fleetHandlers = [
  http.get("*/api/v1/organizations/story-org/fleets/available", () =>
    HttpResponse.json({
      fleets: [
        {
          id: "team-linux",
          scope: "organization",
          spec: { operatingSystem: "linux", cpuMillicores: 4000, memoryMb: 8192 },
        },
        {
          id: "e1-large-amd64",
          scope: "installation",
          spec: { operatingSystem: "linux", cpuMillicores: 8000, memoryMb: 32768 },
        },
      ],
    }),
  ),
];
export const RunnerFleets: Story = { render: () => <FleetPanel />, parameters: { msw: { handlers: fleetHandlers } } };
export const UnavailableFleet: Story = {
  render: () => <FleetPanel unavailable />,
  parameters: { msw: { handlers: fleetHandlers } },
};
export const ReadOnlyFleet: Story = {
  render: () => <FleetPanel readOnly />,
  parameters: { msw: { handlers: fleetHandlers } },
};
export const FleetRequestFailure: Story = {
  render: () => <FleetPanel />,
  parameters: {
    msw: {
      handlers: [
        http.get("*/api/v1/organizations/story-org/fleets/available", () => new HttpResponse(null, { status: 500 })),
      ],
    },
  },
};
export const LoadingFleets: Story = {
  render: () => <FleetPanel />,
  parameters: {
    msw: {
      handlers: [
        http.get("*/api/v1/organizations/story-org/fleets/available", async () => {
          await delay("infinite");
          return HttpResponse.json({ fleets: [] });
        }),
      ],
    },
  },
};
