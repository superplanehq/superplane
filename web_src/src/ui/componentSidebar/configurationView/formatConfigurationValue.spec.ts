import { describe, expect, it } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { EMPTY_DISPLAY_VALUE, formatConfigurationValue } from "./formatConfigurationValue";

describe("formatConfigurationValue", () => {
  it("returns empty display for unset values", () => {
    const field: ConfigurationField = { name: "serviceName", label: "Service name", type: "string" };
    expect(formatConfigurationValue(field, undefined)).toEqual({
      kind: "empty",
      displayText: EMPTY_DISPLAY_VALUE,
    });
  });

  it("formats boolean values as Yes/No", () => {
    const field: ConfigurationField = { name: "enabled", label: "Enabled", type: "boolean" };
    expect(formatConfigurationValue(field, true).displayText).toBe("Yes");
    expect(formatConfigurationValue(field, false).displayText).toBe("No");
  });

  it("resolves select option labels", () => {
    const field: ConfigurationField = {
      name: "environment",
      label: "Environment",
      type: "select",
      typeOptions: {
        select: {
          options: [
            { label: "Development", value: "development" },
            { label: "Production", value: "production" },
          ],
        },
      },
    };
    expect(formatConfigurationValue(field, "production")).toEqual({
      kind: "text",
      displayText: "Production",
    });
  });

  it("formats multi-select values as chips", () => {
    const field: ConfigurationField = {
      name: "channels",
      label: "Channels",
      type: "multi-select",
      typeOptions: {
        multiSelect: {
          options: [
            { label: "Slack", value: "slack" },
            { label: "Webhook", value: "webhook" },
          ],
        },
      },
    };
    expect(formatConfigurationValue(field, ["slack", "webhook"])).toEqual({
      kind: "list",
      displayText: "Slack, Webhook",
      chips: ["Slack", "Webhook"],
    });
  });

  it("formats integration references by installation name", () => {
    const field: ConfigurationField = { name: "integration", label: "Integration", type: "integration" };
    expect(formatConfigurationValue(field, { name: "GitHub Production" }).displayText).toBe("GitHub Production");
  });

  it("formats secret references", () => {
    const field: ConfigurationField = { name: "secret", label: "Secret", type: "secret" };
    expect(formatConfigurationValue(field, { secret: "some-other-secret" }).displayText).toBe("some-other-secret");
  });

  it("masks secret-key references", () => {
    const field: ConfigurationField = { name: "credential", label: "Credential", type: "secret-key" };
    expect(formatConfigurationValue(field, { secret: "prod", key: "api-token" }).displayText).toBe(
      "secret:prod / api-token",
    );
  });

  it("masks sensitive string fields", () => {
    const field: ConfigurationField = {
      name: "token",
      label: "Token",
      type: "string",
      sensitive: true,
    };
    expect(formatConfigurationValue(field, "sp_live_token").displayText).toBe("••••••");
  });

  it("marks url fields as links", () => {
    const field: ConfigurationField = { name: "endpoint", label: "Endpoint", type: "url" };
    const formatted = formatConfigurationValue(field, "https://api.example.com/hook");
    expect(formatted.kind).toBe("url");
    expect(formatted.href).toBe("https://api.example.com/hook");
  });

  it("does not link url fields with non-http(s) values", () => {
    const field: ConfigurationField = { name: "endpoint", label: "Endpoint", type: "url" };
    expect(formatConfigurationValue(field, "javascript:alert(1)")).toEqual({
      kind: "text",
      displayText: "javascript:alert(1)",
    });
    expect(formatConfigurationValue(field, "ftp://files.example.com/data")).toEqual({
      kind: "text",
      displayText: "ftp://files.example.com/data",
    });
  });

  it("uses monospace kind for expressions", () => {
    const field: ConfigurationField = { name: "filter", label: "Filter", type: "expression" };
    expect(formatConfigurationValue(field, '$["trigger"].payload.id').kind).toBe("expression");
  });

  it("does not treat expression values as urls", () => {
    const field: ConfigurationField = { name: "endpoint", label: "Endpoint", type: "expression" };
    expect(formatConfigurationValue(field, "https://api.example.com/hook")).toEqual({
      kind: "expression",
      displayText: "https://api.example.com/hook",
    });
  });
});
