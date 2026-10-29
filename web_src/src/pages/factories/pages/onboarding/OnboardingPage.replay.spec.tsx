import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";
import { StrictMode } from "react";
import { Link } from "react-router";

import { client } from "@/api-client/client.gen";
import { FEATURE_ORGANIZATION_BYOK, FEATURE_ORGANIZATION_BYOK_CUSTOM_PROVIDER } from "@/lib/experimentalFeatures";
import { posthog } from "@/posthog";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  factoriesFixtureWithSetupAnswers,
  SETUP_ANSWERS,
} from "../../__fixtures__/setupStoryFixtures";
import { NO_GRANT_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { OnboardingPage } from "./OnboardingPage";

const installFactory = vi.hoisted(() => vi.fn().mockResolvedValue({ canvasId: "agent-app", canvasName: "agent-app" }));
vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory, isInstalling: false }),
}));

const analyticsWindow = window as Window & { SUPERPLANE_POSTHOG_KEY?: string };
const setupPath = `workspaces/${PRIMARY_FACTORY_KEY}/setup?step=issues`;

function setupFixture() {
  const fixture = factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.issues, {
    organizationWorkspaceUsage: NO_GRANT_USAGE_REPORT,
  });
  fixture.factories = fixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID
      ? { ...factory, lines: [] }
      : { ...factory, lines: [], onboarding: { completedAt: new Date().toISOString() } },
  );
  fixture.appsByFactoryId = { [PRIMARY_FACTORY_ID]: [] };
  fixture.intakesByFactoryId = { [PRIMARY_FACTORY_ID]: [] };
  fixture.organizationBilling = { billingEnabled: false, productFeatures: { byo_key: true } };
  fixture.byokConnectedProviders = ["anthropic"];
  fixture.byokCandidatesByProvider = { anthropic: ["claude-sonnet-4-6"] };
  return fixture;
}

function WorkspaceList() {
  return <Link to={`/${FACTORIES_ORGANIZATION_ID}/${setupPath}`}>Return to onboarding</Link>;
}

function onboardingHarness(pathSuffix = setupPath) {
  return (
    <FactoriesHarness
      mode="app"
      billingEnabled={false}
      pathSuffix={pathSuffix}
      factoriesFixture={setupFixture()}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      experimentalFeatures={[FEATURE_ORGANIZATION_BYOK, FEATURE_ORGANIZATION_BYOK_CUSTOM_PROVIDER]}
      pageOverrides={{ onboarding: OnboardingPage, overview: WorkspaceList }}
    />
  );
}

function dispatchPageTransition(type: "pagehide" | "pageshow", persisted = false) {
  const event = new Event(type);
  Object.defineProperty(event, "persisted", { value: persisted });
  window.dispatchEvent(event);
}

