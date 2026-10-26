import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, fn, userEvent, within } from "storybook/test";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { implementationRuns, verificationRuns } from "@/ui/AutomationRunHistory/fixtures";
import { planningHistoryPreviewRuns } from "./planningHistoryPreview.fixtures";
import { AgentRunHistoryPreview } from "./AgentRunHistoryPreview";
import { IntakeRunHistoryPreview } from "./IntakeRunHistoryPreview";
import { intakeHistoryPreviewRuns } from "./intakeHistoryPreview.fixtures";
import type { IntakePreviewSource, IntakeRunOutcome } from "./intakeRunPreviewModel";

function IntakeHistoryStory({
  sourceId = "github-issues",
  view = "history",
  state = "ready",
}: {
  sourceId?: IntakePreviewSource | "planning" | "implement" | "verify";
  view?: "history" | IntakeRunOutcome;
  state?: "ready" | "loading" | "empty" | "error";
}) {
  const [loadError, setLoadError] = useState(state === "error");
  if (sourceId === "planning" || sourceId === "implement" || sourceId === "verify") {
    return (
      <ComponentStoryShell className="h-dvh bg-background">
        <AgentRunHistoryPreview
          title={
            sourceId === "planning" ? "Plans new tasks" : sourceId === "implement" ? "Implement" : "Address PR feedback"
          }
          runs={
            sourceId === "planning"
              ? planningHistoryPreviewRuns()
              : sourceId === "implement"
                ? implementationRuns
                : verificationRuns
          }
          onClose={fn()}
          onOpenSettings={fn()}
        />
      </ComponentStoryShell>
    );
  }
  const runs = intakeHistoryPreviewRuns(sourceId);
  return (
    <ComponentStoryShell className="h-dvh bg-background">
      <IntakeRunHistoryPreview
        sourceId={sourceId}
        runs={state === "empty" ? [] : runs}
        initialRunId={runs.find((run) => run.outcome === view)?.id}
        loading={state === "loading"}
        error={loadError ? "Intake history could not be loaded." : undefined}
        onRetry={() => setLoadError(false)}
        onOpenSettings={fn()}
        onClose={fn()}
      />
    </ComponentStoryShell>
  );
}

const meta = {
  title: "Factories/Pages/Automation run history",
  component: IntakeHistoryStory,
  decorators: [withFactoriesTheme],
  args: { sourceId: "github-issues", view: "history", state: "ready" },
  argTypes: {
    sourceId: { table: { disable: true } },
    view: {
      name: "Open screen",
      control: "select",
      options: ["history", "created", "filtered", "processing", "failed", "unknown", "cancelled"],
      description: "Open the list or a run with the selected outcome. Rows also open run details.",
    },
    state: {
      name: "History state",
      control: "select",
      options: ["ready", "loading", "empty", "error"],
      description: "Check loading, empty, and retry states on the history list.",
    },
  },
  render: (args) => <IntakeHistoryStory key={`${args.sourceId}:${args.view}:${args.state}`} {...args} />,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shared run history components with sample data for each intake, Planning, Implement, and Verify. Open rows for details. Use Controls for loading, empty, and error states. The app uses these components for automation run history.",
      },
    },
  },
} satisfies Meta<typeof IntakeHistoryStory>;
export default meta;
type Story = StoryObj<typeof meta>;

export const GitHub: Story = {
  name: "GitHub",
  argTypes: {
    view: { options: ["history", "created", "existing", "filtered", "processing", "failed", "unknown", "cancelled"] },
  },
};
export const Sentry: Story = { args: { sourceId: "sentry-exceptions" } };
export const Dependabot: Story = { args: { sourceId: "dependabot-alerts" } };
export const Linear: Story = { args: { sourceId: "linear-issues" } };
export const Jira: Story = { args: { sourceId: "jira-issues" } };
export const Productive: Story = { args: { sourceId: "productive-tasks" } };
export const Datadog: Story = { args: { sourceId: "datadog" } };
export const PagerDuty: Story = {
  name: "PagerDuty",
  args: { sourceId: "pagerduty-incidents" },
  argTypes: { view: { options: ["history", "created", "processing", "failed", "unknown", "cancelled"] } },
};
export const Scheduled: Story = {
  args: { sourceId: "scheduled-intake" },
  argTypes: { view: { options: ["history", "created", "no-proposal", "processing", "failed", "unknown"] } },
};
export const Planning: Story = {
  args: { sourceId: "planning" },
  argTypes: { view: { table: { disable: true } }, state: { table: { disable: true } } },
};
export const Implement: Story = {
  args: { sourceId: "implement" },
  argTypes: { view: { table: { disable: true } }, state: { table: { disable: true } } },
};
export const Verify: Story = {
  args: { sourceId: "verify" },
  argTypes: { view: { table: { disable: true } }, state: { table: { disable: true } } },
};
// Keep interaction coverage without adding another page to the QA navigation.
export const SearchBySource: Story = {
  tags: ["!dev"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole("textbox", { name: "Search task, source, or run ID" }), "#8712");
    await userEvent.click(canvas.getByRole("button", { name: "View superplane#8712" }));
    await expect(canvas.getByRole("region", { name: "Intake run details" })).toBeVisible();
    await expect(canvas.getByRole("tab", { name: "Event data", selected: true })).toBeVisible();
    await expect(canvas.getByRole("tabpanel")).toHaveTextContent("github.issue");
    await userEvent.click(canvas.getByRole("button", { name: "Run history" }));
    await expect(canvas.getByRole("table", { name: "Runs" })).toBeVisible();
    await expect(canvas.getByRole("textbox", { name: "Search task, source, or run ID" })).toHaveValue("#8712");
  },
};
