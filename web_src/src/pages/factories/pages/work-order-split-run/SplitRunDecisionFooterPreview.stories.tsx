import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ReactNode } from "react";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { SplitRunDecisionFooterPreview, type DecisionFooterKind } from "./SplitRunDecisionFooterPreview";

const meta = {
  title: "Factories/Pages/Work Order Split Run/Decision footer",
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

const KINDS: { kind: DecisionFooterKind; label: string; note: string }[] = [
  {
    kind: "running",
    label: "Running",
    note: "No footer. Stop lives on the automation.",
  },
  {
    kind: "waiting",
    label: "Needs attention",
    note: "Reject and Approve in the note.",
  },
  {
    kind: "statusNote",
    label: "Status note + PR",
    note: "Custom note, Review PR, Reject, and Approve.",
  },
  {
    kind: "failed",
    label: "Step failed",
    note: "Debug, Reject, and Rerun.",
  },
  {
    kind: "draft",
    label: "Draft",
    note: "Start and Reject in the note.",
  },
  {
    kind: "completed",
    label: "Completed",
    note: "Result copy. No Reopen.",
  },
  {
    kind: "rejected",
    label: "Rejected",
    note: "Result copy. No Reopen.",
  },
  {
    kind: "closedFailed",
    label: "Closed as failed",
    note: "Reopen in the note.",
  },
];

export const Compare: Story = {
  name: "Compare states",
  render: () => (
    <div className="grid gap-6 lg:grid-cols-2 xl:grid-cols-3">
      {KINDS.map((entry) => (
        <LookColumn key={entry.kind} title={entry.label} note={entry.note}>
          <SplitRunDecisionFooterPreview kind={entry.kind} />
        </LookColumn>
      ))}
    </div>
  ),
};

export const Running: Story = {
  name: "Running — no footer",
  render: () => <SplitRunDecisionFooterPreview kind="running" />,
};

export const Waiting: Story = {
  name: "Needs attention",
  render: () => <SplitRunDecisionFooterPreview kind="waiting" />,
};

export const StatusNote: Story = {
  name: "Status note + PR",
  render: () => <SplitRunDecisionFooterPreview kind="statusNote" />,
};

export const Failed: Story = {
  name: "Step failed",
  render: () => <SplitRunDecisionFooterPreview kind="failed" />,
};

export const Draft: Story = {
  render: () => <SplitRunDecisionFooterPreview kind="draft" />,
};

export const Completed: Story = {
  render: () => <SplitRunDecisionFooterPreview kind="completed" />,
};

export const Rejected: Story = {
  render: () => <SplitRunDecisionFooterPreview kind="rejected" />,
};

export const ClosedFailed: Story = {
  name: "Closed as failed",
  render: () => <SplitRunDecisionFooterPreview kind="closedFailed" />,
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
