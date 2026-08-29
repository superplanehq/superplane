import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { WorkOrderPopupRedesignPlayground } from "./WorkOrderPopupRedesignPlayground";
import { AGENT_WORK_POPUP_RUNNING } from "./workOrderPopupMocks";

/**
 * Work-order board popup as agent work, not a ticket.
 *
 * Storybook-only. The production peek dialog is unchanged.
 *
 * Shared rules: no right sidebar, no status, no author, no mission, no
 * factory lines, no comments. Owner, time, and cost sit in one top row.
 * The first screen is next step, scores, then the activity log.
 * Artifacts hang on the step that produced them. Markdown opens a
 * second popup.
 */
const meta = {
  title: "Factories/Pages/Task Popup Redesign",
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-background p-0">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Compare: Story = {
  name: "Compare Session Trace Job",
  render: () => <WorkOrderPopupRedesignPlayground />,
};

/** Devin-style feed: spec file, outputs, scores, then the agent log. */
export const Session: Story = {
  name: "Session — Agent feed",
  render: () => <WorkOrderPopupRedesignPlayground initialConcept="session" />,
};

/** LangSmith-style tree: input span, tool spans, evals. */
export const Trace: Story = {
  name: "Trace — Span tree",
  render: () => <WorkOrderPopupRedesignPlayground initialConcept="trace" />,
};

/** Deploy-details report: next step, scores, then the activity log. */
export const Job: Story = {
  name: "Job — Run report",
  render: () => <WorkOrderPopupRedesignPlayground initialConcept="job" />,
};

/** In-flight job: no scores until automations finish. The log stays visible. */
export const JobRunning: Story = {
  name: "Job — Running",
  render: () => <WorkOrderPopupRedesignPlayground initialConcept="job" fixture={AGENT_WORK_POPUP_RUNNING} />,
};
