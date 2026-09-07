import type { ConfigurationField } from "@/api-client";
import { describe, expect, it } from "vitest";
import { configurationSubmitPayload, editableConfigurationValue, nextConfigurationValue } from "./legacyConfiguration";

describe("legacy integration configuration", () => {
  const sensitiveField: ConfigurationField = {
    name: "apiKey",
    type: "string",
    sensitive: true,
  };
  const togglableSensitiveField: ConfigurationField = {
    name: "adminKey",
    type: "string",
    sensitive: true,
    togglable: true,
  };

  it("shows a stored sensitive value as an empty replacement field", () => {
    expect(editableConfigurationValue(sensitiveField, "<redacted>")).toBeUndefined();
  });

  it("keeps a togglable stored secret enabled until it is replaced", () => {
    expect(editableConfigurationValue(togglableSensitiveField, "<redacted>")).toBe("");
  });

  it("keeps a stored sensitive value when a replacement is cleared", () => {
    expect(nextConfigurationValue(sensitiveField, "<redacted>", undefined)).toBe("<redacted>");
  });

  it("keeps a stored sensitive value when a togglable field is enabled without a replacement", () => {
    expect(nextConfigurationValue(togglableSensitiveField, "<redacted>", "")).toBe("<redacted>");
  });

  it("turns a togglable stored secret off without turning the switch back on", () => {
    const stored = nextConfigurationValue(togglableSensitiveField, "<redacted>", null);

    expect(stored).toBeNull();
    expect(editableConfigurationValue(togglableSensitiveField, stored)).toBeNull();
  });

  it("turns a newly enabled togglable secret off", () => {
    const stored = nextConfigurationValue(togglableSensitiveField, "", null);

    expect(stored).toBeNull();
    expect(editableConfigurationValue(togglableSensitiveField, stored)).toBeNull();
  });

  it("sends an empty string so a toggled-off secret is cleared", () => {
    expect(
      configurationSubmitPayload([togglableSensitiveField], {
        apiKey: "<redacted>",
        adminKey: null,
      }),
    ).toEqual({
      apiKey: "<redacted>",
      adminKey: "",
    });
  });

  it("uses a replacement sensitive value", () => {
    expect(nextConfigurationValue(sensitiveField, "<redacted>", "new-key")).toBe("new-key");
  });

  it("does not change non-sensitive values", () => {
    const field: ConfigurationField = { name: "region", type: "string" };

    expect(editableConfigurationValue(field, "us-east")).toBe("us-east");
    expect(nextConfigurationValue(field, "us-east", undefined)).toBeUndefined();
  });
});
