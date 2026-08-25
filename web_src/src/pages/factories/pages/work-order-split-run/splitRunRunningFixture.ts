import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

import { OPEN_WORK_ORDER_ARTIFACTS } from "../../__fixtures__/factoryPageFixtureVariants";
import {
  HOUR_AGO,
  REVIEWER_USER,
  RUNNING_WORK_ORDER,
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import { DESCRIPTION_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import { buildSplitRunFooter } from "./splitRunFooter";
import type { SplitRunFixture } from "./splitRunMocks";
import { splitRunSourceForOrder } from "./splitRunSource";

const PLAN_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-plan-md",
  type: "TYPE_MARKDOWN",
  data: {
    name: "plan.md",
    title: "plan.md",
    body: "Add a focused test for the refund reconciliation worker.\nCover the timeout-then-retry path.",
  },
  createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
  createdAt: HOUR_AGO,
};

const OWNER: OrgUserDisplay = {
  id: STORYBOOK_ME_USER_ID,
  name: STORYBOOK_ME_USER_NAME,
  initials: getUserInitials(STORYBOOK_ME_USER_NAME),
  avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
};

export const SPLIT_RUN_RUNNING: SplitRunFixture = {
  title: "Add refund reconciliation test",
  owner: OWNER,
  elapsed: "4 min so far",
  startedLabel: "Started 1h ago",
  costUsd: "$0.73",
  tokensLabel: "2.7k tokens",
  lineName: "plan-and-implement",
  lineStatus: "running",
  currentPhaseId: "implement",
  waitingNotes: [],
  checks: [],
  footer: buildSplitRunFooter({
    kind: "running",
    note: {
      key: "running-step",
      headline: "Implement is running",
      text: "Implementation works on this step now. The log shows live progress.",
    },
  }),
  footerTone: "running",
  source: splitRunSourceForOrder(RUNNING_WORK_ORDER),
  phases: [
    {
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Ingest",
      artifacts: [DESCRIPTION_ARTIFACT],
      canvasKey: "intake",
      triggerName: "On Issue Label",
      appId: "app-refund-backlog",
      stream: [
        {
          id: "backlog-create",
          at: "12:24:02",
          componentName: "Create Work Order",
          status: "passed",
          duration: "2s",
          detail: "description.md",
        },
      ],
      canvasSteps: [
        {
          id: "create-work-order",
          title: "Create work order",
          componentName: "Create Work Order",
          provider: "superplane",
          status: "passed",
          detail: "description.md",
          duration: "2s",
        },
      ],
    },
    {
      id: "plan",
      name: "Create plan",
      status: "passed",
      duration: "1m 12s",
      componentName: "Create plan",
      artifacts: [PLAN_ARTIFACT],
      canvasKey: "planning",
      stream: [
        {
          id: "plan-read",
          at: "12:24:05",
          componentName: "Read Work Order",
          status: "passed",
          duration: "4s",
          detail: "description.md",
        },
        {
          id: "plan-write",
          at: "12:24:09",
          componentName: "Planning",
          status: "passed",
          duration: "1m 8s",
          detail: "plan.md",
        },
      ],
      canvasSteps: [
        {
          id: "read-order",
          title: "Read work order",
          componentName: "Read Work Order",
          provider: "superplane",
          status: "passed",
          detail: "description.md",
          duration: "4s",
        },
        {
          id: "refund-planner",
          title: "Write plan",
          componentName: "Planning",
          provider: "superplane",
          status: "passed",
          detail: "plan.md",
          duration: "1m 8s",
        },
      ],
    },
    {
      id: "implement",
      name: "Implement",
      status: "running",
      duration: "4m",
      componentName: "Implementation",
      artifacts: OPEN_WORK_ORDER_ARTIFACTS.filter((artifact) => artifact.id === "art-branch-1"),
      stream: [
        {
          id: "impl-branch",
          at: "12:25:14",
          componentName: "Create Branch",
          status: "passed",
          duration: "4s",
          detail: "feature/refund-retry",
        },
        {
          id: "impl-read",
          at: "12:25:18",
          componentName: "Read Artifact",
          status: "passed",
          duration: "3s",
          detail: "plan.md",
        },
        {
          id: "impl-write-file",
          at: "12:25:22",
          componentName: "Write File",
          status: "passed",
          duration: "11s",
          detail: "reconciliation_worker_test.go",
        },
        {
          id: "impl-agent",
          at: "12:25:33",
          componentName: "Implementation",
          status: "running",
          duration: "4m so far",
          detail: "reconciliation_worker_test.go",
        },
        {
          id: "impl-pr",
          at: "—",
          componentName: "Create Pull Request",
          status: "pending",
          detail: "Waits on Implementation",
        },
      ],
      canvasSteps: [
        {
          id: "read-plan",
          title: "Read plan",
          componentName: "Read Artifact",
          provider: "superplane",
          status: "passed",
          detail: "plan.md",
          duration: "3s",
        },
        {
          id: "refund-implementer",
          title: "Write test",
          componentName: "Implementation",
          provider: "superplane",
          status: "running",
          detail: "reconciliation_worker_test.go",
          duration: "4m so far",
        },
        {
          id: "open-pr",
          title: "Open draft PR",
          componentName: "Create Pull Request",
          provider: "github",
          status: "pending",
          detail: "Waits on Implementation",
        },
      ],
    },
  ],
};
