import { describe, expect, it } from "vitest";

import {
  analysisFinishedStatus,
  analysisFirstResultDelivered,
  analysisResultDeliveredForRun,
  hasAnalysisPlan,
  hasAnalysisScore,
  statusForAnalysisRun,
} from "./analysisOutcome";

describe("analysisFirstResultDelivered", () => {
  it("is true when a score and spec.md exist", () => {
    expect(
      analysisFirstResultDelivered({
        checks: [{ name: "Confidence score", key: "confidence", score: 4 }],
        artifacts: [{ data: { name: "spec.md", body: "# Add breed\n\n## Executive summary\n\nAdd breed.\n" } }],
      }),
    ).toBe(true);
  });

  it("is false when the score is missing", () => {
    expect(
      analysisFirstResultDelivered({
        checks: [],
        artifacts: [{ data: { name: "spec.md", body: "# Add breed\n" } }],
      }),
    ).toBe(false);
  });

  it("is false when the plan is missing", () => {
    expect(
      analysisFirstResultDelivered({
        checks: [{ name: "Confidence score", score: 3 }],
        artifacts: [],
      }),
    ).toBe(false);
  });

  it("accepts a score of 0", () => {
    expect(hasAnalysisScore([{ key: "confidence", score: 0 }])).toBe(true);
  });

  it("accepts intent.md as the plan", () => {
    expect(hasAnalysisPlan([{ data: { title: "intent.md", body: "A retry loop." } }])).toBe(true);
  });
});

describe("analysisFinishedStatus", () => {
  it("keeps a cancelled or failed stop as passed after the first result", () => {
    expect(analysisFinishedStatus("cancelled", true)).toBe("passed");
    expect(analysisFinishedStatus("failed", true)).toBe("passed");
  });

  it("leaves a true failure in place when no first result exists", () => {
    expect(analysisFinishedStatus("failed", false)).toBe("failed");
    expect(analysisFinishedStatus("cancelled", false)).toBe("cancelled");
  });

  it("does not change a running analysis", () => {
    expect(analysisFinishedStatus("running", true)).toBe("running");
  });
});

describe("analysisResultDeliveredForRun", () => {
  const scoreAndPlan = {
    checks: [{ name: "Confidence score", key: "confidence", score: 4, runId: "run-b" }],
    artifacts: [{ data: { name: "spec.md", body: "# Add breed\n\n## Executive summary\n\nAdd breed.\n" } }],
  };

  it("attributes the first result to the run that reported the score", () => {
    expect(analysisResultDeliveredForRun({ id: "run-b" }, { ...scoreAndPlan, isLast: true })).toBe(true);
    expect(analysisResultDeliveredForRun({ id: "run-a" }, { ...scoreAndPlan, isLast: false })).toBe(false);
  });

  it("lets the last run inherit an unattributed first result", () => {
    expect(
      analysisResultDeliveredForRun(
        { id: "run-a" },
        {
          checks: [{ name: "Confidence score", key: "confidence", score: 4 }],
          artifacts: [{ data: { name: "spec.md", body: "# Add breed\n" } }],
          isLast: true,
        },
      ),
    ).toBe(true);
  });

  it("does not let a historical run inherit an unattributed first result", () => {
    expect(
      analysisResultDeliveredForRun(
        { id: "run-a" },
        {
          checks: [{ name: "Confidence score", key: "confidence", score: 4 }],
          artifacts: [{ data: { name: "spec.md", body: "# Add breed\n" } }],
          isLast: false,
        },
      ),
    ).toBe(false);
  });
});

describe("statusForAnalysisRun", () => {
  it("keeps a cancelled run running when no first result exists", () => {
    expect(statusForAnalysisRun({ result: "RESULT_CANCELLED" }, "failed", false)).toBe("running");
  });

  it("keeps a failed run failed when no first result exists", () => {
    expect(statusForAnalysisRun({ result: "RESULT_FAILED" }, "failed", false)).toBe("failed");
  });

  it("marks a cancelled run passed when a first result exists", () => {
    expect(statusForAnalysisRun({ result: "RESULT_CANCELLED" }, "failed", true)).toBe("passed");
  });

  it("does not change a started analysis", () => {
    expect(statusForAnalysisRun({ state: "STATE_STARTED" }, "running", false)).toBe("running");
  });

  it("keeps a cancelling timeout running when no first result exists", () => {
    expect(statusForAnalysisRun({ state: "STATE_CANCELLING" }, "failed", false)).toBe("running");
  });

  it("keeps an explicitly stopped run failed when no first result exists", () => {
    expect(statusForAnalysisRun({ result: "RESULT_CANCELLED", cancelledBy: { id: "user-1" } }, "failed", false)).toBe(
      "failed",
    );
  });

  it("keeps an explicitly stopping run failed when no first result exists", () => {
    expect(statusForAnalysisRun({ state: "STATE_CANCELLING", cancelledBy: { id: "user-1" } }, "failed", false)).toBe(
      "failed",
    );
  });
});
