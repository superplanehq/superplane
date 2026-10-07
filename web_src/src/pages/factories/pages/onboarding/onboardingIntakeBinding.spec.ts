import { beforeEach, describe, expect, it } from "bun:test";

import {
  forgetOnboardingIntakeBinding,
  onboardingIntakeBinding,
  rememberOnboardingIntakeBinding,
} from "./onboardingIntakeBinding";

describe("onboardingIntakeBinding", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("reads integration and project from a listed intake", () => {
    expect(
      onboardingIntakeBinding({
        id: "intake-1",
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "jira-1",
        resourceId: "PAY",
      } as never),
    ).toEqual({ integrationId: "jira-1", resourceId: "PAY" });
  });

  it("remembers the binding of an intake this session created", () => {
    rememberOnboardingIntakeBinding("intake-1", { integrationId: "jira-1", resourceId: "PAY" });

    expect(onboardingIntakeBinding({ id: "intake-1", source: "SOURCE_JIRA_ISSUES" })).toEqual({
      integrationId: "jira-1",
      resourceId: "PAY",
    });
  });

  it("forgets a binding after the intake is replaced", () => {
    rememberOnboardingIntakeBinding("intake-1", { integrationId: "jira-1", resourceId: "PAY" });
    forgetOnboardingIntakeBinding("intake-1");

    expect(onboardingIntakeBinding({ id: "intake-1", source: "SOURCE_JIRA_ISSUES" })).toEqual({});
  });
});
