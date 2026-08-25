import { describe, expect, it } from "vitest";

import {
  buildSplitRunFooter,
  DEFAULT_SPLIT_RUN_STOP_CHOICE,
  doneFooterForStatus,
  SPLIT_RUN_STOP_CHOICES,
} from "./splitRunFooter";

const PR_NOTE = {
  key: "pr",
  headline: "Review the pull request",
  text: "Merge #6812 to continue Verify.",
  cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
  source: { name: "PR Closure" },
};

const FAILED_NOTE = {
  key: "implement-failed",
  headline: "Implement did not pass",
  text: "Backend tests failed on the reconciliation worker.",
  cta: { label: "Open failed run", href: "https://superplanehq.semaphoreci.com/" },
  source: { name: "Implementation" },
};

const DRAFT_NOTE = {
  key: "draft-plan-ready",
  headline: "Review the plan, then start",
  text: "From GitHub issue PAY-842. Confidence 5/5.",
};

describe("buildSplitRunFooter", () => {
  it("keeps a draft note above the state bar with Start and Reject", () => {
    expect(buildSplitRunFooter({ kind: "draft", note: DRAFT_NOTE })).toEqual({
      kind: "draft",
      sentence: "This work order is a draft.",
      note: { headline: "Review the plan, then start", text: "From GitHub issue PAY-842. Confidence 5/5." },
      actions: [
        { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" },
        { id: "start", kind: "start", label: "Start", emphasis: "primary" },
      ],
    });
  });

  it("keeps a running order on the state bar with Stop", () => {
    const footer = buildSplitRunFooter({
      kind: "running",
      note: { key: "running-step", headline: "Implement is running", text: "The log shows live progress." },
    });

    expect(footer.sentence).toBe("This work order is running.");
    expect(footer.note?.headline).toBe("Implement is running");
    expect(footer.actions).toEqual([{ id: "stop", kind: "stop", label: "Stop", emphasis: "quiet" }]);
    expect(DEFAULT_SPLIT_RUN_STOP_CHOICE).toBe("canceled");
    expect(SPLIT_RUN_STOP_CHOICES.map((choice) => choice.label)).toEqual([
      "Stop as Canceled",
      "Stop as Completed",
      "Stop and return to Draft",
    ]);
  });

  it("keeps a waiting note off the bar and shows Stop", () => {
    const footer = buildSplitRunFooter({ kind: "waiting", note: PR_NOTE });

    expect(footer.note).toEqual({
      headline: "Review the pull request",
      text: "Merge #6812 to continue Verify.",
      sourceName: "PR Closure",
    });
    expect(footer.sentence).toBe("This work order needs attention.");
    expect(footer.actions).toEqual([{ id: "stop", kind: "stop", label: "Stop", emphasis: "quiet" }]);
  });

  it("still shows Stop when a waiting order has no run note", () => {
    expect(buildSplitRunFooter({ kind: "waiting" })).toEqual({
      kind: "waiting",
      sentence: "This work order needs attention.",
      actions: [{ id: "stop", kind: "stop", label: "Stop", emphasis: "quiet" }],
    });
  });

  it("treats a failed implement as a run note plus Stop", () => {
    const footer = buildSplitRunFooter({ kind: "failed", note: FAILED_NOTE });

    expect(footer.note?.headline).toBe("Implement did not pass");
    expect(footer.sentence).toBe("This work order needs attention.");
    expect(footer.actions).toEqual([{ id: "stop", kind: "stop", label: "Stop", emphasis: "quiet" }]);
  });

  it("shows a done summary with no actions", () => {
    expect(doneFooterForStatus("completed")).toEqual({
      kind: "done",
      sentence: "Work order completed successfully.",
      actions: [],
    });
    expect(doneFooterForStatus("rejected").sentence).toBe("A person rejected this work order.");
    expect(doneFooterForStatus("cancelled").sentence).toBe("This work order was canceled.");
    expect(doneFooterForStatus("failed").sentence).toBe("Closed as failed. Line execution did not pass.");
  });
});
