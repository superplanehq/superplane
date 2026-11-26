import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { DEFAULT_GITHUB_INTAKE_SETTINGS, intakeSettingsToApi } from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { useIntakeWebhookRetry } from "./useIntakeWebhookRetry";

const updateIntakeMutate = vi.fn();

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useUpdateFactoryIntake: () => ({ mutate: updateIntakeMutate, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

const GITHUB_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-github",
  appId: "app-github-issues-intake",
  healthy: false,
  paused: false,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, labels: ["bug"] },
  source: lineIntakeSourceById("github-issues")!,
  health: "HEALTH_WEBHOOK_FAILED",
  healthMessage: "GitHub did not create the webhook.",
  resourceId: "acme/app",
};

beforeEach(() => {
  updateIntakeMutate.mockReset();
});

afterEach(() => {
  cleanup();
});

describe("useIntakeWebhookRetry", () => {
  it("saves the intake settings to start webhook registration again", () => {
    const { result } = renderHook(() => useIntakeWebhookRetry("org-1", "factory-1"));

    act(() => {
      result.current.retryIntake(GITHUB_INTAKE);
    });

    expect(updateIntakeMutate).toHaveBeenCalledWith(
      {
        intakeId: "intake-github",
        settings: intakeSettingsToApi(GITHUB_INTAKE.settings, "github-issues"),
      },
      expect.objectContaining({ onError: expect.any(Function) }),
    );
  });

  it("tells the user when the retry request fails", async () => {
    const { showErrorToast } = await import("@/lib/toast");
    updateIntakeMutate.mockImplementation((_variables: unknown, options: { onError: () => void }) => {
      options.onError();
    });
    const { result } = renderHook(() => useIntakeWebhookRetry("org-1", "factory-1"));

    act(() => {
      result.current.retryIntake(GITHUB_INTAKE);
    });

    expect(showErrorToast).toHaveBeenCalledWith("SuperPlane could not retry the webhook setup. Try again.");
  });
});
