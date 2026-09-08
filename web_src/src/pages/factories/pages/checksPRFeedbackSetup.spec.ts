import { describe, expect, it } from "vitest";

import {
  catalogStatusCheckNames,
  requiredStatusCheckNames,
  suggestedIntegrationsForChecks,
} from "./checksPRFeedbackSetup";

describe("checksPRFeedbackSetup", () => {
  it("collects required check names", () => {
    expect(
      requiredStatusCheckNames([
        { name: "lint", required: true },
        { name: "unit", required: false },
        { required: true },
      ]),
    ).toEqual(["lint"]);
  });

  it("collects every catalog check name", () => {
    expect(
      catalogStatusCheckNames([
        { name: "lint", required: true },
        { name: "unit", required: false },
        { name: "  ", required: false },
        { required: true },
      ]),
    ).toEqual(["lint", "unit"]);
  });

  it("suggests integrations only for selected checks", () => {
    expect(
      suggestedIntegrationsForChecks(
        [
          { name: "lint", suggestedIntegration: "semaphore" },
          { name: "e2e", suggestedIntegration: "circleci" },
          { name: "actions", suggestedIntegration: "github" },
        ],
        ["lint", "actions"],
      ),
    ).toEqual(["semaphore"]);
  });
});
