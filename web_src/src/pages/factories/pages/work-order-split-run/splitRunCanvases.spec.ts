import { describe, expect, it } from "vitest";

import { groupSplitRunStream } from "./PhaseLogCard";
import {
  canvasKeyForAutomation,
  canvasKeyForPhase,
  claudeCodeSteps,
  lineAutomationPresentation,
  parseSplitRunCanvasKey,
  richStreamForCanvas,
  splitRunCanvasForPhase,
} from "./splitRunCanvases";
import { SPLIT_RUN_RUNNING } from "./splitRunMocks";

describe("parseSplitRunCanvasKey", () => {
  it("accepts known canvas keys", () => {
    expect(parseSplitRunCanvasKey("implementation")).toBe("implementation");
  });

  it("returns undefined for unknown keys", () => {
    expect(parseSplitRunCanvasKey("canvas")).toBeUndefined();
  });
});

describe("canvasKeyForAutomation", () => {
  it("renames leftover refund apps to the line automations", () => {
    expect(
      lineAutomationPresentation({ id: "app-refund-implementer", name: "Refund Implementer" }, "Refund Implementer"),
    ).toEqual({
      name: "Implement",
      componentName: "Implementation",
    });
    expect(lineAutomationPresentation({ id: "app-refund-planner", name: "Refund Planner" }, "Plan")).toEqual({
      name: "Plan",
      componentName: "Planning",
    });
    expect(lineAutomationPresentation({ id: "app-refund-verifier", name: "Refund Verifier" }, "Verify")).toEqual({
      name: "Verify",
      componentName: "Risk Assessment",
    });
  });

  it("maps planner, implementer, verifier, and closure apps", () => {
    expect(canvasKeyForAutomation({ id: "app-refund-planner" })).toBe("planning");
    expect(canvasKeyForAutomation({ id: "app-refund-implementer" })).toBe("implementation");
    expect(canvasKeyForAutomation({ id: "app-refund-verifier", name: "Refund Verifier" })).toBe("risk");
    expect(canvasKeyForAutomation({ name: "Verify" })).toBe("risk");
    expect(canvasKeyForAutomation({ id: "app-pr-closure", name: "PR Closure" })).toBe("closure");
    expect(canvasKeyForAutomation({ id: "app-refund-backlog", name: "Ingest" })).toBe("intake");
    expect(canvasKeyForAutomation({ name: "Sentry" })).toBe("sentry");
    expect(canvasKeyForAutomation({ name: "Slack" })).toBe("slack");
  });

  it("labels intake automations as Backlog plus the source canvas", () => {
    expect(lineAutomationPresentation({ id: "app-refund-backlog", name: "Ingest" })).toEqual({
      name: "Backlog",
      componentName: "Ingest",
    });
    expect(lineAutomationPresentation({ name: "Sentry" })).toEqual({
      name: "Backlog",
      componentName: "Sentry",
    });
    expect(lineAutomationPresentation({ name: "Slack" })).toEqual({
      name: "Backlog",
      componentName: "Slack",
    });
  });

  it("returns undefined when the app does not name a canvas", () => {
    expect(canvasKeyForAutomation({ id: "app-custom" })).toBeUndefined();
  });
});

