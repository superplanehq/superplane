import type { Decorator, Meta, StoryObj } from "@storybook/react-vite";
import { useEffect } from "react";
import { expect, within } from "storybook/test";

import { FleetManagerStep } from "./FleetManagerStep";
import { OWNER_SETUP_COPY } from "./ownerSetupCopy";

const sampleYaml = "id: self-host\nsuperplaneUrl: https://superplane.example\ninstallationAdminToken: token\n";

function OwnerSetupStoryFrame({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen justify-center bg-background p-8">
      <div className="w-full max-w-xl">{children}</div>
    </div>
  );
}

function mockFleetPrepare(mode: "pending" | "ok" | "error") {
  const original = window.fetch.bind(window);
  window.fetch = (async (input: RequestInfo | URL) => {
    const url = String(input);
    if (!url.includes("/admin/api/installation/first-run/fleet-manager")) {
      return original(input);
    }
    if (mode === "pending") {
      return new Promise<Response>(() => undefined);
    }
    if (mode === "error") {
      return new Response("{}", { status: 500 });
    }
    return new Response(JSON.stringify({ yaml: sampleYaml, fleet_id: "e1-large-amd64", token: "tok" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  }) as typeof fetch;
  return original;
}

function withFleetPrepare(mode: "pending" | "ok" | "error"): Decorator {
  return (Story) => {
    const original = mockFleetPrepare(mode);
    useEffect(
      () => () => {
        window.fetch = original;
      },
      [],
    );
    return (
      <OwnerSetupStoryFrame>
        <Story />
      </OwnerSetupStoryFrame>
    );
  };
}

const meta = {
  title: "Auth/Owner setup/Fleet Manager",
  component: FleetManagerStep,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Owner setup shows Fleet Manager YAML to copy. Skip is available while the YAML loads.",
      },
    },
  },
  tags: ["autodocs"],
  args: {
    onContinue: () => {
      // eslint-disable-next-line no-console
      console.log("Continue");
    },
    onSkip: () => {
      // eslint-disable-next-line no-console
      console.log("Skip");
    },
  },
} satisfies Meta<typeof FleetManagerStep>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Preparing YAML. Continue stays disabled. Skip stays available. */
export const Loading: Story = {
  decorators: [withFleetPrepare("pending")],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText(OWNER_SETUP_COPY.fleet.loading)).toBeInTheDocument();
  },
};

/** YAML is ready to copy. Continue is enabled. */
export const Ready: Story = {
  decorators: [withFleetPrepare("ok")],
};

/** Prepare failed. Retry and Skip stay available. */
export const Failure: Story = {
  decorators: [withFleetPrepare("error")],
};
