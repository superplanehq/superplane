import type { ConfigurationField } from "@/api-client";
import { describe, expect, it } from "vitest";

import { selectCreateStepFields, selectVisibleFields, selectWebhookStepFields } from "./configurationFields";

const apiKey: ConfigurationField = { name: "apiKey", type: "string", required: true };
const adminKey: ConfigurationField = { name: "adminKey", type: "string", required: false, togglable: true };
const signingSecret: ConfigurationField = { name: "signingSecret", type: "string", required: false };
const unnamed: ConfigurationField = { type: "string" };

describe("selectVisibleFields", () => {
  it("keeps every named field when nothing is hidden", () => {
    expect(selectVisibleFields([apiKey, adminKey], [])).toEqual([apiKey, adminKey]);
  });

  it("drops hidden fields so onboarding never renders them", () => {
    expect(selectVisibleFields([apiKey, adminKey], ["adminKey"])).toEqual([apiKey]);
  });

  it("drops fields without a name", () => {
    expect(selectVisibleFields([apiKey, unnamed], [])).toEqual([apiKey]);
  });
});

describe("selectCreateStepFields", () => {
  it("uses every visible field when the steps are not split", () => {
    expect(selectCreateStepFields([apiKey, signingSecret], undefined)).toEqual([apiKey, signingSecret]);
  });

  it("keeps only the requested fields when the steps are split", () => {
    expect(selectCreateStepFields([apiKey, signingSecret], ["apiKey"])).toEqual([apiKey]);
  });
});

describe("selectWebhookStepFields", () => {
  it("shows the webhook secrets when the steps are not split", () => {
    expect(selectWebhookStepFields([apiKey, signingSecret], undefined)).toEqual([signingSecret]);
  });

  it("shows the remaining fields when the steps are split", () => {
    expect(selectWebhookStepFields([apiKey, signingSecret], ["apiKey"])).toEqual([signingSecret]);
  });

  it("never shows a hidden field, because it is not visible", () => {
    const visible = selectVisibleFields([apiKey, adminKey], ["adminKey"]);
    expect(selectWebhookStepFields(visible, ["apiKey"])).toEqual([]);
  });
});
