import type { Meta, StoryObj } from "@storybook/react-vite";
import { MoreHorizontal, Plus } from "lucide-react";

import githubIcon from "@/assets/icons/integrations/github.svg";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import type { ColumnAutomation } from "../lib/columnAutomations";
import { WorkOrderBoardLane, WorkOrderKanbanBoard } from "../workOrders/WorkOrderBoardChrome";
import { ColumnAutomationRows } from "./ColumnAutomationRows";
import { lineBoardColumnLaneClassName } from "./lineBoardColumnColors";

const GITHUB_INTAKE: ColumnAutomation = {
  id: "intake-github",
  kind: "intake",
  name: "GitHub issues",
  trigger: "On GitHub issue",
  action: "Create a task",
  iconSrc: githubIcon,
  iconAlt: "GitHub",
  health: "healthy",
  runningCount: 0,
  catalogId: "github-issues",
};

const ANALYSIS: ColumnAutomation = {
  ...GITHUB_INTAKE,
  id: "analysis",
  kind: "analysis",
  name: "Task analysis",
  trigger: "On task in Backlog",
  action: "Score the task",
  iconSrc: "",
  iconAlt: "",
  catalogId: "analysis",
};

const IMPLEMENT: ColumnAutomation = {
  ...GITHUB_INTAKE,
  id: "implement-agent",
  kind: "agent-step",
  name: "Implement",
  trigger: "On task in Implement",
  action: "Run the Implement agent",
  iconSrc: "",
  iconAlt: "",
  catalogId: "agent-step",
};

const VERIFY: ColumnAutomation = {
  ...IMPLEMENT,
  id: "verify-agent",
  name: "Verify",
  trigger: "On task in Verify",
  action: "Run the Verify agent",
};

const meta = {
  title: "Factories/Components/ColumnAutomationRows",
  component: ColumnAutomationRows,
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-screen bg-background p-6 dark:bg-background">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const SettingsRows: Story = {
  name: "Settings rows",
  render: () => <Board />,
};

export const EmptyWithAdd: Story = {
  name: "Empty with Add automation",
  render: () => (
    <div className="h-[240px] max-w-xs">
      <Lane title="Review" automations={[]} onAdd={() => undefined} cards={[]} />
    </div>
  ),
};

function Board() {
  return (
    <div className="h-[280px]">
      <WorkOrderKanbanBoard testId="automation-rows-settings">
        <Lane
          title="Backlog"
          colorId="lime"
          automations={[GITHUB_INTAKE, ANALYSIS]}
          showAdd
          cards={[{ title: "Show a clearer empty state on the setup page", meta: "1 hour ago" }]}
        />
        <Lane
          title="Implement"
          automations={[IMPLEMENT]}
          cards={[{ title: "Notify on status change after a refund", meta: "Draft #114" }]}
        />
        <Lane
          title="Verify"
          automations={[VERIFY]}
          cards={[{ title: "Add refund reason enum to schema", meta: "Draft #1022" }]}
        />
        <Lane
          title="Review"
          automations={[]}
          onAdd={() => undefined}
          cards={[{ title: "Ship idempotent refund retries", meta: "Review #6812" }]}
        />
      </WorkOrderKanbanBoard>
    </div>
  );
}

function Lane({
  title,
  colorId,
  automations,
  showAdd,
  onAdd,
  cards,
}: {
  title: string;
  colorId?: "lime";
  automations: ColumnAutomation[];
  showAdd?: boolean;
  onAdd?: () => void;
  cards: Array<{ title: string; meta: string }>;
}) {
  return (
    <WorkOrderBoardLane
      title={title}
      count={cards.length}
      emptyDescription={`No tasks in ${title}.`}
      keepChildrenWhenEmpty
      surfaceClassName={lineBoardColumnLaneClassName(colorId)}
      className={colorId ? undefined : "bg-muted"}
      actions={
        <div className="flex shrink-0 items-center gap-0.5">
          {showAdd ? (
            <span className="flex size-6 items-center justify-center rounded-md text-muted-foreground">
              <Plus className="size-3.5" aria-hidden />
            </span>
          ) : null}
          <span className="flex size-6 items-center justify-center rounded-md text-muted-foreground">
            <MoreHorizontal className="size-3.5" aria-hidden />
          </span>
        </div>
      }
      subheader={
        <ColumnAutomationRows
          title={title}
          automations={automations}
          rowCount={2}
          onRowAction={() => undefined}
          onAdd={onAdd}
          testId={`${title.toLowerCase()}-rows`}
        />
      }
      testId={`${title.toLowerCase()}-lane`}
    >
      <ul className="flex flex-col gap-2 p-0.5">
        {cards.map((card) => (
          <li key={card.title} className="rounded-lg border border-border/70 bg-background px-3 py-2.5">
            <p className="truncate text-[13px] font-medium text-foreground">{card.title}</p>
            <p className="mt-1 text-[12px] text-muted-foreground">{card.meta}</p>
          </li>
        ))}
      </ul>
    </WorkOrderBoardLane>
  );
}
