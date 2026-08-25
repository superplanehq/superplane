import { describe, expect, it } from "vitest";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import {
  collectSplitRunArtifacts,
  defaultSplitRunPopupTab,
  splitRunDescriptionMarkdown,
  splitRunLogTabDotClass,
} from "./splitRunPopupModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

describe("splitRunPopupModel", () => {
  it("opens the description tab for drafts and the log for later states", () => {
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER))).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER))).toBe("log");
  });

  it("maps log-tab dots to the line status colors", () => {
    expect(splitRunLogTabDotClass("running")).toContain("--status-running-dot");
    expect(splitRunLogTabDotClass("waiting")).toContain("--status-waiting-dot");
    expect(splitRunLogTabDotClass("failed")).toContain("--status-failed-dot");
    expect(splitRunLogTabDotClass("passed")).toContain("--status-completed-dot");
    expect(splitRunLogTabDotClass("pending")).toContain("--status-draft-dot");
  });

  it("reads details.md as the work-order description", () => {
    const fixture = splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]);
    const artifacts = collectSplitRunArtifacts(fixture);
    const description = splitRunDescriptionMarkdown(artifacts);

    expect(description).toContain("Webhook delivery stops after a transient provider error");
    expect(artifacts.some((artifact) => artifact.id?.endsWith("-plan"))).toBe(true);
  });
});
