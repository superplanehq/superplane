import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ReactNode } from "react";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { AutomationLookPreview } from "./AutomationLookPreview";

const meta = {
  title: "Factories/Pages/Task Split Run/Automation look",
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

export const Bar: Story = {
  name: "1 — Bar",
  render: () => <AutomationLookPreview look="bar" />,
};

export const Card: Story = {
  name: "2 — Card",
  render: () => <AutomationLookPreview look="card" />,
};

export const Rule: Story = {
  name: "3 — Rule",
  render: () => <AutomationLookPreview look="rule" />,
};

export const Compare: Story = {
  name: "Compare three looks",
  render: () => (
    <div className="grid gap-6 lg:grid-cols-3">
      <LookColumn title="Bar" note="Muted bar. Sans name. No caret.">
        <AutomationLookPreview look="bar" />
      </LookColumn>
      <LookColumn title="Card" note="Border, status stripe, log inside the card.">
        <AutomationLookPreview look="card" />
      </LookColumn>
      <LookColumn title="Rule" note="Section title and a hairline. Lightest change.">
        <AutomationLookPreview look="rule" />
      </LookColumn>
    </div>
  ),
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
