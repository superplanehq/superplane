import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const meta = {
  title: "Factories/Pages/Task Split Run/Pull request review strip",
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

function PopupShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[420px] w-full max-w-[750px] flex-col overflow-hidden rounded-lg border border-border bg-background shadow-sm">
      <header className="flex shrink-0 items-center gap-3 border-b border-border px-5 py-3">
        <h2 className="min-w-0 flex-1 truncate text-[16px] font-semibold tracking-[-0.02em]">
          {OPEN_WORK_ORDER.title}
        </h2>
      </header>
      <div className="min-h-0 flex-1 px-5 py-4 text-[13px] text-muted-foreground">Automations log.</div>
      {children}
    </div>
  );
}

const footer = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER).footer;

export const Default: Story = {
  name: "Pull request is ready",
  render: () => (
    <PopupShell>
      {footer.note ? (
        <SplitRunAttentionNote note={footer.note} tone="waiting" actions={footer.actions} onAction={() => {}} />
      ) : null}
    </PopupShell>
  ),
};

export const ReadOnly: Story = {
  name: "Pull request is ready — no close actions",
  render: () => (
    <PopupShell>
      {footer.note ? <SplitRunAttentionNote note={footer.note} tone="waiting" actions={[]} /> : null}
    </PopupShell>
  ),
};
