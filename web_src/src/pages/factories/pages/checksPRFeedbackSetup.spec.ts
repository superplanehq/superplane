import { describe, expect, it } from "vitest";

import {
  catalogStatusCheckNames,
  checksHandlerIntegrationRows,
  hasSelectedSuggestedIntegration,
  isChecksHandlerCIIntegration,
  readyChecksHandlerIntegrationIds,
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

  it("keeps only common status-check integrations", () => {
    expect(isChecksHandlerCIIntegration("Semaphore")).toBe(true);
    expect(isChecksHandlerCIIntegration("slack")).toBe(false);
    expect(suggestedIntegrationsForChecks([{ name: "lint", suggestedIntegration: "jenkins" }], ["lint"])).toEqual([]);
  });

  it("lists connected instances as selectable rows and types without a connection", () => {
    expect(
      checksHandlerIntegrationRows(
        [
          { name: "circleci", label: "CircleCI" },
          { name: "semaphore", label: "Semaphore" },
        ],
        [
          {
            metadata: { id: "int-cci", name: "circleci-prod", integrationName: "circleci" },
          },
        ],
      ),
    ).toEqual([
      { type: "circleci", displayName: "circleci-prod", instanceId: "int-cci" },
      { type: "semaphore", displayName: "Semaphore" },
    ]);
  });

  it("collects ready CI integration ids", () => {
    expect(
      readyChecksHandlerIntegrationIds([
        { metadata: { id: "int-cci", integrationName: "circleci" }, status: { state: "ready" } },
        { metadata: { id: "int-slack", integrationName: "slack" }, status: { state: "ready" } },
        { metadata: { id: "int-sem", integrationName: "semaphore" }, status: { state: "broken" } },
      ]),
    ).toEqual(["int-cci"]);
  });

  it("detects when a suggested integration is selected", () => {
    const ready = [{ metadata: { id: "int-cci", integrationName: "circleci" } }];
    expect(hasSelectedSuggestedIntegration(["circleci"], ["int-cci"], ready)).toBe(true);
    expect(hasSelectedSuggestedIntegration(["circleci"], [], ready)).toBe(false);
    expect(hasSelectedSuggestedIntegration([], ["int-cci"], ready)).toBe(false);
  });
});
