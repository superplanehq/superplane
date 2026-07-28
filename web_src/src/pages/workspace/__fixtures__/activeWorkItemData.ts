import type { ActiveWorkItemData } from "../activeWorkItemTypes";

export const activeWorkItemData = {
  id: "PX-128",
  title: "Add SSO session recovery",
  goal: "Recover expired SSO sessions without losing the user route or form state.",
  projectName: "Project X",
  repository: "superplane/project-x",
  branch: "feat/sso-recovery",
  baseBranch: "main",
  elapsed: "12m",
  state: "awaiting-approval",
  filesChanged: 0,
  checksPassed: 18,
  checksTotal: 18,
  timeline: [
    {
      id: "request-accepted",
      time: "10:02",
      actor: "Maya Chen",
      title: "Work requested",
      description:
        "Add graceful recovery when an SSO session expires. Preserve the current route and any unsaved form state.",
      kind: "request",
      details: ["Linear PX-128", "Priority: High"],
    },
    {
      id: "context-loaded",
      time: "10:03",
      actor: "Planner",
      title: "Repository context loaded",
      description: "Mapped the authentication boundary, route restoration flow, and existing session renewal behavior.",
      kind: "system",
      details: ["47 files inspected", "18 relevant tests", "main @ 3f8a2c1"],
    },
    {
      id: "plan-v1",
      time: "10:05",
      actor: "Planner",
      title: "Plan v1 proposed",
      description:
        "Introduce a recovery controller around the SSO callback, then restore the intercepted route after renewal.",
      kind: "agent",
      details: ["4 implementation steps", "Estimated change: 6 files"],
    },
    {
      id: "plan-feedback",
      time: "10:07",
      actor: "You",
      title: "Plan feedback",
      description:
        "Reuse the existing session renewal path. Do not add a second recovery controller, and cover unsaved forms explicitly.",
      kind: "user",
    },
    {
      id: "plan-v2",
      time: "10:09",
      actor: "Planner",
      title: "Plan v2 ready for approval",
      description:
        "Revised around the existing renewal path and added a focused form-state recovery test. Work is paused at this checkpoint.",
      kind: "approval",
      details: ["2 changes from v1", "No new service boundary"],
    },
  ],
  plan: {
    version: 2,
    summary:
      "Extend the existing renewal flow, preserve navigation state, then verify recovery across routes and forms.",
    changesFromPrevious: [
      "Removed the proposed recovery controller.",
      "Added explicit coverage for unsaved form state.",
    ],
    steps: [
      {
        id: "trace",
        title: "Trace session expiry into the renewal path",
        description: "Confirm the callback and router boundaries before changing behavior.",
        state: "complete",
      },
      {
        id: "preserve",
        title: "Preserve route and form state",
        description: "Capture recoverable client state before redirecting into SSO renewal.",
        state: "changed",
      },
      {
        id: "restore",
        title: "Restore after session renewal",
        description: "Resume the intercepted navigation through the existing renewal handler.",
        state: "changed",
      },
      {
        id: "verify",
        title: "Verify recovery paths",
        description: "Cover route restoration, unsaved forms, and failed renewal behavior.",
        state: "queued",
      },
    ],
  },
  agents: [
    { name: "Planner", role: "Scope and decisions", status: "Awaiting approval", active: true },
    { name: "Builder", role: "Implementation", status: "Ready", active: false },
    { name: "Verifier", role: "Tests and review", status: "Ready", active: false },
  ],
} satisfies ActiveWorkItemData;
