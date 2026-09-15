import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ReactNode } from "react";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { PlanStripPreview, PlanStripPreviewToggle, type PlanStripLook } from "./PlanStripPreview";

const meta = {
  title: "Factories/Pages/Task Split Run/Plan strip",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-muted/30 p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

const LOOKS: { look: PlanStripLook; label: string; note: string }[] = [
  {
    look: "today",
    label: "Today",
    note: "Two bars. Plan updated, then 2/5 with the why sentence, Archive, and Start.",
  },
  {
    look: "oneBar",
    label: "One bar",
    note: "Score bars, Plan updated, Show plan, Archive, and Start. No second 2/5. No why sentence.",
  },
  {
    look: "oneBarOpen",
    label: "One bar, plan open",
    note: "Score bars, Plan updated, and Hide plan. Start and Archive are not on this bar.",
  },
];

export const Compare: Story = {
  name: "Compare looks",
  render: () => (
    <div className="grid gap-6 xl:grid-cols-3">
      {LOOKS.map((entry) => (
        <LookColumn key={entry.look} title={entry.label} note={entry.note}>
          <PlanStripPreview look={entry.look} />
        </LookColumn>
      ))}
    </div>
  ),
};

export const Today: Story = {
  name: "Today — two bars",
  render: () => <PlanStripPreview look="today" />,
};

export const OneBar: Story = {
  name: "One bar",
  render: () => <PlanStripPreview look="oneBar" />,
};

export const OneBarOpen: Story = {
  name: "One bar, plan open",
  render: () => <PlanStripPreview look="oneBarOpen" />,
};

export const TryOneBar: Story = {
  name: "Try one bar",
  render: () => <PlanStripPreviewToggle />,
};

function LookColumn({ title, note, children }: { title: string; note: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-[13px] font-medium tracking-[-0.01em]">{title}</p>
      <p className="mb-3 text-[13px] text-muted-foreground">{note}</p>
      {children}
    </div>
  );
}
