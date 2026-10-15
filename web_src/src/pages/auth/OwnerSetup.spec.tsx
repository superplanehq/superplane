import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import OwnerSetup from "./OwnerSetup";
import { OWNER_SETUP_COPY } from "./ownerSetup/ownerSetupCopy";

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

const { createToken } = vi.hoisted(() => ({ createToken: vi.fn() }));

vi.mock("@/hooks/useUserTokens", () => ({
  useCreateUserToken: () => ({ mutateAsync: createToken, isPending: false }),
}));

vi.mock("./ownerSetup/LicenseStep", () => ({
  LicenseStep: ({ onContinue }: { onContinue: () => void }) => (
    <button type="button" data-testid="owner-setup-license-skip" onClick={onContinue}>
      Continue with Community
    </button>
  ),
}));

const posthogMock = vi.hoisted(() => ({
  enabled: false,
  getActiveMatchingSurveys: vi.fn(),
  capture: vi.fn(),
}));

vi.mock("@/posthog", () => ({
  isPostHogEnabled: () => posthogMock.enabled,
  posthog: {
    getActiveMatchingSurveys: posthogMock.getActiveMatchingSurveys,
    capture: posthogMock.capture,
  },
}));

const ONBOARDING_LABELS = ["Discover", "Awaiting code", "Verify", "01 Plan", "02 Build", "03 Check", "04 Review"];

function expectOwnerSetupPreview() {
  const root = screen.getByTestId("owner-setup");
  expect(root).toHaveAttribute("data-visual", "preview");
  expect(screen.getByTestId("first-run-logo")).toHaveAttribute("alt", "SuperPlane");
  expect(root.querySelector("canvas")).toBeNull();
  expect(screen.queryByTestId("owner-setup-pane")).not.toBeInTheDocument();
  expect(screen.queryByTestId("first-run-art-pill")).not.toBeInTheDocument();
  const pane = screen.getByTestId("first-run-art-pane");
  expect(pane).toHaveAttribute("data-art-mode", "globe");
  expect(pane).toHaveAttribute("data-art-background", "#b09532");
  expect(pane).toHaveAttribute("data-art-arrow-color", "#eeede9");
  expect(pane).toHaveAttribute("data-art-count", "400");
  for (const label of ONBOARDING_LABELS) {
    expect(screen.queryByText(label)).not.toBeInTheDocument();
  }
}

const OWNER_SETUP_SURVEY = {
  id: "owner-setup-survey",
  name: "Owner Setup Survey",
  questions: [
    {
      id: "role",
      question: "What best describes your role?",
      type: "single_choice",
      choices: ["Software engineer"],
    },
  ],
};

const fleetManagerMetadata = {
  defaultRunnerVersion: "v0.1.0",
  defaultFleetManagerVersion: "v0.1.0",
  defaultChartVersion: "0.1.0",
  runnerReleaseUrlTemplate: "https://example/releases/runner%2F{version}",
  chartRepository: "oci://ghcr.io/superplanehq/fleet-manager-chart",
  imageRepository: "ghcr.io/superplanehq/fleet-manager",
  runnerVersions: ["v0.1.0"],
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
  infrastructureDefaults: {
    source: "terraform",
    provider: "aws",
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    fields: {
      superplaneUrl: "https://superplane.example.com",
      region: "us-east-1",
      imageAmd64: "ami-amd64",
    },
  },
};

function stubOwnerSetupFetch() {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/api/v1/setup-owner") && init?.method === "POST") {
        return jsonResponse({ organization_id: "org-1", organization_slug: "acme" });
      }
      if (url.endsWith("/admin/api/installation/fleet-manager/setup")) {
        return jsonResponse(fleetManagerMetadata);
      }
      return new Response("", { status: 404 });
    }),
  );
}

function renderOwnerSetup() {
  render(
    <MemoryRouter>
      <OwnerSetup />
    </MemoryRouter>,
  );
}

