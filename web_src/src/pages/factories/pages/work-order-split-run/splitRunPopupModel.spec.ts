import { describe, expect, it } from "vitest";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { LINE_BOARD_DONE_RECEIPTS_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import {
  collectSplitRunArtifacts,
  defaultSplitRunPopupTab,
  resolveSplitRunPopupArtifacts,
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

  it("uses live artifacts for a real work order and fixture artifacts in Storybook", () => {
    const fixtureArtifacts = collectSplitRunArtifacts(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER));
    const liveArtifacts = [
      {
        id: "art-live-pr",
        type: "TYPE_PR" as const,
        data: { number: 88, url: "https://github.com/acme/app/pull/88" },
      },
    ];

    expect(resolveSplitRunPopupArtifacts({ fixtureArtifacts, liveArtifacts, useLive: true })).toEqual(liveArtifacts);
    expect(resolveSplitRunPopupArtifacts({ fixtureArtifacts, liveArtifacts, useLive: false })).toEqual(
      fixtureArtifacts,
    );
  });

  it("lists description artifacts oldest first", () => {
    const artifacts = splitRunLinkedArtifacts([
      {
        id: "newer",
        type: "TYPE_PR",
        createdAt: "2026-08-25T12:00:00.000Z",
        data: { number: 2 },
      },
      {
        id: "older",
        type: "TYPE_BRANCH",
        createdAt: "2026-08-25T10:00:00.000Z",
        data: { name: "feature/a" },
      },
      {
        id: "description",
        type: "TYPE_MARKDOWN",
        createdAt: "2026-08-25T09:00:00.000Z",
        data: { name: "description.md" },
      },
      { id: "undated", type: "TYPE_LINK", data: { title: "late" } },
    ]);

    expect(artifacts.map((artifact) => artifact.id)).toEqual(["older", "newer", "undated"]);
  });
});
