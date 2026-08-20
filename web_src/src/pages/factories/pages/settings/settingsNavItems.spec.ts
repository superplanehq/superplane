import { describe, expect, it } from "vitest";

import { isYouSettingsSection, settingsSectionFromPathname } from "./settingsNavItems";

describe("settingsSectionFromPathname", () => {
  it("reads the section after /settings/", () => {
    expect(settingsSectionFromPathname("/org/workspaces/RF/settings/profile")).toBe("profile");
    expect(settingsSectionFromPathname("/org/workspaces/RF/settings/general")).toBe("general");
    expect(settingsSectionFromPathname("/org/workspaces/RF/settings/notifications")).toBe("notifications");
  });

  it("returns undefined when the path is not a known settings section", () => {
    expect(settingsSectionFromPathname("/org/workspaces/RF/overview")).toBeUndefined();
    expect(settingsSectionFromPathname("/org/workspaces/RF/settings")).toBeUndefined();
    expect(settingsSectionFromPathname("/org/workspaces/RF/settings/unknown")).toBeUndefined();
  });
});

describe("isYouSettingsSection", () => {
  it("is true for profile and notifications", () => {
    expect(isYouSettingsSection("profile")).toBe(true);
    expect(isYouSettingsSection("notifications")).toBe(true);
  });

  it("is false for workspace settings and missing sections", () => {
    expect(isYouSettingsSection("general")).toBe(false);
    expect(isYouSettingsSection("members")).toBe(false);
    expect(isYouSettingsSection(undefined)).toBe(false);
  });
});
