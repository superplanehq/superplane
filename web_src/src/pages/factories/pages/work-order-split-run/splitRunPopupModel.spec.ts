import { describe, expect, it } from "vitest";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { LINE_BOARD_DONE_RECEIPTS_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import {
  collectSplitRunArtifacts,
  defaultSplitRunPopupTab,
  SPLIT_RUN_PANE_GRID_CLASSNAME,
  splitRunDescriptionMarkdown,
  splitRunLinkedArtifacts,
  splitRunLogTabDotClass,
} from "./splitRunPopupModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

describe("splitRunPopupModel", () => {
  it("uses a 3/2 pane split for Description and Log", () => {
    expect(SPLIT_RUN_PANE_GRID_CLASSNAME).toContain("minmax(0,3fr)_minmax(0,2fr)");
  });

  it("opens the description tab for drafts and done cards, and the log for later states", () => {
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER))).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER))).toBe("description");
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
    expect(splitRunLinkedArtifacts(artifacts).some((artifact) => artifact.id?.endsWith("-details"))).toBe(false);
    expect(splitRunLinkedArtifacts(artifacts).some((artifact) => artifact.id?.endsWith("-plan"))).toBe(true);
  });
});
