import { describe, expect, it } from "vitest";

import {
  defaultNotificationSettings,
  notificationTypeTogglesFromSettings,
  notificationTypesFromToggles,
} from "./notificationSettings";

describe("notificationSettings", () => {
  it("defaults every known type to on", () => {
    const settings = defaultNotificationSettings();
    expect(settings.types?.every((toggle) => toggle.enabled)).toBe(true);
    expect(notificationTypeTogglesFromSettings(undefined).TYPE_WORK_ORDER_ASSIGNED).toBe(true);
  });

  it("keeps omitted types on when some toggles are saved off", () => {
    const toggles = notificationTypeTogglesFromSettings({
      types: [{ type: "TYPE_WORK_ORDER_COMMENT_OWNED", enabled: false }],
    });
    expect(toggles.TYPE_WORK_ORDER_COMMENT_OWNED).toBe(false);
    expect(toggles.TYPE_WORK_ORDER_ASSIGNED).toBe(true);
  });

  it("round-trips toggles into the API payload", () => {
    const toggles = notificationTypeTogglesFromSettings({
      types: [{ type: "TYPE_WORK_ORDER_ARTIFACT_OWNED", enabled: false }],
    });
    expect(notificationTypesFromToggles(toggles)).toContainEqual({
      type: "TYPE_WORK_ORDER_ARTIFACT_OWNED",
      enabled: false,
    });
  });
});
