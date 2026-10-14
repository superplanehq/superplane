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

  it("explains token creation failures and permits a retry", async () => {
    createToken.mockRejectedValueOnce(new Error("request failed"));
    render(<FleetManagerSetup />);

    fireEvent.click(await screen.findByRole("button", { name: "Create Fleet Manager token" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("SuperPlane could not create the token. Try again.");
    expect(screen.getByRole("button", { name: "Create Fleet Manager token" })).toBeEnabled();
  });

  it("prevents copying values when capacity is invalid", async () => {
    render(<FleetManagerSetup />);
    fireEvent.change(await screen.findByLabelText("Warm capacity"), { target: { value: "2" } });

    expect(screen.getByRole("button", { name: "Copy values.yaml" })).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent("Warm capacity cannot exceed maximum capacity.");
  });

  it("keeps infrastructure fields separate for each provider", async () => {
    render(<FleetManagerSetup />);
    fireEvent.change(await screen.findByLabelText("EC2 instance type for AMD64"), {
      target: { value: "m8a.large" },
    });

    fireEvent.click(screen.getByRole("button", { name: "GCP" }));
    expect(screen.getByLabelText("Compute Engine machine type for AMD64")).toHaveValue("");

    fireEvent.click(screen.getByRole("button", { name: "AWS" }));
    expect(screen.getByLabelText("EC2 instance type for AMD64")).toHaveValue("m8a.large");
  });
});
