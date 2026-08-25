import { describe, expect, it } from "vitest";

import { afterOnboardingPath } from "./useFinishOnboarding";

describe("afterOnboardingPath", () => {
  it("opens the intake drawer on the provisioned line", () => {
    expect(
      afterOnboardingPath({
        organizationId: "org-1",
        factoryKey: "SP",
        lineId: "line-1",
      }),
    ).toBe("/org-1/workspaces/SP/lines/line-1?intake=1");
  });

  it("expands the GitHub intake chevron", () => {
    expect(
      afterOnboardingPath({
        organizationId: "org-1",
        factoryKey: "SP",
        lineId: "line-1",
        githubIntakeId: "intake-github",
      }),
    ).toBe("/org-1/workspaces/SP/lines/line-1?intake=1&intakeId=intake-github");
  });
});