async function reachFleetStep() {
  fireEvent.change(screen.getByPlaceholderText("First name"), { target: { value: "Ada" } });
  fireEvent.change(screen.getByPlaceholderText("Last name"), { target: { value: "Lovelace" } });
  fireEvent.change(screen.getByPlaceholderText("you@example.com"), { target: { value: "ada@example.com" } });
  fireEvent.change(screen.getByPlaceholderText("Password"), { target: { value: "Secret1x" } });
  fireEvent.change(screen.getByPlaceholderText("Confirm password"), { target: { value: "Secret1x" } });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));

  expect(await screen.findByTestId("owner-setup-license-skip")).toBeInTheDocument();
  fireEvent.click(screen.getByTestId("owner-setup-license-skip"));

  expect(await screen.findByTestId("owner-setup-smtp")).toBeInTheDocument();
  fireEvent.click(screen.getByTestId("owner-setup-smtp-skip"));

  expect(await screen.findByTestId("owner-setup-fleet")).toBeInTheDocument();
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
}

describe("OwnerSetup", () => {
  const originalLocation = window.location;

  beforeEach(() => {
    window.localStorage.clear();
    posthogMock.enabled = false;
    posthogMock.getActiveMatchingSurveys.mockReset();
    posthogMock.capture.mockReset();
    createToken.mockReset();
    createToken.mockResolvedValue({ plaintext: "owner-setup-token" });
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, href: "http://localhost/setup" },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", { configurable: true, value: originalLocation });
    vi.unstubAllGlobals();
  });

  it("shows the gold arrow preview without onboarding labels", () => {
    renderOwnerSetup();

    expectOwnerSetupPreview();
    expect(screen.queryByTestId("first-run-signed-in")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-log-out")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Create the owner account" })).toBeInTheDocument();
  });

  it("reaches workspace setup when SMTP and Fleet Manager are skipped", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    expectOwnerSetupPreview();
    expect(screen.getByTestId("first-run-signed-in")).toHaveTextContent("ada@example.com");
    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-progress")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    await waitFor(() => expect(window.location.href).toBe("/acme/workspaces/new"));
    expect(screen.queryByText(OWNER_SETUP_COPY.smtp.headline)).not.toBeInTheDocument();
  });

  it("opens the Fleet Manager wizard inside owner setup", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue }));

    expect(await screen.findByRole("heading", { name: "Choose a cloud provider" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /fleet manager setup/i })).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-progress")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByTestId("owner-setup-fleet")).toBeInTheDocument();
  });

  it("navigates forward and backward through the Fleet Manager steps", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue }));
    expect(await screen.findByRole("heading", { name: "Choose a cloud provider" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("heading", { name: "Choose runner fleets" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Generate install command" }));
    expect(await screen.findByRole("heading", { name: "Install Fleet Manager" })).toBeInTheDocument();
    expect(createToken).toHaveBeenCalledWith({ name: "Fleet Manager" });

    for (const heading of ["Choose runner fleets", "Choose a cloud provider", OWNER_SETUP_COPY.fleet.headline]) {
      fireEvent.click(screen.getByRole("button", { name: "Back" }));
      expect(await screen.findByRole("heading", { name: heading })).toBeInTheDocument();
    }
  });

  it("keeps the gold scene and hides step dots while the survey is completed", async () => {
    posthogMock.enabled = true;
    posthogMock.getActiveMatchingSurveys.mockImplementation((callback: (surveys: unknown[]) => void) => {
      callback([OWNER_SETUP_SURVEY]);
    });
    stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    expect(screen.getByTestId("first-run-progress")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    expect(await screen.findByText("What best describes your role?")).toBeInTheDocument();
    expectOwnerSetupPreview();
    expect(screen.queryByTestId("first-run-progress")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /software engineer/i }));

    await waitFor(() => expect(window.location.href).toBe("/acme/workspaces/new"));
    expect(posthogMock.capture).toHaveBeenCalledWith(
      "survey sent",
      expect.objectContaining({ $survey_id: "owner-setup-survey", $survey_completed: true }),
    );
  });
});
