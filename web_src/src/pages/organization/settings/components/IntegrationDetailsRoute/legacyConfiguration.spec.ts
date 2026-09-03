import type { ConfigurationField } from "@/api-client";
import { editableConfigurationValue, nextConfigurationValue } from "./legacyConfiguration";

describe("legacy integration configuration", () => {
  const sensitiveField: ConfigurationField = {
    name: "apiKey",
    type: "string",
    sensitive: true,
  };

  it("shows a stored sensitive value as an empty replacement field", () => {
    expect(editableConfigurationValue(sensitiveField, "<redacted>")).toBeUndefined();
  });

  it("keeps a stored sensitive value when a replacement is cleared", () => {
    expect(nextConfigurationValue(sensitiveField, "<redacted>", undefined)).toBe("<redacted>");
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
