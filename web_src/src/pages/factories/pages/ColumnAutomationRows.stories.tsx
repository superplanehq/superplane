import type { Meta, StoryObj } from "@storybook/react-vite";
import { MoreHorizontal, Plus } from "lucide-react";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";

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
  action: "Plan the task",
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

const VERIFY_AUTOMATIONS: ColumnAutomation[] = [
  { ...GITHUB_INTAKE, id: "pr-comments", kind: "pr-discussion", name: "Pull request comments" },
  { ...GITHUB_INTAKE, id: "pr-checks", kind: "pr-checks", name: "Status checks" },
  { ...ANALYSIS, id: "merge-confidence", kind: "risk-score", name: "Merge confidence" },
];

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
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const labelOffsets: number[] = [];
    const cardTops: number[] = [];
    for (const title of ["Backlog", "Implement", "Verify", "Review"]) {
      const lane = within(canvas.getByRole("region", { name: title }));
      const titleLeft = lane.getByRole("heading", { name: title }).getBoundingClientRect().left;
      const rows = within(lane.getByRole("list", { name: `${title} automations` }));
      for (const button of rows.getAllByRole("button", { name: /^(Open automation:|Add automation)/ })) {
        const icon = button.querySelector("img, svg");
        const label = button.lastElementChild;
        if (!icon || !label) throw new Error("The automation row is missing its icon or label.");
        expect(Math.abs(icon.getBoundingClientRect().left - titleLeft)).toBeLessThanOrEqual(1);
        labelOffsets.push(label.getBoundingClientRect().left - titleLeft);
      }
      cardTops.push(lane.getAllByRole("listitem").at(-1)!.getBoundingClientRect().top);
    }
    expect(Math.max(...labelOffsets) - Math.min(...labelOffsets)).toBeLessThanOrEqual(1);
    expect(Math.max(...cardTops) - Math.min(...cardTops)).toBeLessThanOrEqual(1);
  },
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

export const ScheduleControls: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole("button", { name: "Open automation: Scheduled repository review" }));
    await expect(canvas.getByRole("status")).toHaveTextContent("Automation settings opened.");

    const run = canvas.getByRole("button", { name: "Run Scheduled repository review" });
    await userEvent.click(run);
    await expect(run).toBeDisabled();

    // Use browser hit testing: a disabled button can pass pointer clicks to the row.
    const bounds = run.getBoundingClientRect();
    const target = canvasElement.ownerDocument.elementFromPoint(
      bounds.x + bounds.width / 2,
      bounds.y + bounds.height / 2,
    );
    if (!target) throw new Error("The pending Run control is not visible.");
    await userEvent.click(target);
    await expect(canvas.getByRole("status")).toHaveTextContent("Automation run requested.");
  },
  render: function ScheduleControlsStory() {
    const [pending, setPending] = useState(false);
    const [status, setStatus] = useState("");
    const automation: ColumnAutomation = {
      ...VERIFY,
      id: "scheduled-review",
      canvasId: "scheduled-review",
      name: "Scheduled repository review",
      trigger: "On schedule",
      canRunNow: true,
    };
    return (
      <div className="max-w-sm space-y-4 rounded-lg bg-muted p-4">
        <h2 className="text-sm font-semibold">Verify automations</h2>
        <ColumnAutomationRows
          title="Verify"
          automations={[automation]}
          rowCount={1}
          testId="schedule-controls"
          onRowAction={(selected, action) => {
            console.log("Automation action", selected.id, action);
            setStatus("Automation settings opened.");
          }}
          onRun={(selected) => {
            console.log("Run automation", selected.id);
            setPending(true);
            setStatus("Automation run requested.");
          }}
          runPending={pending}
          runningAutomationId={pending ? automation.canvasId : undefined}
        />
        {status ? (
          <p role="status" className="text-sm">
            {status}
          </p>
        ) : null}
      </div>
    );
  },
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
          automations={[
            { ...IMPLEMENT, name: "Scheduled repository implementation and documentation review", canRunNow: true },
          ]}
          cards={[{ title: "Notify on status change after a refund", meta: "Draft #114" }]}
        />
        <Lane
          title="Verify"
          automations={VERIFY_AUTOMATIONS}
          cards={[{ title: "Add refund reason enum to schema", meta: "Draft #1022" }]}
        />
        <Lane
          title="Review"
          automations={[]}
          onAdd={() => console.log("Add automation", "Review")}
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
          rowCount={3}
          onRowAction={(automation, action) => console.log("Automation action", automation.id, action)}
          onRun={(automation) => console.log("Run automation", automation.id)}
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