describe("splitRunCanvasForPhase", () => {
  it("opens the implementation canvas for the running implement step", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    expect(implement).toBeDefined();
    expect(canvasKeyForPhase(implement!)).toBe("implementation");

    const canvas = splitRunCanvasForPhase(implement!);
    expect(canvas.title).toBe("Implement");
    expect(canvas.nodes.map((node) => node.name)).toContain("Create Branch");
    expect(canvas.nodes.map((node) => node.name)).toContain("Agent - Implement from order description");
    expect(canvas.nodes.map((node) => node.name)).toContain("Create Draft Pull Request");
    expect(canvas.nodes.map((node) => node.name)).toContain("Attach PR to Work Order");
    expect(canvas.statuses["create-branch"]).toBe("passed");
    expect(canvas.statuses["implementation-agent-no-issue"]).toBe("running");
    expect(canvas.statuses["create-draft-pr"]).toBe("did_not_run");
    expect(canvas.statuses["attach-pr-artifact"]).toBe("did_not_run");
  });

  it("opens a draft pull request after a finished implement run", () => {
    const canvas = splitRunCanvasForPhase({
      id: "implement-0",
      name: "Implement",
      status: "passed",
      duration: "8m",
      componentName: "Implementation",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });

    expect(canvas.statuses["implementation-agent-no-issue"]).toBe("passed");
    expect(canvas.statuses["create-draft-pr"]).toBe("passed");
    expect(canvas.statuses["attach-pr-artifact"]).toBe("passed");

    const stream = richStreamForCanvas(canvas);
    expect(stream.find((line) => line.id === "create-draft-pr")).toMatchObject({
      componentType: "github.createPullRequest",
      componentName: "Create Draft Pull Request",
      action: "passed",
    });
    expect(stream.find((line) => line.id === "attach-pr-artifact")?.artifact?.type).toBe("TYPE_PR");
  });

  it("opens the planning canvas for a completed plan step", () => {
    const plan = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "plan");
    expect(canvasKeyForPhase(plan!)).toBe("planning");

    const canvas = splitRunCanvasForPhase(plan!);
    expect(canvas.title).toBe("Plan");
    expect(canvas.statuses["onrun-create-plan"]).toBe("triggered");
    expect(canvas.statuses["planner-agent-no-issue"]).toBe("passed");
    expect(canvas.statuses["add-plan-artifact"]).toBe("passed");
  });

  it("writes one log line per canvas node and extra Claude Code notes", () => {
    const canvas = splitRunCanvasForPhase({
      id: "done",
      name: "Done",
      status: "passed",
      duration: "1m 12s",
      componentName: "PR Closure",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas);

    expect(canvas.nodes.length).toBeGreaterThanOrEqual(5);
    expect(stream.filter((line) => !line.note).length).toBe(canvas.nodes.length);
    expect(stream.filter((line) => line.nodeId === "find-work-order").map((line) => line.id)).toEqual([
      "find-work-order",
    ]);
    expect(stream.some((line) => line.artifact?.type === "TYPE_PR")).toBe(true);
    expect(
      stream.some(
        (line) =>
          line.artifact?.data && "name" in line.artifact.data && line.artifact.data.name === "merge-screenshot.png",
      ),
    ).toBe(true);
    expect(stream.some((line) => line.artifact?.type === "TYPE_MARKDOWN")).toBe(true);
  });

  it("omits invented files and ledger pull requests when demo artifacts are off", () => {
    const canvas = splitRunCanvasForPhase({
      id: "done",
      name: "Done",
      status: "passed",
      duration: "1m 12s",
      componentName: "PR Closure",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas, undefined, { demoArtifacts: false });

    expect(stream.some((line) => line.artifact)).toBe(false);
  });

  it("adds a verbose transcript for Claude Code and a score for checks", () => {
    const canvas = splitRunCanvasForPhase({
      id: "verify",
      name: "Verify",
      status: "passed",
      duration: "2m",
      componentName: "Risk Assessment",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas);
    const agentNotes = stream.filter((line) => line.nodeId === "assess-pr-risk" && line.note);
    expect(agentNotes.map((line) => line.componentName)).toEqual([
      "Reading the pull request diff.",
      "Scoring retry-policy risk.",
      "Writing the risk review.",
    ]);
    expect(stream.find((line) => line.id === "report-risk-check")).toMatchObject({
      kind: "check",
      componentName: "Risk review",
      action: "65/100",
    });
    const groups = groupSplitRunStream(stream);
    expect(groups.some((group) => group.line.action === "did not run")).toBe(false);
    const agent = groups.find((group) => group.line.id === "assess-pr-risk");
    expect(agent?.notes.map((line) => line.componentName)).toEqual([
      "Reading the pull request diff.",
      "Scoring retry-policy risk.",
      "Writing the risk review.",
    ]);
  });

  it("writes icon, type, name, and action on each canvas node line", () => {
    const canvas = splitRunCanvasForPhase({
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Ingest",
      artifacts: [],
      stream: [],
      canvasSteps: [],
      canvasKey: "intake",
      triggerName: "On Issue Label",
    });
    const stream = richStreamForCanvas(canvas);
    const labeled = stream.find((line) => line.id === "on-issue-labeled");
    const factoryLabel = stream.find((line) => line.id === "has-factory-label");
    const created = stream.find((line) => line.id === "create-work-order");
    const idle = stream.find((line) => line.id === "on-issue-assigned");

    expect(labeled).toMatchObject({
      componentType: "github.onIssue",
      componentName: "On Issue Label",
      action: "triggered",
      iconSlug: "github",
    });
    expect(factoryLabel).toMatchObject({
      componentType: "Filter",
      componentName: "Factory Label?",
      action: "passed",
      iconSlug: "funnel",
    });
    expect(created).toMatchObject({
      componentType: "Create Work Order",
      componentName: "Create Work Order",
      action: "passed",
    });
    expect(idle).toMatchObject({
      componentType: "github.onIssue",
      componentName: "On Issue Assignment",
      action: "did not run",
    });
  });

  it("uses catalog labels and namespaced ids for planning components", () => {
    const canvas = splitRunCanvasForPhase({
      id: "plan",
      name: "Plan",
      status: "passed",
      duration: "1m",
      componentName: "Planning",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas);
    expect(stream.find((line) => line.id === "add-plan-artifact")).toMatchObject({
      componentType: "Add Work Order Artifact",
      componentName: "Add Plan Artifact",
    });
    expect(stream.find((line) => line.id === "planner-agent-no-issue")).toMatchObject({
      componentType: "Run Claude Code",
      componentName: "Agent - No GH Issue Plan",
    });
    const plannerNotes = stream.filter((line) => line.nodeId === "planner-agent-no-issue" && line.note);
    expect(
      plannerNotes
        .filter((line) => !line.noteParentId)
        .map((line) => ({
          name: line.componentName,
          type: line.componentType,
        })),
    ).toEqual([
      { name: "Clone Repo", type: "bash" },
      { name: "Provide description", type: "bash" },
      { name: "Write Implementation Plan", type: "prompt" },
      { name: "Use plan as output", type: "bash" },
    ]);
    expect(
      plannerNotes.some(
        (line) =>
          line.componentName === "Clone Repo" && line.status === "passed" && line.detail?.includes("Cloning into"),
      ),
    ).toBe(true);
    expect(plannerNotes.some((line) => line.componentName === "cat /tmp/ORDER.md" && line.noteParentId)).toBe(true);
    expect(
      plannerNotes.some(
        (line) =>
          line.componentType === "note" && line.componentName === "Let me examine the key reference files in detail.",
      ),
    ).toBe(true);
    expect(
      plannerNotes.some((line) => line.componentName.includes("LineListCard.tsx") && line.componentType === "read"),
    ).toBe(true);
    expect(plannerNotes.some((line) => line.componentName.includes("import type"))).toBe(false);
    expect(claudeCodeSteps(canvas.nodes.find((node) => node.id === "planner-agent-no-issue") ?? {})).toEqual([
      { name: "Clone Repo", type: "bash" },
      { name: "Write Implementation Plan", type: "prompt" },
      { name: "Use plan as output", type: "bash" },
    ]);
  });

  it("uses the implementation runner log under Claude Code", () => {
    const canvas = splitRunCanvasForPhase({
      id: "implement",
      name: "Implement",
      status: "running",
      duration: "4m",
      componentName: "Implementation",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas);
    const agentNotes = stream.filter((line) => line.nodeId === "implementation-agent-no-issue" && line.note);
    expect(
      agentNotes
        .filter((line) => !line.noteParentId)
        .map((line) => ({
          name: line.componentName,
          type: line.componentType,
        })),
    ).toEqual([
      { name: "Set Up Git User", type: "bash" },
      { name: "Provide order", type: "bash" },
      { name: "Provide Plan", type: "bash" },
      { name: "Checkout Branch", type: "bash" },
      { name: "Set Up DCO Signing", type: "bash" },
      { name: "Set Up Environment", type: "bash" },
      { name: "Implementation", type: "prompt" },
      { name: "Format Code", type: "bash" },
      { name: "Commit and Push", type: "bash" },
    ]);
    expect(agentNotes.some((line) => line.componentName === "cat /tmp/ORDER.md" && line.noteParentId)).toBe(true);
    expect(
      agentNotes.some(
        (line) =>
          line.componentType === "note" &&
          line.componentName ===
            "Now let's look at the messages file, factory_notification_consumer.go, and other referenced files.",
      ),
    ).toBe(true);
    expect(agentNotes.some((line) => line.componentName.includes("package models"))).toBe(false);
  });

  it("paints the GitHub label path on Ingest and leaves assignment idle", () => {
    const canvas = splitRunCanvasForPhase({
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Ingest",
      artifacts: [],
      stream: [],
      canvasSteps: [],
      canvasKey: "intake",
      triggerName: "On Issue Label",
    });

    expect(canvas.title).toBe("Ingest");
    expect(canvas.statuses["on-issue-labeled"]).toBe("triggered");
    expect(canvas.statuses["has-factory-label"]).toBe("passed");
    expect(canvas.statuses["create-work-order"]).toBe("passed");
    expect(canvas.statuses["on-issue-assigned"]).toBe("did_not_run");
    expect(canvas.statuses["assigned-to-agent"]).toBe("did_not_run");
  });

  it("opens Sentry and Slack intake canvases", () => {
    const sentry = splitRunCanvasForPhase({
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Sentry",
      artifacts: [],
      stream: [],
      canvasSteps: [],
      canvasKey: "sentry",
    });
    const slack = splitRunCanvasForPhase({
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Slack",
      artifacts: [],
      stream: [],
      canvasSteps: [],
      canvasKey: "slack",
    });

    expect(sentry.title).toBe("Sentry");
    expect(sentry.nodes.map((node) => node.name)).toEqual(["On Issue", "Factory project?", "Create Work Order"]);
    expect(slack.title).toBe("Slack");
    expect(slack.nodes.map((node) => node.name)).toEqual(["On Mention", "Mentioned the agent?", "Create Work Order"]);
  });

  it("returns an empty canvas when a person created the work order", () => {
    const canvas = splitRunCanvasForPhase({
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Created manually",
      artifacts: [],
      stream: [],
      canvasSteps: [],
      canvasKey: null,
    });

    expect(canvas.nodes).toEqual([]);
    expect(canvas.title).toBe("");
  });
});