describe("onboarding session replay", () => {
  let startRecording: ReturnType<typeof vi.spyOn>;
  let stopRecording: ReturnType<typeof vi.spyOn>;

  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  beforeEach(() => {
    localStorage.clear();
    analyticsWindow.SUPERPLANE_POSTHOG_KEY = "test-key";
    startRecording = vi.spyOn(posthog, "startSessionRecording").mockImplementation(() => {});
    stopRecording = vi.spyOn(posthog, "stopSessionRecording").mockImplementation(() => {});
  });

  afterEach(() => {
    cleanup();
    startRecording.mockRestore();
    stopRecording.mockRestore();
    delete analyticsWindow.SUPERPLANE_POSTHOG_KEY;
    localStorage.clear();
  });

  it("records across steps, an integration dialog, and analysis until opening the board", async () => {
    const user = userEvent.setup();
    render(onboardingHarness());
    await screen.findByTestId("first-run-analyze-tickets", {}, { timeout: 8000 });
    expect(startRecording).toHaveBeenCalledWith({ sampling: true, linked_flag: true });

    await user.click(screen.getByTestId("first-run-analyze-tickets"));
    await user.click(await screen.findByRole("button", { name: "Connect Custom provider" }));
    const dialog = await screen.findByRole("dialog");
    expect(startRecording).toHaveBeenCalledTimes(1);
    expect(stopRecording).not.toHaveBeenCalled();
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "Connect Anthropic" }));
    await waitFor(() => expect(screen.getByRole("combobox", { name: "Model" })).toBeEnabled());
    await user.click(screen.getByTestId("first-run-finish-setup"));

    await screen.findByTestId("first-run-analysis", {}, { timeout: 8000 });
    expect(startRecording).toHaveBeenCalledTimes(1);
    expect(stopRecording).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("first-run-go-to-board"));
    await waitFor(() => expect(screen.queryByTestId("workspace-setup")).not.toBeInTheDocument());
    expect(stopRecording).toHaveBeenCalledTimes(1);
  });

  it("stops when completion skips the initial import and navigates away", async () => {
    const user = userEvent.setup();
    render(onboardingHarness());
    await user.click(
      await screen.findByRole("switch", { name: FIRST_RUN_COPY.tickets.importExisting }, { timeout: 8000 }),
    );
    await user.click(screen.getByTestId("first-run-analyze-tickets"));
    await user.click(await screen.findByRole("button", { name: "Connect Anthropic" }));
    await waitFor(() => expect(screen.getByRole("combobox", { name: "Model" })).toBeEnabled());
    await user.click(screen.getByTestId("first-run-finish-setup"));
    await waitFor(() => expect(screen.queryByTestId("workspace-setup")).not.toBeInTheDocument());
    expect(startRecording).toHaveBeenCalledTimes(1);
    expect(stopRecording).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("first-run-analysis")).not.toBeInTheDocument();
  });

  it("stops when switching workspaces and starts again when returning to onboarding", async () => {
    const user = userEvent.setup();
    const fixture = setupFixture();
    const otherWorkspace = fixture.factories.find((factory) => factory.id !== PRIMARY_FACTORY_ID)!;
    render(onboardingHarness());
    await user.click(await screen.findByTestId("first-run-workspace-switch", {}, { timeout: 8000 }));
    await user.click(screen.getByTestId(`first-run-workspace-option-${otherWorkspace.id}`));
    await screen.findByRole("link", { name: "Return to onboarding" });
    expect(stopRecording).toHaveBeenCalledTimes(1);
    expect(startRecording).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole("link", { name: "Return to onboarding" }));
    await screen.findByTestId("workspace-setup");
    expect(startRecording).toHaveBeenCalledTimes(2);
  });

  it("stops on document exit, resumes a cached page, and removes listeners on unmount", async () => {
    const view = render(onboardingHarness());
    await screen.findByTestId("workspace-setup", {}, { timeout: 8000 });
    dispatchPageTransition("pageshow");
    expect(startRecording).toHaveBeenCalledTimes(1);
    dispatchPageTransition("pagehide", true);
    expect(stopRecording).toHaveBeenCalledTimes(1);
    dispatchPageTransition("pageshow", true);
    expect(startRecording).toHaveBeenCalledTimes(2);
    dispatchPageTransition("pagehide");
    expect(stopRecording).toHaveBeenCalledTimes(2);

    view.unmount();
    expect(stopRecording).toHaveBeenCalledTimes(3);
    dispatchPageTransition("pageshow", true);
    dispatchPageTransition("pagehide");
    expect(startRecording).toHaveBeenCalledTimes(2);
    expect(stopRecording).toHaveBeenCalledTimes(3);
  });

  it("leaves recording active after Strict Mode setup and stopped after final unmount", async () => {
    const view = render(<StrictMode>{onboardingHarness()}</StrictMode>);
    await screen.findByTestId("workspace-setup", {}, { timeout: 8000 });
    expect(startRecording).toHaveBeenCalled();
    expect(startRecording.mock.invocationCallOrder.at(-1)).toBeGreaterThan(
      stopRecording.mock.invocationCallOrder.at(-1) ?? 0,
    );
    view.unmount();
    expect(stopRecording.mock.invocationCallOrder.at(-1)).toBeGreaterThan(
      startRecording.mock.invocationCallOrder.at(-1)!,
    );
  });

  it("does not control recording when analytics is disabled", async () => {
    delete analyticsWindow.SUPERPLANE_POSTHOG_KEY;
    const view = render(onboardingHarness());
    await screen.findByTestId("workspace-setup", {}, { timeout: 8000 });
    dispatchPageTransition("pagehide");
    dispatchPageTransition("pageshow", true);
    view.unmount();
    expect(startRecording).not.toHaveBeenCalled();
    expect(stopRecording).not.toHaveBeenCalled();
  });

  it("does not start recording on normal workspace pages", async () => {
    render(onboardingHarness("workspaces"));
    await screen.findByRole("link", { name: "Return to onboarding" }, { timeout: 8000 });
    expect(startRecording).not.toHaveBeenCalled();
    expect(stopRecording).not.toHaveBeenCalled();
  });
});
