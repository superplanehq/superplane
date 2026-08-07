import { describe, expect, it } from "vitest";
import { appPath, appSecretsPath, appSettingsPath } from "./appPaths";

describe("appSettingsPath", () => {
  it("builds the canvas settings path", () => {
    expect(appSettingsPath("org-1", "app-1")).toBe("/org-1/apps/app-1/settings");
  });
});

describe("appSecretsPath", () => {
  it("builds the canvas secrets path under settings", () => {
    expect(appSecretsPath("org-1", "app-1")).toBe("/org-1/apps/app-1/settings/secrets");
  });

  it("is distinct from the canvas path", () => {
    expect(appSecretsPath("org-1", "app-1")).not.toBe(appPath("org-1", "app-1"));
  });
});
