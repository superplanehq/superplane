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
    look: "oneBar",
    label: "One bar",
    note: "Score bars, Plan updated, Show plan, Archive, and Start.",
  },
  {
    look: "oneBarOpen",
    label: "One bar, plan open",
    note: "Score bars, Hide plan, Archive, and Start. No Plan updated title.",
  },
  {
    look: "chips",
    label: "Decision strip, ready",
    note: "Verdict and the draft actions on top. Clarity, Confidence, and the Plan toggle below. Hover a score to read its summary.",
  },
  {
    look: "chipsOpen",
    label: "Decision strip, plan open",
    note: "The Plan toggle shows the open state. No unread dot while the plan is visible.",
  },
  {
    look: "stripBlocked",
    label: "Decision strip, blocked",
    note: "Clarity 2 blocks Start with a red verdict and one line of help. Start stays available.",
  },
  {
    look: "stripCaution",
    label: "Decision strip, caution",
    note: "Confidence 2 warns about agent fit. Clarity is fine.",
  },
  {
    look: "stripAnalyzing",
    label: "Decision strip, analyzing",
    note: "The verdict and both score slots show the matrix while the agent works.",
  },
];

export const Compare: Story = {
  name: "Compare looks",
  render: () => (
    <div className="grid gap-6 xl:grid-cols-2">
      {LOOKS.map((entry) => (
        <LookColumn key={entry.look} title={entry.label} note={entry.note}>
          <PlanStripPreview look={entry.look} />
        </LookColumn>
      ))}
    </div>
  ),
};

export const OneBar: Story = {
  name: "One bar",
  render: () => <PlanStripPreview look="oneBar" />,
};

export const OneBarOpen: Story = {
  name: "One bar, plan open",
  render: () => <PlanStripPreview look="oneBarOpen" />,
};

export const StripReady: Story = {
  name: "Decision strip, ready",
  render: () => <PlanStripPreview look="chips" />,
};

export const StripPlanOpen: Story = {
  name: "Decision strip, plan open",
  render: () => <PlanStripPreview look="chipsOpen" />,
};

export const StripBlocked: Story = {
  name: "Decision strip, blocked",
  render: () => <PlanStripPreview look="stripBlocked" />,
};

export const StripCaution: Story = {
  name: "Decision strip, caution",
  render: () => <PlanStripPreview look="stripCaution" />,
};

export const StripAnalyzing: Story = {
  name: "Decision strip, analyzing",
  render: () => <PlanStripPreview look="stripAnalyzing" />,
};

export const TryOneBar: Story = {
  name: "Try one bar",
  render: () => <PlanStripPreviewToggle look="oneBar" />,
};

export const TryStrip: Story = {
  name: "Try the decision strip",
  render: () => <PlanStripPreviewToggle look="chips" />,
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
