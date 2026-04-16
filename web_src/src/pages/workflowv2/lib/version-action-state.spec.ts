import { describe, expect, it } from "vitest";
import { getVersionActionAvailability } from "./version-action-state";

describe("getVersionActionAvailability", () => {
  it("keeps publish enabled while local draft changes are still being saved", () => {
    const result = getVersionActionAvailability({
      isChangeManagementDisabled: true,
      hasEditableVersion: true,
      createChangeRequestPending: false,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("keeps publish enabled after local save activity has settled but draft changes are still pending", () => {
    const result = getVersionActionAvailability({
      isChangeManagementDisabled: true,
      hasEditableVersion: true,
      createChangeRequestPending: false,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("keeps change request creation enabled when draft changes are pending locally", () => {
    const result = getVersionActionAvailability({
      isChangeManagementDisabled: false,
      hasEditableVersion: true,
      createChangeRequestPending: false,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
    });

    expect(result.createChangeRequestDisabled).toBe(false);
    expect(result.createChangeRequestDisabledTooltip).toBeUndefined();
    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });
});
