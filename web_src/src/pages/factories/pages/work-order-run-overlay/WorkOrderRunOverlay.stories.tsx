import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { WorkOrderRunOverlayPlayground } from "./WorkOrderRunOverlayPlayground";

/**
 * Work-order overlay as a run, not a ticket. Storybook-only concepts.
 *
 * Shared rules: no Factory Lines picker, no assignees/mission sidebar, no
 * ticket key as the hero. Checks stay as scorecards. One line per workspace.
 *
 * Mobbin MCP was not available in this environment. Patterns come from
 * GitHub Actions run graphs, Vercel/Railway deploy inspectors, and
 * n8n / SuperPlane canvas executions.
 */
const meta = {
  title: "Factories/Pages/Work Order Run Overlay",
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

/** Switch A / B / C on the same board. */
export const Compare: Story = {
  name: "Compare A B C",
  render: () => <WorkOrderRunOverlayPlayground />,
};

/** GitHub Actions / CircleCI: phase stepper, check strip, compact job graph. */
export const ConceptA: Story = {
  name: "A — Pipeline run",
  render: () => <WorkOrderRunOverlayPlayground initialConcept="a" />,
};

/** Vercel / Railway: left phase rail, right inspector for that step. */
export const ConceptB: Story = {
  name: "B — Phase inspector",
  render: () => <WorkOrderRunOverlayPlayground initialConcept="b" />,
};

/** n8n / SuperPlane canvas: React Flow body, checks as a score strip. */
export const ConceptC: Story = {
  name: "C — Live canvas",
  render: () => <WorkOrderRunOverlayPlayground initialConcept="c" />,
};
