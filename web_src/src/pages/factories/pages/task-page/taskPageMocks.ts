import {
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import {
  OPEN_WORK_ORDER_ARTIFACTS,
  OPEN_WORK_ORDER_PULL_REQUESTS,
} from "../../__fixtures__/factoryPageFixtureVariants";
import { OPEN_WORK_ORDER_CHECKS, RUNNING_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { CHECKOUT_RELIABILITY_MISSION } from "../missions/missionMocks";
import { buildTaskPageRecord, type TaskPageActivityItem, type TaskPageRecord } from "./taskPageModel";

const OPEN_ACTIVITY: TaskPageActivityItem[] = [
  {
    id: "open-created",
    actor: "On Issue Label",
    text: "created this task",
    timeLabel: "1h ago",
    kind: "event",
  },
  {
    id: "open-assigned",
    actor: STORYBOOK_ME_USER_NAME,
    text: "became the owner",
    timeLabel: "58m ago",
    kind: "event",
  },
  {
    id: "open-comment",
    actor: STORYBOOK_ME_USER_NAME,
    text: "The retry window is the likely cause. Review the pull request before we start the next line step.",
    timeLabel: "25m ago",
    kind: "comment",
  },
];

const RUNNING_ACTIVITY: TaskPageActivityItem[] = [
  {
    id: "running-created",
    actor: "On Issue Label",
    text: "created this task",
    timeLabel: "1d ago",
    kind: "event",
  },
  {
    id: "running-started",
    actor: STORYBOOK_ME_USER_NAME,
    text: "started Plan and Implement",
    timeLabel: "1h ago",
    kind: "event",
  },
  {
    id: "running-comment",
    actor: STORYBOOK_ME_USER_NAME,
    text: "The implementer is adding the regression test. I will review the log when the step finishes.",
    timeLabel: "40m ago",
    kind: "comment",
  },
];

const CLOSED_ACTIVITY: TaskPageActivityItem[] = [
  {
    id: "closed-created",
    actor: "On Issue Label",
    text: "created this task",
    timeLabel: "7d ago",
    kind: "event",
  },
  {
    id: "closed-completed",
    actor: STORYBOOK_ME_USER_NAME,
    text: "marked this task completed",
    timeLabel: "1d ago",
    kind: "event",
  },
  {
    id: "closed-comment",
    actor: STORYBOOK_ME_USER_NAME,
    text: "Audit entries match the provider export for every month in the backfill window.",
    timeLabel: "1d ago",
    kind: "comment",
  },
];

export const OPEN_TASK_PAGE: TaskPageRecord = buildTaskPageRecord({
  order: OPEN_WORK_ORDER,
  factoryKey: PRIMARY_FACTORY_KEY,
  missionName: CHECKOUT_RELIABILITY_MISSION.name,
  checks: OPEN_WORK_ORDER_CHECKS,
  artifacts: OPEN_WORK_ORDER_ARTIFACTS,
  pullRequests: OPEN_WORK_ORDER_PULL_REQUESTS,
  activity: OPEN_ACTIVITY,
});

export const RUNNING_TASK_PAGE: TaskPageRecord = buildTaskPageRecord({
  order: RUNNING_WORK_ORDER,
  factoryKey: PRIMARY_FACTORY_KEY,
  missionName: CHECKOUT_RELIABILITY_MISSION.name,
  checks: RUNNING_WORK_ORDER_CHECKS,
  artifacts: OPEN_WORK_ORDER_ARTIFACTS,
  pullRequests: OPEN_WORK_ORDER_PULL_REQUESTS,
  activity: RUNNING_ACTIVITY,
});

export const CLOSED_TASK_PAGE: TaskPageRecord = buildTaskPageRecord({
  order: CLOSED_WORK_ORDER,
  factoryKey: PRIMARY_FACTORY_KEY,
  missionName: CHECKOUT_RELIABILITY_MISSION.name,
  checks: OPEN_WORK_ORDER_CHECKS,
  artifacts: OPEN_WORK_ORDER_ARTIFACTS,
  pullRequests: OPEN_WORK_ORDER_PULL_REQUESTS,
  activity: CLOSED_ACTIVITY,
});

export const EMPTY_TASK_PAGE: TaskPageRecord = buildTaskPageRecord({
  order: {
    ...DRAFT_WORK_ORDER,
    title: "Clarify refund retry window",
    description: "",
    assignees: [],
    lineDispatches: [],
  },
  factoryKey: PRIMARY_FACTORY_KEY,
});
