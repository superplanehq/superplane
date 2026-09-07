import type { ConfigurationField } from "@/api-client";
import { describe, expect, it } from "vitest";

import {
  areRequiredCreateFieldsFilled,
  selectCreateStepFields,
  selectVisibleFields,
  selectWebhookStepFields,
} from "./configurationFields";

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

describe("areRequiredCreateFieldsFilled", () => {
  const hiddenRequired: ConfigurationField = {
    name: "token",
    type: "string",
    required: true,
    visibilityConditions: [{ field: "mode", values: ["advanced"] }],
  };

  it("rejects an empty required apiKey", () => {
    expect(areRequiredCreateFieldsFilled([apiKey, adminKey], {})).toBe(false);
  });

  it("rejects a whitespace-only apiKey", () => {
    expect(areRequiredCreateFieldsFilled([apiKey], { apiKey: "   " })).toBe(false);
  });

  it("accepts a filled apiKey", () => {
    expect(areRequiredCreateFieldsFilled([apiKey, adminKey], { apiKey: "sk-test" })).toBe(true);
  });

  it("does not require an optional togglable adminKey", () => {
    expect(areRequiredCreateFieldsFilled([apiKey, adminKey], { apiKey: "sk-test" })).toBe(true);
  });

  it("does not require a togglable field that is turned off", () => {
    const togglableRequired: ConfigurationField = {
      name: "optionalSecret",
      type: "string",
      required: true,
      togglable: true,
    };
    expect(areRequiredCreateFieldsFilled([togglableRequired], {})).toBe(true);
  });

  it("does not require a field that is not visible", () => {
    expect(areRequiredCreateFieldsFilled([hiddenRequired], {})).toBe(true);
  });

  it("requires a visible field that has visibility conditions", () => {
    expect(areRequiredCreateFieldsFilled([hiddenRequired], { mode: "advanced" })).toBe(false);
    expect(areRequiredCreateFieldsFilled([hiddenRequired], { mode: "advanced", token: "abc" })).toBe(true);
  });

  it("accepts a create step with no required fields", () => {
    expect(areRequiredCreateFieldsFilled([], {})).toBe(true);
    expect(areRequiredCreateFieldsFilled([signingSecret], {})).toBe(true);
  });
});
