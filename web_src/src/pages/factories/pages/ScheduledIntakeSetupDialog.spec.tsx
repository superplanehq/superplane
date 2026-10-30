import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ScheduledIntakeSetupDialog } from "./ScheduledIntakeSetupDialog";
import { addIntakeTemplatesForOrg } from "./addIntakeTemplates";
import { FEATURE_FACTORY_SCHEDULED_INTAKE } from "@/lib/experimentalFeatures";

const mocks = vi.hoisted(() => ({
  subscription: false,
  enabled: true,
  create: vi.fn(),
  previewError: null as Error | null,
}));
vi.mock("../layout/factoriesLayoutContext", () => ({
  useOptionalFactoriesLayout: () => ({ factory: { onboarding: { codexSubscription: mocks.subscription } } }),
}));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useFactoryOwnerEligibility", () => ({
  useFactoryOwnerEligibility: () => ({
    data: { members: [{ userId: "alice", eligible: true }] },
    isPending: false,
    isError: false,
  }),
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => mocks.enabled }) }));
vi.mock("@/hooks/useFactoryData", () => ({
  useCreateFactoryAutomation: () => ({ mutateAsync: mocks.create, isPending: false }),
  useFactorySchedulePreview: () => ({
    data: new Date(Date.now() + 86_400_000).toISOString(),
    isFuture: true,
    error: mocks.previewError,
    isFetching: false,
    isPending: false,
    isSuccess: !mocks.previewError,
    refetch: vi.fn(),
  }),
}));

beforeEach(() => {
  mocks.subscription = false;
  mocks.enabled = true;
  mocks.previewError = null;
  mocks.create.mockReset();
  mocks.create.mockResolvedValue({ id: "new-automation" });
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});
describe("Scheduled intake", () => {
  it("hides the picker entry and setup when its feature is off", () => {
    mocks.enabled = false;
    expect(addIntakeTemplatesForOrg(() => false).some((entry) => entry.id === "scheduled-intake")).toBe(false);
    expect(
      addIntakeTemplatesForOrg((id) => id === FEATURE_FACTORY_SCHEDULED_INTAKE).some(
        (entry) => entry.id === "scheduled-intake",
      ),
    ).toBe(true);
    render(
      <ScheduledIntakeSetupDialog organizationId="org" factoryId="factory" onClose={vi.fn()} onCreated={vi.fn()} />,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
  it("passes the required subscription owner through install parameters", async () => {
    mocks.subscription = true;
    const user = userEvent.setup();
    render(
      <ScheduledIntakeSetupDialog organizationId="org" factoryId="factory" onClose={vi.fn()} onCreated={vi.fn()} />,
    );
    expect(screen.getByRole("button", { name: "Create intake" })).toBeDisabled();
    await user.click(screen.getByRole("combobox", { name: /Default task owner/ }));
    await user.click(await screen.findByRole("option", { name: "Alice" }));
    await user.click(screen.getByRole("button", { name: "Create intake" }));
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({ installParams: expect.objectContaining({ defaultOwnerId: "alice" }) }),
      ),
    );
  });
  it("submits schedule and prompt settings and reports the created automation", async () => {
    vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(-330);
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ScheduledIntakeSetupDialog organizationId="org" factoryId="factory" onClose={vi.fn()} onCreated={onCreated} />,
    );
    expect(screen.queryByLabelText("UTC offset")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Next run:");
    await user.clear(screen.getByLabelText("What should the agent look for?"));
    await user.type(screen.getByLabelText("What should the agent look for?"), "Review AGENTS.md");
    await user.click(screen.getByRole("button", { name: "Create intake" }));
    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("new-automation"));
    expect(mocks.create).toHaveBeenCalledWith({
      name: "Scheduled intake",
      templateId: "scheduled-intake",
      installParams: { type: "days", hour: "09", minute: "00", timezone: "5.5", prompt: "Review AGENTS.md" },
    });
  });
  it("keeps the settings and shows a recoverable creation error", async () => {
    mocks.create.mockRejectedValue(new Error("Could not create intake"));
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ScheduledIntakeSetupDialog organizationId="org" factoryId="factory" onClose={vi.fn()} onCreated={onCreated} />,
    );
    await user.click(screen.getByRole("button", { name: "Create intake" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not create intake");
    expect(onCreated).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Name")).toHaveValue("Scheduled intake");
  });
  it("switches to cron, submits the expression, and preserves the daily time", async () => {
    vi.useFakeTimers();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(
      <ScheduledIntakeSetupDialog organizationId="org" factoryId="factory" onClose={vi.fn()} onCreated={vi.fn()} />,
    );
    await user.click(screen.getByRole("tab", { name: "Cron expression" }));
    await user.clear(screen.getByLabelText("Cron expression", { exact: true }));
    await user.type(screen.getByLabelText("Cron expression", { exact: true }), "*/10 * * * * *");
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(screen.getByRole("button", { name: "Create intake" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Create intake" }));
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({
        installParams: expect.objectContaining({ type: "cron", cronExpression: "*/10 * * * * *" }),
      }),
    );
    await user.click(screen.getByRole("tab", { name: "Daily" }));
    expect(screen.getByLabelText("Time")).toHaveValue("09:00");
  });
  it("blocks creation when the server cannot validate the schedule", () => {
    mocks.previewError = new Error("Invalid cron expression");
    render(
      <ScheduledIntakeSetupDialog organizationId="org" factoryId="factory" onClose={vi.fn()} onCreated={vi.fn()} />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Invalid cron expression");
    expect(screen.getByRole("button", { name: "Create intake" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Retry preview" })).toBeEnabled();
  });
});
