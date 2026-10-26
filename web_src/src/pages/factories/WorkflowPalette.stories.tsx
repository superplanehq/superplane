import type { Meta, StoryObj } from "@storybook/react-vite";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "./lib/workOrderProgress";
import { lineBoardColumnLaneProps, type LineBoardColumnColorId } from "./pages/lineBoardColumnColors";
import { WorkOrderBoardLane, WorkOrderKanbanBoard } from "./workOrders/WorkOrderBoardChrome";

const stages: { title: string; color: LineBoardColumnColorId; status: WorkOrderDisplayStatus; task: string }[] = [
  { title: "Backlog", color: "slate", status: "draft", task: "Prepare the implementation plan" },
  { title: "Implement", color: "sky", status: "running", task: "Add repository notifications" },
  { title: "Verify", color: "yellow", status: "waiting", task: "Review the pull request" },
  { title: "Done", color: "lime", status: "completed", task: "Connect the issue tracker" },
];

function WorkflowPalette({ empty = false }: { empty?: boolean }) {
  return (
    <div className="space-y-6 p-6">
      {(["vivid", "dim", "borders"] as const).map((view) => (
        <section key={view}>
          <h1 className="mb-3 text-sm font-semibold">
            {view === "dim" ? "Soft" : view === "borders" ? "Bordered" : "Vivid"}
          </h1>
          <div className="flex h-64">
            <WorkOrderKanbanBoard testId={`palette-${view}`}>
              {stages.map((stage) => {
                const status = getWorkOrderDisplayStatusMeta(stage.status);
                return (
                  <WorkOrderBoardLane
                    key={stage.title}
                    title={stage.title}
                    count={empty ? 0 : 1}
                    emptyDescription="Nothing here."
                    subheader={<p className="text-xs text-muted-foreground">{empty ? "0 tasks" : "1 task"}</p>}
                    {...lineBoardColumnLaneProps(stage.color, view)}
                  >
                    <article className="rounded-md border border-border bg-card p-3">
                      <p className="mb-3 text-sm font-medium">{stage.task}</p>
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs ${status.className}`}
                      >
                        <span className={`size-1.5 rounded-full ${status.dotClassName}`} />
                        {status.label}
                      </span>
                    </article>
                  </WorkOrderBoardLane>
                );
              })}
            </WorkOrderKanbanBoard>
          </div>
        </section>
      ))}
    </div>
  );
}

const meta = {
  title: "Factories/Design System/Workflow Palette",
  component: WorkflowPalette,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkflowPalette>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Light: Story = { globals: { theme: "light" } };
export const Dark: Story = { globals: { theme: "dark" } };
export const EmptyDark: Story = { globals: { theme: "dark" }, args: { empty: true } };
