import type { Meta, StoryObj } from "@storybook/react-vite";

import { cn } from "@/lib/utils";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { REQUEST_CARD_CLASSNAME } from "./chatBubbleStyle";
import { splitRunIntakeSource } from "./splitRunSource";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

const github = splitRunIntakeSource("https://github.com/acme/payments-service/issues/842");
const jira = splitRunIntakeSource("https://acme.atlassian.net/browse/PAY-12");

const meta = {
  title: "Factories/Pages/Task Split Run/Source logo",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-background p-6 text-foreground">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const RequestBubble: Story = {
  name: "Request bubble",
  render: () => (
    <div className="flex w-full max-w-[420px] flex-col items-end gap-3">
      <div className={cn(REQUEST_CARD_CLASSNAME, "max-w-[85%]")} data-testid="source-logo-github-bubble">
        <div className="mb-2">
          <WorkOrderSplitRunSource source={github} compact />
        </div>
        <p>Show a clearer empty state</p>
      </div>
      <div className={cn(REQUEST_CARD_CLASSNAME, "max-w-[85%]")} data-testid="source-logo-jira-bubble">
        <div className="mb-2">
          <WorkOrderSplitRunSource source={jira} compact />
        </div>
        <p>Jira keeps its brand color</p>
      </div>
    </div>
  ),
};

export const CardSurface: Story = {
  name: "Card surface",
  render: () => (
    <div
      className="w-full max-w-[360px] rounded-lg border border-border bg-card p-4 text-card-foreground"
      data-testid="source-logo-card"
    >
      <WorkOrderSplitRunSource source={github} />
      <WorkOrderSplitRunSource source={jira} />
    </div>
  ),
};
