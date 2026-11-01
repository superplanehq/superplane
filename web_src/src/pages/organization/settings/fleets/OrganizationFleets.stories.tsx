import type { Meta, StoryObj } from "@storybook/react-vite";
import { userEvent, within } from "storybook/test";
import { FactoriesHarness } from "@/pages/factories/__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "@/pages/factories/__fixtures__/factoryPageResponses";
import { FEATURE_ORGANIZATION_RUNNERS } from "@/lib/experimentalFeatures";

const fleet = {
  id: "self-hosted",
  enabled: true,
  runnerVersion: "v1.0",
  spec: {
    operatingSystem: "linux",
    architecture: "amd64",
    cpuMillicores: 2000,
    memoryMb: 4096,
    diskGb: 40,
    capabilities: ["docker"],
  },
};
const runner = {
  id: "11111111-1111-4111-8111-111111111111",
  fleetId: fleet.id,
  state: "busy",
  ephemeral: true,
  runnerVersion: "v1.0",
  hostname: "self-hosted-01",
  os: "linux",
  arch: "amd64",
  ip: "192.0.2.10",
  registeredAt: new Date().toISOString(),
};

function FleetStory({
  management,
  suffix = "",
  featureEnabled = true,
}: {
  management: NonNullable<FactoriesFixture["organizationFleetManagement"]>;
  suffix?: string;
  featureEnabled?: boolean;
}) {
  return (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/fleets${suffix}`}
      experimentalFeatures={featureEnabled ? [FEATURE_ORGANIZATION_RUNNERS] : []}
      factoriesFixture={{ ...defaultFactoriesFixture, organizationFleetManagement: management }}
    />
  );
}
const meta = {
  title: "Factories/Pages/Settings/Fleets",
  component: FleetStory,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FleetStory>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Populated: Story = { args: { management: { fleets: [fleet], runners: [runner] } } };
export const Empty: Story = { args: { management: { fleets: [], runners: [] } } };
export const ReadOnly: Story = {
  args: { management: { fleets: [fleet], runners: [runner], readOnly: true }, suffix: "/self-hosted" },
};
export const FeatureDisabled: Story = {
  args: { management: { fleets: [fleet], runners: [runner] }, featureEnabled: false },
};
export const ApiError: Story = { args: { management: { fleets: [], runners: [], error: true } } };
export const WaitingForConnection: Story = {
  args: { management: { fleets: [fleet], runners: [], connection: "waiting" }, suffix: "/self-hosted" },
  play: async ({ canvasElement }) => {
    await userEvent.click(await within(canvasElement).findByRole("button", { name: "Connect runner" }));
    await userEvent.click(await within(document.body).findByRole("button", { name: "Generate registration" }));
  },
};
export const Connected: Story = {
  ...WaitingForConnection,
  args: { management: { fleets: [fleet], runners: [], connection: "connected" }, suffix: "/self-hosted" },
};
export const ExpiredRegistration: Story = {
  ...WaitingForConnection,
  args: { management: { fleets: [fleet], runners: [], connection: "expired" }, suffix: "/self-hosted" },
};
export const PendingTermination: Story = {
  args: {
    management: { fleets: [fleet], runners: [runner], pendingTermination: true },
    suffix: `/self-hosted/runners/${runner.id}`,
  },
  play: async ({ canvasElement }) => {
    await userEvent.click(await within(canvasElement).findByRole("button", { name: "Terminate runner" }));
    await userEvent.click(await within(document.body).findByRole("button", { name: "Confirm termination" }));
  },
};
