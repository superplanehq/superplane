import { describe, expect, it } from "vitest";
import { getVersionActionAvailability } from "./version-action-state";

describe("getVersionActionAvailability", () => {
  it("keeps publish enabled while local draft changes are still being saved", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("keeps publish enabled after local save activity has settled but draft changes are still pending", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("disables publish while publish is pending", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: true,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(true);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("disables publish when the latest draft matches live", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: false,
    });

    expect(result.publishVersionDisabled).toBe(true);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });
});
