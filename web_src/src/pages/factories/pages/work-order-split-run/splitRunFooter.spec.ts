import { describe, expect, it } from "vitest";

import {
  availableSplitRunStopChoices,
  buildSplitRunFooter,
  DEFAULT_SPLIT_RUN_STOP_CHOICE,
  defaultSplitRunStopChoice,
  doneFooterForStatus,
  rerunStartStepIndex,
  splitRunCloseNeedsConfirm,
  SPLIT_RUN_STOP_CHOICES,
} from "./splitRunFooter";

const REJECT_APPROVE = [
  { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" },
  { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" },
] as const;

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
  cta: { label: "Review the run" },
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

  it("keeps Reject and Approve on a running order", () => {
    const footer = buildSplitRunFooter({
      kind: "running",
      note: { key: "running-step", headline: "Implement is running", text: "The log shows live progress." },
    });

    expect(footer.sentence).toBe("This work order is running.");
    expect(footer.note?.headline).toBe("Implement is running");
    expect(footer.run).toBeUndefined();
    expect(footer.actions).toEqual([...REJECT_APPROVE]);
    expect(splitRunCloseNeedsConfirm("running")).toBe(true);
    expect(DEFAULT_SPLIT_RUN_STOP_CHOICE).toBe("canceled");
    expect(SPLIT_RUN_STOP_CHOICES.map((choice) => choice.label)).toEqual([
      "Stop and Close",
      "Stop and Complete",
      "Rerun this step",
      "Rerun from the start",
    ]);
    expect(SPLIT_RUN_STOP_CHOICES.map((choice) => choice.actionLabel)).toEqual([
      "Stop and Close",
      "Stop and Complete",
      "Rerun step",
      "Rerun from start",
    ]);
    expect(SPLIT_RUN_STOP_CHOICES.map((choice) => choice.description)).toEqual([
      "Marks this task as Canceled",
      "Marks this task as Completed",
      "Starts this step again",
      "Starts this task from the first step",
    ]);
  });

  it("keeps a waiting note off the bar and shows Reject and Approve", () => {
    const footer = buildSplitRunFooter({ kind: "waiting", note: PR_NOTE });

    expect(footer.attentionCard).toBeUndefined();
    expect(footer.note).toEqual({
      headline: "Review the pull request",
      text: "Merge #6812 to continue Verify.",
      sourceName: "PR Closure",
      cta: PR_NOTE.cta,
    });
    expect(footer.sentence).toBe("This work order needs attention.");
    expect(footer.actions).toEqual([...REJECT_APPROVE]);
    expect(splitRunCloseNeedsConfirm("waiting")).toBe(false);
  });

  it("flags a visible waiting note as an attention card", () => {
    const footer = buildSplitRunFooter({ kind: "waiting", note: PR_NOTE, attentionCard: true });

    expect(footer.attentionCard).toBe(true);
    expect(footer.note?.cta).toEqual(PR_NOTE.cta);
    expect(footer.actions).toEqual([...REJECT_APPROVE]);
  });

  it("still shows Reject and Approve when a waiting order has no run note", () => {
    expect(buildSplitRunFooter({ kind: "waiting" })).toEqual({
      kind: "waiting",
      sentence: "This work order needs attention.",
      actions: [...REJECT_APPROVE],
    });
  });

  it("treats a failed open implement as a run note plus Reject and Approve", () => {
    const footer = buildSplitRunFooter({ kind: "failed", note: FAILED_NOTE, attentionCard: true });

    expect(footer.attentionCard).toBe(true);
    expect(footer.note?.headline).toBe("Implement did not pass");
    expect(footer.note?.cta?.label).toBe("Review the run");
    expect(footer.sentence).toBe("This work order needs attention.");
    expect(footer.actions).toEqual([...REJECT_APPROVE]);
    expect(splitRunCloseNeedsConfirm("failed")).toBe(false);
  });

  it("shows a done summary with Reopen", () => {
    expect(doneFooterForStatus("completed")).toEqual({
      kind: "done",
      sentence: "Work order completed successfully.",
      actions: [{ id: "reopen", kind: "reopen", label: "Reopen", emphasis: "primary" }],
      status: "completed",
    });
    expect(doneFooterForStatus("rejected").sentence).toBe("A person rejected this work order.");
    expect(doneFooterForStatus("cancelled").sentence).toBe("This work order was canceled.");
    expect(doneFooterForStatus("failed").sentence).toBe("Closed as failed. Line execution did not pass.");
  });

  it("offers Reopen on a closed failed footer instead of Stop", () => {
    const footer = buildSplitRunFooter({ kind: "failed", note: FAILED_NOTE, attentionCard: true, status: "failed" });

    expect(footer.actions).toEqual([{ id: "reopen", kind: "reopen", label: "Reopen", emphasis: "primary" }]);
  });
});

describe("availableSplitRunStopChoices", () => {
  it("keeps every Stop outcome while the work order is still open", () => {
    expect(availableSplitRunStopChoices("running").map((choice) => choice.id)).toEqual([
      "canceled",
      "completed",
      "rerun-step",
      "rerun-start",
    ]);
    expect(availableSplitRunStopChoices("waiting").map((choice) => choice.id)).toEqual([
      "canceled",
      "completed",
      "rerun-step",
      "rerun-start",
    ]);
    expect(defaultSplitRunStopChoice("running")).toBe("canceled");
    expect(defaultSplitRunStopChoice("waiting")).toBe("canceled");
    expect(defaultSplitRunStopChoice("waiting", "failed")).toBe("rerun-step");
  });

  it("drops the outcome that already matches the work order", () => {
    expect(availableSplitRunStopChoices("draft").map((choice) => choice.id)).toEqual(["canceled", "completed"]);
  });

  it("offers Reopen when the work order is already closed", () => {
    expect(availableSplitRunStopChoices("completed").map((choice) => choice.id)).toEqual(["reopen"]);
    expect(availableSplitRunStopChoices("cancelled").map((choice) => choice.id)).toEqual(["reopen"]);
    expect(availableSplitRunStopChoices("rejected").map((choice) => choice.id)).toEqual(["reopen"]);
    expect(availableSplitRunStopChoices("failed").map((choice) => choice.id)).toEqual(["reopen"]);
    expect(defaultSplitRunStopChoice("completed")).toBe("reopen");
  });
});

describe("rerunStartStepIndex", () => {
  it("uses the first step for Rerun from the start", () => {
    expect(rerunStartStepIndex("rerun-start", 2)).toBe(0);
  });

  it("keeps the current step for Rerun this step", () => {
    expect(rerunStartStepIndex("rerun-step", 2)).toBe(2);
  });
});
