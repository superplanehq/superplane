import { describe, expect, it } from "vitest";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";

describe("integrationUtils", () => {
  it("extracts the webhook url from metadata", () => {
    expect(getIntegrationWebhookUrl({ webhookUrl: "https://example.com/webhook" })).toBe("https://example.com/webhook");
  });

  it("returns undefined when the webhook url is missing or not a string", () => {
    expect(getIntegrationWebhookUrl({})).toBeUndefined();
    expect(getIntegrationWebhookUrl({ webhookUrl: 123 })).toBeUndefined();
    expect(getIntegrationWebhookUrl(undefined)).toBeUndefined();
  });
});
