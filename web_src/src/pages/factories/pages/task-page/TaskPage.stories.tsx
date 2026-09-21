import { useState, type ReactElement } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import { TaskPage } from "./TaskPage";
import { CLOSED_TASK_PAGE, EMPTY_TASK_PAGE, OPEN_TASK_PAGE, RUNNING_TASK_PAGE } from "./taskPageMocks";
import type { TaskPageRecord, TaskPageView } from "./taskPageModel";

/**
 * Dedicated task page candidate. Storybook-only: mounted through the
 * workspace chrome with the Tasks route swapped for the Clay-style record.
 * The live board popup does not change.
 */
const meta = {
  title: "Factories/Pages/Task Page",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj;

const tasksPath = `workspaces/${PRIMARY_FACTORY_KEY}/tasks`;

function logAction() {
  console.log("primary action");
}

function ReadyTaskPage({ record: initial }: { record: TaskPageRecord }) {
  const [record, setRecord] = useState(initial);

  return (
    <TaskPage
      view={{ state: "ready", record }}
      onTitleSave={(title) => {
        console.log("save title", title);
        setRecord((current) => ({ ...current, title }));
      }}
      onPrimaryAction={logAction}
      onCommentSubmit={(body) => {
        console.log("comment", body);
        setRecord((current) => ({
          ...current,
          activity: [
            ...current.activity,
            {
              id: `story-comment-${current.activity.length + 1}`,
              actor: STORYBOOK_ME_USER_NAME,
              text: body,
              timeLabel: "now",
              kind: "comment",
            },
          ],
        }));
      }}
    />
  );
}

function OpenTask() {
  return <ReadyTaskPage record={OPEN_TASK_PAGE} />;
}

function RunningTask() {
  return <ReadyTaskPage record={RUNNING_TASK_PAGE} />;
}

function ClosedTask() {
  return <ReadyTaskPage record={CLOSED_TASK_PAGE} />;
}

function EmptyTask() {
  return <ReadyTaskPage record={EMPTY_TASK_PAGE} />;
}

function LoadingTask() {
  return <TaskPage view={{ state: "loading" }} />;
}

function ErrorTask() {
  const [view, setView] = useState<TaskPageView>({ state: "error" });
  if (view.state === "ready") {
    return <ReadyTaskPage record={view.record} />;
  }
  return <TaskPage view={view} onRetry={() => setView({ state: "ready", record: OPEN_TASK_PAGE })} />;
}

function taskStory(Page: () => ReactElement): Story {
  return {
    render: () => (
      <FactoriesHarness
        pathSuffix={tasksPath}
        factoriesFixture={defaultFactoriesFixture}
        enableOnboarding={false}
        pageOverrides={{ workOrders: Page }}
      />
    ),
  };
}

/** Open task waiting for review. Properties, write-up, and related lists stay visible. */
export const Open: Story = taskStory(OpenTask);

/** Running task. Spend and factory lines show the active line. */
export const Running: Story = taskStory(RunningTask);

/** Closed task. Reopen is the primary action. */
export const Closed: Story = taskStory(ClosedTask);

/** Draft task with no write-up, checks, artifacts, pull requests, or activity. */
export const Empty: Story = {
  name: "Empty",
  ...taskStory(EmptyTask),
};

/** Loading placeholders while the record is not ready. */
export const Loading: Story = taskStory(LoadingTask);

/** Failed load with a Retry action that opens the record. */
export const Error: Story = taskStory(ErrorTask);
