import { describe, expect, it } from "bun:test";

import { OPEN_WORK_ORDER_SECONDARY, QUESTION_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { addedByForSource, splitRunSourceForOrder } from "./splitRunSource";

describe("splitRunSourceForOrder automation attribution", () => {
  it.each([
    { appName: " \t ", nodeName: " Run intake ", name: "Run intake" },
    { appName: undefined, nodeName: " ", name: "Automation" },
  ])("keeps automation attribution when its name is missing: $name", ({ appName, nodeName, name }) => {
    expect(splitRunSourceForOrder({ createdBy: { automation: { appName, nodeName } } })).toMatchObject({
      kind: "automation",
      name,
    });
  });

  it.each([undefined, { url: "https://github.com/acme/payments/issues/12" }])(
    "keeps a custom GitHub creator distinct from its origin: %j",
    (origin) => {
      const automation = {
        appId: "custom-automation",
        appName: "GitHub incident triage",
        templateId: "custom-verification",
      };
      const source = splitRunSourceForOrder({ origin, createdBy: { automation } });
      expect(addedByForSource(source)).toEqual({ kind: "automation", name: "GitHub incident triage" });
      if (!origin) {
        expect(source).toEqual({ kind: "automation", name: "GitHub incident triage" });
        return;
      }
      expect(source).toMatchObject({ kind: "intake", ticket: { href: origin.url } });
    },
  );

  it("uses the template identity of a renamed built-in intake", () => {
    const automation = { appName: "Production errors", templateId: "intake:sentry-exceptions" };
    expect(splitRunSourceForOrder({ createdBy: { automation } })).toMatchObject({
      kind: "intake",
      name: "Sentry exceptions",
    });
  });

  it("keeps a custom creator distinct from its origin ticket", () => {
    const source = splitRunSourceForOrder({
      origin: { url: "https://github.com/acme/payments/issues/12" },
      createdBy: { automation: { appId: "custom-automation", appName: "Elastic case intake" } },
    });
    expect(source).toMatchObject({
      kind: "intake",
      iconAlt: "GitHub",
      ticket: { href: "https://github.com/acme/payments/issues/12" },
    });
    expect(addedByForSource(source)).toEqual({ kind: "automation", name: "Elastic case intake" });
  });

  it("maps automation-created orders without origin to the intake source", () => {
    const sentry = splitRunSourceForOrder({ ...OPEN_WORK_ORDER_SECONDARY, origin: undefined });
    expect(sentry).toEqual(expect.objectContaining({ kind: "intake", name: "Sentry exceptions" }));
    expect(sentry).not.toHaveProperty("ticket");

    const slack = splitRunSourceForOrder({ ...QUESTION_WORK_ORDER, origin: undefined });
    expect(slack).toEqual(expect.objectContaining({ kind: "intake", name: "Slack" }));
    expect(slack).not.toHaveProperty("ticket");
  });
});
