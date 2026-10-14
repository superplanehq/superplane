import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import FleetManagerSetup from "./FleetManagerSetup";

const createToken = vi.fn();

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [{ id: "org-1", name: "Example" }] }),
}));

vi.mock("@/hooks/useUserTokens", () => ({
  useCreateUserToken: () => ({ mutateAsync: createToken, isPending: false }),
}));

const metadata = {
  defaultRunnerVersion: "v0.1.0",
  defaultFleetManagerVersion: "v0.1.0",
  defaultChartVersion: "0.1.0",
  runnerReleaseUrlTemplate: "https://example/releases/runner%2F{version}",
  chartRepository: "oci://ghcr.io/superplanehq/fleet-manager-chart",
  imageRepository: "ghcr.io/superplanehq/fleet-manager",
  fleetPresets: [
    {
      id: "e1-large-amd64",
      spec: {
        operatingSystem: "linux",
        architecture: "amd64",
        cpuMillicores: 2000,
        memoryMb: 8192,
        diskGb: 30,
        capabilities: ["docker"],
      },
    },
  ],
};

describe("FleetManagerSetup", () => {
  beforeEach(() => {
    createToken.mockReset();
    createToken.mockResolvedValue({ plaintext: "one-time-token" });
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify(metadata), { status: 200 })),
    );
  });

  afterEach(() => vi.unstubAllGlobals());

  it("loads setup metadata without creating a token", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("heading", { name: "1. Select a provider and fleets" })).toBeInTheDocument();
    expect(createToken).not.toHaveBeenCalled();
    expect((screen.getByLabelText("Fleet Manager values") as HTMLTextAreaElement).value).toContain("provider: aws");
  });

  it("creates and shows the token only after an explicit action", async () => {
    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("button", { name: "Create Fleet Manager token" }));

    await waitFor(() => expect(createToken).toHaveBeenCalledWith({ name: "Fleet Manager" }));
    expect(screen.getByLabelText("Fleet Manager token")).toHaveValue("one-time-token");
    expect((screen.getByLabelText("Fleet Manager values") as HTMLTextAreaElement).value).not.toContain(
      "one-time-token",
    );
    expect((screen.getByLabelText("Install Fleet Manager command") as HTMLTextAreaElement).value).not.toContain(
      "token",
    );
  });
});
